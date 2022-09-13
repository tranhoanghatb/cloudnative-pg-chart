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

package reconciler

import (
	"context"
	"fmt"

	apiv1 "github.com/cloudnative-pg/cloudnative-pg/api/v1"
	"github.com/cloudnative-pg/cloudnative-pg/internal/management/controller/slots/infrastructure"
	"github.com/cloudnative-pg/cloudnative-pg/pkg/management/log"
)

// ReconcileReplicationSlots reconciles the replication slots of a given instance
func ReconcileReplicationSlots(
	ctx context.Context,
	instanceName string,
	manager infrastructure.Manager,
	cluster *apiv1.Cluster,
) error {
	if cluster.Spec.ReplicationSlots == nil ||
		cluster.Spec.ReplicationSlots.HighAvailability == nil {
		return nil
	}

	if cluster.Status.CurrentPrimary == instanceName || cluster.Status.TargetPrimary == instanceName {
		return reconcilePrimaryReplicationSlots(ctx, manager, cluster)
	}
	return nil
}

func reconcilePrimaryReplicationSlots(
	ctx context.Context,
	manager infrastructure.Manager,
	cluster *apiv1.Cluster,
) error {
	// if the replication slots feature was deactivated, ensure any existing
	// replication slots get cleaned up
	if !cluster.Spec.ReplicationSlots.HighAvailability.Enabled {
		return dropPrimaryReplicationSlots(ctx, cluster)
	}

	contextLogger := log.FromContext(ctx)
	contextLogger.Debug("Updating primary HA replication slots")

	currentSlots, err := manager.List(ctx, cluster.Spec.ReplicationSlots)
	if err != nil {
		return err
	}

	expectedSlots := make(map[string]bool)

	// Add every slot that is missing
	for _, instanceName := range cluster.Status.InstanceNames {
		if instanceName == cluster.Status.CurrentPrimary {
			continue
		}

		slotName := cluster.GetSlotNameFromInstanceName(instanceName)
		expectedSlots[slotName] = true

		if currentSlots.Has(slotName) {
			continue
		}

		// at this point, the cluster instance does not have a replication slot
		if err := manager.Create(ctx, infrastructure.ReplicationSlot{SlotName: slotName}); err != nil {
			return fmt.Errorf("updating primary HA replication slots: %w", err)
		}
	}

	contextLogger.Trace("Status of primary HA replication slots",
		"currentSlots", currentSlots,
		"expectedSlots", expectedSlots)

	// Delete any replication slots in the instance that is not from an existing cluster instance
	for _, slot := range currentSlots.Items {
		if !expectedSlots[slot.SlotName] {
			// Avoid deleting active slots.
			// It would trow an error on Postgres side.
			if slot.Active {
				contextLogger.Trace("Skipping deletion of replication slot because it is active",
					"slot", slot)
				continue
			}
			contextLogger.Trace("Attempt to delete replication slot",
				"slot", slot)
			if err := manager.Delete(ctx, slot); err != nil {
				return fmt.Errorf("failure deleting replication slot %q: %w", slot.SlotName, err)
			}
		}
	}

	return nil
}

func dropPrimaryReplicationSlots(ctx context.Context, cluster *apiv1.Cluster) error {
	contextLogger := log.FromContext(ctx)
	contextLogger.Debug("UNINPLEMENTED drop standby HA replication slots")
	// TODO: implement the logic to remove all the slots
	return nil
}
