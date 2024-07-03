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

package v1

import (
	"k8s.io/utils/ptr"

	"github.com/cloudnative-pg/cloudnative-pg/pkg/utils"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ensuring the correctness of synchronous replica data calculation", func() {
	It("should return all the non primary pods as electable", func() {
		cluster := createFakeCluster("example")
		number, names := cluster.GetSyncReplicasData()
		Expect(number).To(Equal(2))
		Expect(names).To(Equal([]string{"example-2", "example-3"}))
	})

	It("should return only the pod in the different AZ", func() {
		const (
			primaryPod     = "exampleAntiAffinity-1"
			sameZonePod    = "exampleAntiAffinity-2"
			differentAZPod = "exampleAntiAffinity-3"
		)

		cluster := createFakeCluster("exampleAntiAffinity")
		cluster.Spec.PostgresConfiguration.SyncReplicaElectionConstraint = SyncReplicaElectionConstraints{
			Enabled:                true,
			NodeLabelsAntiAffinity: []string{"az"},
		}
		cluster.Status.Topology = Topology{
			SuccessfullyExtracted: true,
			Instances: map[PodName]PodTopologyLabels{
				primaryPod: map[string]string{
					"az": "one",
				},
				sameZonePod: map[string]string{
					"az": "one",
				},
				differentAZPod: map[string]string{
					"az": "three",
				},
			},
		}

		number, names := cluster.GetSyncReplicasData()

		Expect(number).To(Equal(1))
		Expect(names).To(Equal([]string{differentAZPod}))
	})

	It("should lower the synchronous replica number to enforce self-healing when minSyncReplicas is not enforced", func() {
		cluster := createFakeCluster("exampleOnePod")
		cluster.Status = ClusterStatus{
			CurrentPrimary: "exampleOnePod-1",
			InstancesStatus: map[utils.PodStatus][]string{
				utils.PodHealthy: {"exampleOnePod-1"},
				utils.PodFailed:  {"exampleOnePod-2", "exampleOnePod-3"},
			},
		}
		number, names := cluster.GetSyncReplicasData()

		Expect(number).To(BeZero())
		Expect(names).To(BeEmpty())
		Expect(cluster.Spec.MinSyncReplicas).To(Equal(1))
	})

	It("should list all the replicas when minSyncReplicas is enforced", func() {
		cluster := createFakeCluster("exampleOnePod")
		cluster.Spec.PostgresConfiguration.MinSyncReplicasEnforcement = ptr.To(
			MinSyncReplicasEnforcementTypeRequired)
		cluster.Status = ClusterStatus{
			CurrentPrimary: "exampleOnePod-1",
			InstancesStatus: map[utils.PodStatus][]string{
				utils.PodHealthy: {"exampleOnePod-1"},
				utils.PodFailed:  {"exampleOnePod-2", "exampleOnePod-3"},
			},
			InstanceNames: []string{"exampleOnePod-1", "exampleOnePod-2", "exampleOnePod-3"},
		}
		number, names := cluster.GetSyncReplicasData()

		Expect(number).To(Equal(1))
		Expect(names).To(HaveLen(2))
		Expect(cluster.Spec.MinSyncReplicas).To(Equal(1))
	})

	It("should behave correctly if there is no ready host when minSyncReplicas is not enforced", func() {
		cluster := createFakeCluster("exampleNoPods")
		cluster.Status = ClusterStatus{
			CurrentPrimary: "example-1",
			InstancesStatus: map[utils.PodStatus][]string{
				utils.PodFailed: {"exampleNoPods-1", "exampleNoPods-2", "exampleNoPods-3"},
			},
		}
		number, names := cluster.GetSyncReplicasData()

		Expect(number).To(BeZero())
		Expect(names).To(BeEmpty())
	})

	It("should behave correctly if there is no ready host when minSyncReplicas is enforced", func() {
		cluster := createFakeCluster("exampleNoPods")
		cluster.Spec.PostgresConfiguration.MinSyncReplicasEnforcement = ptr.To(
			MinSyncReplicasEnforcementTypeRequired)
		cluster.Status = ClusterStatus{
			CurrentPrimary: "exampleNoPods-1",
			InstancesStatus: map[utils.PodStatus][]string{
				utils.PodFailed: {"exampleNoPods-1", "exampleNoPods-2", "exampleNoPods-3"},
			},
			InstanceNames: []string{"exampleNoPods-1", "exampleNoPods-2", "exampleNoPods-3"},
		}
		number, names := cluster.GetSyncReplicasData()

		Expect(number).To(Equal(1))
		Expect(names).To(HaveLen(2))
	})
})

var _ = Describe("should understand whether to consider non-ready replicas as synchronous candidates", func() {
	When("when MinSyncReplicasEnforcement is not set", func() {
		It("should retain the existing behavior", func() {
			cluster := createFakeCluster("example")
			Expect(cluster.Spec.PostgresConfiguration.IsMinSyncReplicasEnforcementRequired()).To(BeFalse())
		})
	})

	When("when MinSyncReplicasEnforcement is set to preferred", func() {
		It("should retain the existing behavior", func() {
			cluster := createFakeCluster("example")
			cluster.Spec.PostgresConfiguration.MinSyncReplicasEnforcement = ptr.To(MinSyncReplicasEnforcementTypePreferred)
			Expect(cluster.Spec.PostgresConfiguration.IsMinSyncReplicasEnforcementRequired()).To(BeFalse())
		})
	})

	When("when MinSyncReplicasEnforcement is set to required", func() {
		It("should enforce unready replicas as synchronous", func() {
			cluster := createFakeCluster("example")
			cluster.Spec.PostgresConfiguration.MinSyncReplicasEnforcement = ptr.To(MinSyncReplicasEnforcementTypeRequired)
			Expect(cluster.Spec.PostgresConfiguration.IsMinSyncReplicasEnforcementRequired()).To(BeTrue())
		})
	})
})
