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

package roles

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	apiv1 "github.com/cloudnative-pg/cloudnative-pg/api/v1"
	"github.com/cloudnative-pg/cloudnative-pg/internal/management/utils"
	"github.com/cloudnative-pg/cloudnative-pg/pkg/management/log"
	"github.com/cloudnative-pg/cloudnative-pg/pkg/management/postgres"
)

// A RoleSynchronizer is a Kubernetes manager.Runnable
// that makes sure the Roles in the PostgreSQL databases are in sync with the spec
//
// c.f. https://pkg.go.dev/sigs.k8s.io/controller-runtime/pkg/manager#Runnable
type RoleSynchronizer struct {
	instance *postgres.Instance
	client   client.Client
}

// NewRoleSynchronizer creates a new RoleSynchronizer
func NewRoleSynchronizer(instance *postgres.Instance, client client.Client) *RoleSynchronizer {
	runner := &RoleSynchronizer{
		instance: instance,
		client:   client,
	}
	return runner
}

// Start starts running the RoleSynchronizer
func (sr *RoleSynchronizer) Start(ctx context.Context) error {
	contextLog := log.FromContext(ctx).WithName("RoleSynchronizer")
	contextLog.Info("starting up the runnable")
	isPrimary, err := sr.instance.IsPrimary()
	if err != nil {
		return err
	}
	if !isPrimary {
		contextLog.Info("skipping the role syncrhonization in replicas")
	}
	go func() {
		config := <-sr.instance.RoleSynchronizerChan()
		contextLog.Info("setting up role syncrhonizer loop")
		updateInterval := 1 * time.Minute // TODO: make configurable
		ticker := time.NewTicker(updateInterval)

		defer func() {
			ticker.Stop()
			contextLog.Info("Terminated RoleSynchronizer loop")
		}()

		for {
			select {
			case <-ctx.Done():
				return
			case config = <-sr.instance.RoleSynchronizerChan():
			case <-ticker.C:
			}

			// If the spec contains no roles to manage, stop the timer,
			// the process will resume through the wakeUp channel if necessary
			if config == nil || len(config.Roles) == 0 {
				ticker.Stop()
				// we set updateInterval to 0 to make sure the Ticker will be reset
				// if the feature is enabled again
				updateInterval = 0
				continue
			}

			// Update the ticker if the update interval has changed
			newUpdateInterval := updateInterval // TODO: make configurable
			if updateInterval != newUpdateInterval {
				ticker.Reset(newUpdateInterval)
				updateInterval = newUpdateInterval
			}

			err := sr.reconcile(ctx, config)
			if err != nil {
				contextLog.Error(err, "synchronizing roles", "config", config)
				continue
			}
		}
	}()
	<-ctx.Done()
	return nil
}

func (sr *RoleSynchronizer) reconcile(ctx context.Context, config *apiv1.ManagedConfiguration) error {
	var err error

	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("recovered from a panic: %s", r)
		}
	}()

	superUserDB, err := sr.instance.GetSuperUserDB()
	if err != nil {
		return fmt.Errorf("while reconciling managed roles: %w", err)
	}
	err = synchronizeRoles(
		ctx,
		NewPostgresRoleManager(superUserDB, sr.client, sr.instance),
		sr,
		config,
	)
	return err
}

// areEquivalent does a constrained check of two roles
// TODO: this needs to be completed, there is a ticket to track that
// we should see if DeepEquals will serve
func areEquivalent(role1, role2 apiv1.RoleConfiguration) bool {
	reduced := []struct {
		BypassRLS       bool
		CreateDB        bool
		CreateRole      bool
		Inherit         bool
		Replication     bool
		Superuser       bool
		Login           bool
		ConnectionLimit int64
	}{
		{
			CreateDB:        role1.CreateDB,
			CreateRole:      role1.CreateRole,
			Inherit:         role1.Inherit,
			Superuser:       role1.Superuser,
			Login:           role1.Login,
			BypassRLS:       role1.BypassRLS,
			Replication:     role1.Replication,
			ConnectionLimit: role1.ConnectionLimit,
		},
		{
			CreateDB:        role2.CreateDB,
			CreateRole:      role2.CreateRole,
			Inherit:         role2.Inherit,
			Superuser:       role2.Superuser,
			Login:           role2.Login,
			BypassRLS:       role2.BypassRLS,
			Replication:     role2.Replication,
			ConnectionLimit: role2.ConnectionLimit,
		},
	}
	return reduced[0] == reduced[1]
}

// synchronizeRoles aligns roles in the database to the spec
func synchronizeRoles(
	ctx context.Context,
	roleManager RoleManager,
	sr *RoleSynchronizer,
	config *apiv1.ManagedConfiguration,
) error {
	contextLog := log.FromContext(ctx).WithName("RoleSynchronizer")
	contextLog.Info("synchronizing roles",
		"podName", sr.instance.PodName,
		"managedConfig", config)

	wrapErr := func(err error) error {
		return fmt.Errorf("while synchronizing roles in primary: %w", err)
	}

	rolesInDB, err := roleManager.List(ctx, config)
	if err != nil {
		return wrapErr(err)
	}
	contextLog.Info("found roles in DB", "roles", rolesInDB)

	rolesInSpec := config.Roles
	// setup a map name -> role for the spec roles
	roleInSpecNamed := make(map[string]apiv1.RoleConfiguration)
	for _, r := range rolesInSpec {
		roleInSpecNamed[r.Name] = r
	}

	// 1. do any of the roles in the DB require update/delete?
	roleInDBNamed := make(map[string]apiv1.RoleConfiguration)
	for _, role := range rolesInDB {
		roleInDBNamed[role.Name] = role
		inSpec, found := roleInSpecNamed[role.Name]
		switch {
		case found && inSpec.Ensure == apiv1.EnsureAbsent:
			contextLog.Info("role in DB and Spec, but spec wants it absent. Deleting", "role", role.Name)
			err = roleManager.Delete(ctx, role)
			if err != nil {
				return wrapErr(err)
			}
		case found && (!areEquivalent(inSpec, role) || passwordNeedSync(ctx, sr, inSpec, *role.Password)):
			contextLog.Info("role in DB and Spec, are different. Updating", "role", role.Name)
			err = roleManager.Update(ctx, inSpec)
			if err != nil {
				return wrapErr(err)
			}
		case !found:
			contextLog.Debug("role in DB but not Spec. Ignoring it", "role", role.Name)
		}
	}

	// 2. create managed roles that are not in the DB
	for _, r := range rolesInSpec {
		_, found := roleInDBNamed[r.Name]
		if !found && r.Ensure == apiv1.EnsurePresent {
			contextLog.Info("role not in DB and spec wants it present. Creating", "role", r.Name)
			err = roleManager.Create(ctx, r)
			if err != nil {
				return wrapErr(err)
			}
		}
	}

	return nil
}

// roleInSpecPasswordChanged Check if the password stored in database is the same with password in external secrets
func passwordNeedSync(ctx context.Context, sr *RoleSynchronizer, role apiv1.RoleConfiguration, password string) bool {
	secretName := role.GetRoleSecretsName()
	if secretName == "" {
		return false
	}

	var secret corev1.Secret
	err := sr.client.Get(
		ctx,
		client.ObjectKey{Namespace: sr.instance.Namespace, Name: secretName},
		&secret)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return false
		}
		log.Error(err, "error while retrieving the role user password secret")
		return false
	}
	usernameFromSecret, passwordFromSecret, err := utils.GetUserPasswordFromSecret(&secret)
	if role.Name != usernameFromSecret {
		err := fmt.Errorf("wrong username '%v' in secret, expected '%v'", usernameFromSecret, role.Name)
		log.Error(err, "error while retrieving the role user password secret")
		return false
	}
	if err != nil {
		log.Error(err, "error while retrieving the role user password secret")
		return false
	}
	return password != passwordFromSecret
}
