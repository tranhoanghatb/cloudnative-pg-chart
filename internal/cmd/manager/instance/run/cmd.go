/*
Copyright The CloudNativePG Contributors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Package run implements the "instance run" subcommand of the operator
package run

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/cloudnative-pg/machinery/pkg/log"
	"github.com/spf13/cobra"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/runtime"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/util/retry"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/metrics/server"

	apiv1 "github.com/cloudnative-pg/cloudnative-pg/api/v1"
	"github.com/cloudnative-pg/cloudnative-pg/internal/cmd/manager/instance/run/lifecycle"
	"github.com/cloudnative-pg/cloudnative-pg/internal/management/controller"
	"github.com/cloudnative-pg/cloudnative-pg/internal/management/controller/externalservers"
	"github.com/cloudnative-pg/cloudnative-pg/internal/management/controller/roles"
	"github.com/cloudnative-pg/cloudnative-pg/internal/management/controller/slots/runner"
	"github.com/cloudnative-pg/cloudnative-pg/internal/management/controller/tablespaces"
	"github.com/cloudnative-pg/cloudnative-pg/internal/management/istio"
	"github.com/cloudnative-pg/cloudnative-pg/internal/management/linkerd"
	"github.com/cloudnative-pg/cloudnative-pg/pkg/concurrency"
	"github.com/cloudnative-pg/cloudnative-pg/pkg/management"
	"github.com/cloudnative-pg/cloudnative-pg/pkg/management/postgres"
	"github.com/cloudnative-pg/cloudnative-pg/pkg/management/postgres/logpipe"
	"github.com/cloudnative-pg/cloudnative-pg/pkg/management/postgres/webserver"
	"github.com/cloudnative-pg/cloudnative-pg/pkg/management/postgres/webserver/metricserver"
	pg "github.com/cloudnative-pg/cloudnative-pg/pkg/postgres"
	"github.com/cloudnative-pg/cloudnative-pg/pkg/versions"
)

var (
	scheme = runtime.NewScheme()

	// errNoFreeWALSpace is raised when there's not enough disk space
	// to store two WAL files
	errNoFreeWALSpace = fmt.Errorf("no free disk space for WALs")
)

func init() {
	_ = clientgoscheme.AddToScheme(scheme)
	_ = apiv1.AddToScheme(scheme)
}

// NewCmd creates the "instance run" subcommand
func NewCmd() *cobra.Command {
	var pgData string
	var podName string
	var clusterName string
	var namespace string
	var statusPortTLS bool
	var metricsPortTLS bool

	cmd := &cobra.Command{
		Use: "run [flags]",
		PreRunE: func(cmd *cobra.Command, _ []string) error {
			return management.WaitForGetCluster(cmd.Context(), client.ObjectKey{
				Name:      clusterName,
				Namespace: namespace,
			})
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			ctx := log.IntoContext(cmd.Context(), log.GetLogger())
			instance := postgres.NewInstance()

			instance.PgData = pgData
			instance.Namespace = namespace
			instance.PodName = podName
			instance.ClusterName = clusterName
			instance.StatusPortTLS = statusPortTLS
			instance.MetricsPortTLS = metricsPortTLS

			err := retry.OnError(retry.DefaultRetry, isRunSubCommandRetryable, func() error {
				return runSubCommand(ctx, instance)
			})

			if errors.Is(err, errNoFreeWALSpace) {
				os.Exit(apiv1.MissingWALDiskSpaceExitCode)
			}

			return err
		},
		PostRunE: func(cmd *cobra.Command, _ []string) error {
			if err := istio.TryInvokeQuitEndpoint(cmd.Context()); err != nil {
				return err
			}

			return linkerd.TryInvokeShutdownEndpoint(cmd.Context())
		},
	}

	cmd.Flags().StringVar(&pgData, "pg-data", os.Getenv("PGDATA"), "The PGDATA to be started up")
	cmd.Flags().StringVar(&podName, "pod-name", os.Getenv("POD_NAME"), "The name of this pod, to "+
		"be checked against the cluster state")
	cmd.Flags().StringVar(&clusterName, "cluster-name", os.Getenv("CLUSTER_NAME"), "The name of the "+
		"current cluster in k8s, used to coordinate switchover and failover")
	cmd.Flags().StringVar(&namespace, "namespace", os.Getenv("NAMESPACE"), "The namespace of "+
		"the cluster and of the Pod in k8s")
	cmd.Flags().BoolVar(&statusPortTLS, "status-port-tls", false,
		"Enable TLS for communicating with the operator")
	cmd.Flags().BoolVar(&metricsPortTLS, "metrics-port-tls", false,
		"Enable TLS for metrics scraping")
	return cmd
}

func runSubCommand(ctx context.Context, instance *postgres.Instance) error {
	var err error
	setupLog := log.WithName("setup")

	setupLog.Info("Starting CloudNativePG Instance Manager",
		"version", versions.Version,
		"build", versions.Info)

	setupLog.Info("Checking for free disk space for WALs before starting PostgreSQL")
	hasDiskSpaceForWals, err := instance.CheckHasDiskSpaceForWAL(ctx)
	if err != nil {
		setupLog.Error(err, "Error while checking if there is enough disk space for WALs, skipping")
	} else if !hasDiskSpaceForWals {
		setupLog.Info("Detected low-disk space condition, avoid starting the instance")
		return errNoFreeWALSpace
	}

	mgr, err := ctrl.NewManager(config.GetConfigOrDie(), ctrl.Options{
		Scheme: scheme,
		Cache: cache.Options{
			ByObject: map[client.Object]cache.ByObject{
				&apiv1.Cluster{}: {
					Field: fields.OneTermEqualSelector("metadata.name", instance.ClusterName),
					Namespaces: map[string]cache.Config{
						instance.Namespace: {},
					},
				},
			},
		},
		// We don't need a cache for secrets and configmap, as all reloads
		// should be driven by changes in the Cluster we are watching
		Client: client.Options{
			Cache: &client.CacheOptions{
				DisableFor: []client.Object{
					&corev1.Secret{},
					&corev1.ConfigMap{},
					// we don't have the permissions to cache backups, as the ServiceAccount
					// doesn't have watch permission on the backup status
					&apiv1.Backup{},
				},
			},
		},
		Metrics: server.Options{
			BindAddress: "0", // TODO: merge metrics to the manager one
		},
	})
	if err != nil {
		setupLog.Error(err, "unable to set up overall controller manager")
		return err
	}

	postgresStartConditions := concurrency.MultipleExecuted{}
	exitedConditions := concurrency.MultipleExecuted{}

	metricsExporter := metricserver.NewExporter(instance)
	reconciler := controller.NewInstanceReconciler(instance, mgr.GetClient(), metricsExporter)
	err = ctrl.NewControllerManagedBy(mgr).
		For(&apiv1.Cluster{}).
		Complete(reconciler)
	if err != nil {
		setupLog.Error(err, "unable to create controller")
		return err
	}
	postgresStartConditions = append(postgresStartConditions, reconciler.GetExecutedCondition())

	// postgres CSV logs handler (PGAudit too)
	postgresLogPipe := logpipe.NewLogPipe()
	if err := mgr.Add(postgresLogPipe); err != nil {
		return err
	}
	postgresStartConditions = append(postgresStartConditions, postgresLogPipe.GetInitializedCondition())
	exitedConditions = append(exitedConditions, postgresLogPipe.GetExitedCondition())

	// raw logs handler
	rawPipe := logpipe.NewRawLineLogPipe(filepath.Join(pg.LogPath, pg.LogFileName),
		logpipe.LoggingCollectorRecordName)
	if err := mgr.Add(rawPipe); err != nil {
		return err
	}
	postgresStartConditions = append(postgresStartConditions, rawPipe.GetExecutedCondition())
	exitedConditions = append(exitedConditions, rawPipe.GetExitedCondition())

	// json logs handler
	jsonPipe := logpipe.NewJSONLineLogPipe(filepath.Join(pg.LogPath, pg.LogFileName+".json"))
	if err := mgr.Add(jsonPipe); err != nil {
		return err
	}
	postgresStartConditions = append(postgresStartConditions, jsonPipe.GetExecutedCondition())
	exitedConditions = append(exitedConditions, jsonPipe.GetExitedCondition())

	if err := reconciler.ReconcileWalStorage(ctx); err != nil {
		return err
	}

	postgresLifecycleManager := lifecycle.NewPostgres(ctx, instance, postgresStartConditions)
	if err = mgr.Add(postgresLifecycleManager); err != nil {
		setupLog.Error(err, "unable to create instance runnable")
		return err
	}

	if err = mgr.Add(lifecycle.NewPostgresOrphansReaper(instance)); err != nil {
		setupLog.Error(err, "unable to create zombie reaper")
		return err
	}

	slotReplicator := runner.NewReplicator(instance)
	if err = mgr.Add(slotReplicator); err != nil {
		setupLog.Error(err, "unable to create slot replicator")
		return err
	}

	roleSynchronizer := roles.NewRoleSynchronizer(instance, reconciler.GetClient())
	if err = mgr.Add(roleSynchronizer); err != nil {
		setupLog.Error(err, "unable to create role synchronizer")
		return err
	}

	// onlineUpgradeCtx is a child context of the postgres context.
	// onlineUpgradeCtx will be the context passed to all the manager handled Runnables via Start(ctx),
	// its deletion will imply all Runnables to stop, but will be handled
	// appropriately by the Postgres Lifecycle Manager, which won't terminate Postgres in this case.
	// The parent GlobalContext will only be deleted by the Postgres Lifecycle Manager itself when required,
	// which will imply the deletion of the child onlineUpgradeCtx too, again, terminating all the Runnables.
	onlineUpgradeCtx, onlineUpgradeCancelFunc := context.WithCancel(postgresLifecycleManager.GetGlobalContext())
	defer onlineUpgradeCancelFunc()
	remoteSrv, err := webserver.NewRemoteWebServer(instance, onlineUpgradeCancelFunc, exitedConditions)
	if err != nil {
		return err
	}
	if err = mgr.Add(remoteSrv); err != nil {
		setupLog.Error(err, "unable to add remote webserver runnable")
		return err
	}

	localSrv, err := webserver.NewLocalWebServer(
		instance,
		mgr.GetClient(),
		mgr.GetEventRecorderFor("local-webserver"),
	)
	if err != nil {
		return err
	}
	if err = mgr.Add(localSrv); err != nil {
		setupLog.Error(err, "unable to add local webserver runnable")
		return err
	}

	metricsServer, err := metricserver.New(instance, metricsExporter)
	if err != nil {
		return err
	}
	if err = mgr.Add(metricsServer); err != nil {
		setupLog.Error(err, "unable to add local webserver runnable")
		return err
	}

	setupLog.Info("starting tablespace manager")
	if err := tablespaces.NewTablespaceReconciler(instance, mgr.GetClient()).
		SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create tablespace reconciler")
		return err
	}

	setupLog.Info("starting external server manager")
	if err := externalservers.NewReconciler(instance, mgr.GetClient()).
		SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create external servers reconciler")
		return err
	}

	setupLog.Info("starting controller-runtime manager")
	if err := mgr.Start(onlineUpgradeCtx); err != nil {
		setupLog.Error(err, "unable to run controller-runtime manager")
		return makeUnretryableError(err)
	}

	setupLog.Info("Checking for free disk space for WALs after PostgreSQL finished")
	hasDiskSpaceForWals, err = instance.CheckHasDiskSpaceForWAL(ctx)
	if err != nil {
		setupLog.Error(err, "Error while checking if there is enough disk space for WALs, skipping")
	} else if !hasDiskSpaceForWals {
		setupLog.Info("Detected low-disk space condition")
		return errNoFreeWALSpace
	}

	return nil
}
