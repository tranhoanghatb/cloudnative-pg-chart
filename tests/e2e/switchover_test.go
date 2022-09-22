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

package e2e

import (
	"github.com/cloudnative-pg/cloudnative-pg/tests"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Switchover", func() {
	const (
		namespace                     = "switchover-e2e"
		sampleFileWithoutReplSlots    = fixturesDir + "/base/cluster-storage-class.yaml.template"
		sampleFileWithReplSlotsEnable = fixturesDir + "/base/cluster-storage-class-with-repl-slots.yaml.template"
		clusterName                   = "postgresql-storage-class"
		level                         = tests.Medium
	)

	sampleFile := sampleFileWithReplSlotsEnable
	BeforeEach(func() {
		if testLevelEnv.Depth < int(level) {
			Skip("Test depth is lower than the amount requested for this test")
		}
	})
	JustAfterEach(func() {
		if CurrentSpecReport().Failed() {
			env.DumpNamespaceObjects(namespace, "out/"+CurrentSpecReport().LeafNodeText+".log")
		}
	})
	AfterEach(func() {
		err := env.DeleteNamespace(namespace)
		Expect(err).ToNot(HaveOccurred())
	})
	It("reacts to switchover requests", func() {
		// Create a cluster in a namespace we'll delete after the test
		err := env.CreateNamespace(namespace)
		Expect(err).ToNot(HaveOccurred())

		if env.PostgresVersion == 10 {
			// Cluster file without replication slot since it requires PostgreSQL 11 or above
			sampleFile = sampleFileWithoutReplSlots
		}
		AssertCreateCluster(namespace, clusterName, sampleFile, env)
		AssertSwitchover(namespace, clusterName, env)
		AssertPvcHasLabels(namespace, clusterName)
		AssertRepSlotsAreExistsAndAligned(namespace, clusterName)
	})
})
