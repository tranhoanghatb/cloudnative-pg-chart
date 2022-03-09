/*
This file is part of Cloud Native PostgreSQL.

Copyright (C) 2019-2021 EnterpriseDB Corporation.
*/

package lifecycle

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/EnterpriseDB/cloud-native-postgresql/pkg/concurrency"
	"github.com/EnterpriseDB/cloud-native-postgresql/pkg/management/log"
	"github.com/EnterpriseDB/cloud-native-postgresql/pkg/management/postgres"
)

// PostgresLifecycle implements the manager.Runnable interface for a postgres.Instance
type PostgresLifecycle struct {
	instance *postgres.Instance

	ctx                  context.Context
	cancel               context.CancelFunc
	systemInitialization *concurrency.Executed
}

// NewPostgres creates a new PostgresLifecycle
func NewPostgres(
	ctx context.Context,
	instance *postgres.Instance,
	initialization *concurrency.Executed,
) *PostgresLifecycle {
	childCtx, cancel := context.WithCancel(ctx)
	return &PostgresLifecycle{
		instance:             instance,
		ctx:                  childCtx,
		cancel:               cancel,
		systemInitialization: initialization,
	}
}

// GetContext returns the PostgresLifecycle's context
func (i *PostgresLifecycle) GetContext() context.Context {
	return i.ctx
}

// Start starts running the PostgresLifecycle
// nolint:gocognit
func (i *PostgresLifecycle) Start(ctx context.Context) error {
	contextLog := log.FromContext(ctx)
	signals := make(chan os.Signal, 1)
	signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)

	// Ensure that at the end of this runnable the instance
	// manager will shut down
	defer i.cancel()

	// Every cycle correspond to the lifespan of a postmaster process
	for {
		// Start the postmaster. The postMasterErrChan channel
		// will contain any error returned by the process.
		postMasterErrChan := i.runPostgresAndWait(ctx)

		select {
		case err := <-postMasterErrChan:
			// We didn't request postmaster to shut down or to restart, nevertheless
			// the postmaster is terminated. This can happen in the following
			// conditions:
			//
			// 1 - postmaster has crashed
			// 2 - a postmaster child has crashed, and postmaster decided to fly away
			//
			// In this case we want to terminate the instance manager and let the Kubelet
			// restart the Pod.
			if err != nil {
				var exitError *exec.ExitError
				if !errors.As(err, &exitError) {
					contextLog.Error(err, "Error waiting on PostgreSQL process")
				} else {
					contextLog.Error(exitError, "PostgreSQL process exited with errors")
				}
			}
			return err

		case <-ctx.Done():
			// The controller manager asked us to terminate our operations.
			// We shut down PostgreSQL and terminate using the maximum available
			// stop delay. We are doing that because we are not going to receive
			// a SIGKILL by the Kubelet, which is not informed about what's
			// happening.
			log.Info("Context has been cancelled, shutting down and exiting")
			fullShutDownSequence(i.instance, tryShuttingDownSmartFast(i.instance.MaxStopDelay))
			return nil

		case sig := <-signals:
			// The kubelet is asking us to terminate by sending a signal
			// to our process. In this case we terminate as fast as we can,
			// otherwise we'll receive a SIGKILL by the Kubelet, possibly
			// resulting in a data corruption.
			//
			// This is why we are trying a smart shutdown for half-time
			// of our stop delay, and then we proceed.
			log.Info("Received termination signal", "signal", sig)
			fullShutDownSequence(i.instance, tryShuttingDownSmartFast(i.instance.MaxStopDelay/2))
			return nil

		case req := <-i.instance.GetInstanceCommandChan():
			// We received a command issued by the reconciliation loop of
			// the instance manager.
			log.Info("Received request for postgres", "req", req)

			// We execute the requested operation
			restartNeeded, err := i.handleInstanceCommandRequests(req)
			if err != nil {
				log.Error(err, "while handling instance command request")
			}
			if !restartNeeded {
				return nil
			}
		}

		// Here the postmaster is terminated. We need to start a new postmaster
		// process
	}
}

// handleInstanceCommandRequests execute a command requested by the reconciliation
// loop.
func (i *PostgresLifecycle) handleInstanceCommandRequests(
	req postgres.InstanceCommand,
) (restartNeeded bool, err error) {
	switch req {
	case postgres.RestartSmartFast:
		return true, tryShuttingDownSmartFast(i.instance.MaxSwitchoverDelay)(i.instance)
	case postgres.ShutDownFastImmediate:
		fullShutDownSequence(i.instance, tryShuttingDownFastImmediate(i.instance.MaxStopDelay))
		return false, nil
	default:
		return false, fmt.Errorf("unrecognized request: %s", req)
	}
}
