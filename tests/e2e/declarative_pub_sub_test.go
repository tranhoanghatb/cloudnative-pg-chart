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
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/types"

	apiv1 "github.com/cloudnative-pg/cloudnative-pg/api/v1"
	"github.com/cloudnative-pg/cloudnative-pg/tests"
	"github.com/cloudnative-pg/cloudnative-pg/tests/utils"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// - spinning up a cluster, apply a declarative publication/subscription on it

// Set of tests in which we use the declarative publication and subscription CRDs on an existing cluster
var _ = Describe("Declarative publication and subscription test", Label(tests.LabelSmoke, tests.LabelBasic,
	tests.LabelDeclarativePubSub), func() {
	const (
		sourceClusterManifest      = fixturesDir + "/declarative_pub_sub/source-cluster.yaml.template"
		destinationClusterManifest = fixturesDir + "/declarative_pub_sub/destination-cluster.yaml.template"
		databaseManifest           = fixturesDir + "/declarative_pub_sub/database.yaml.template"
		pubManifest                = fixturesDir + "/declarative_pub_sub/pub.yaml.template"
		subManifest                = fixturesDir + "/declarative_pub_sub/sub.yaml.template"
		level                      = tests.Medium
	)

	BeforeEach(func() {
		if testLevelEnv.Depth < int(level) {
			Skip("Test depth is lower than the amount requested for this test")
		}
	})

	Context("in a plain vanilla cluster", Ordered, func() {
		const (
			namespacePrefix = "declarative-pub-sub"
			dbname          = "declarative"
			pubName         = "declarative-pub"
			subName         = "declarative-sub"
		)
		var (
			sourceClusterName, destinationClusterName, namespace string
			databaseObjectName, pubObjectName, subObjectName     string
			database                                             *apiv1.Database
			pub                                                  *apiv1.Publication
			sub                                                  *apiv1.Subscription
			err                                                  error
		)

		BeforeAll(func() {
			// Create a cluster in a namespace we'll delete after the test
			namespace, err = env.CreateUniqueTestNamespace(namespacePrefix)
			Expect(err).ToNot(HaveOccurred())

			sourceClusterName, err = env.GetResourceNameFromYAML(sourceClusterManifest)
			Expect(err).ToNot(HaveOccurred())

			destinationClusterName, err = env.GetResourceNameFromYAML(destinationClusterManifest)
			Expect(err).ToNot(HaveOccurred())

			By("setting up source cluster", func() {
				AssertCreateCluster(namespace, sourceClusterName, sourceClusterManifest, env)
			})

			By("setting up destination cluster", func() {
				AssertCreateCluster(namespace, destinationClusterName, destinationClusterManifest, env)
			})
		})

		assertDatabaseExists := func(namespace, primaryPod, dbname string) {
			Eventually(func(g Gomega) {
				stdout, _, err := env.ExecQueryInInstancePod(
					utils.PodLocator{
						Namespace: namespace,
						PodName:   primaryPod,
					},
					"postgres",
					"\\l")
				g.Expect(err).ToNot(HaveOccurred())
				g.Expect(stdout).Should(ContainSubstring(dbname))
			}, 300).Should(Succeed())
		}

		assertPublicationExists := func(namespace, primaryPod string, pub *apiv1.Publication) {
			query := fmt.Sprintf("select count(*) from pg_publication where pubname = '%s'",
				pub.Spec.Name)
			Eventually(func(g Gomega) {
				stdout, _, err := env.ExecQueryInInstancePod(
					utils.PodLocator{
						Namespace: namespace,
						PodName:   primaryPod,
					},
					"postgres",
					query)
				g.Expect(err).ToNot(HaveOccurred())
				g.Expect(stdout).Should(ContainSubstring("1"), "expected publication not found")
			}, 30).Should(Succeed())
		}

		assertSubscriptionExists := func(namespace, primaryPod string, sub *apiv1.Subscription) {
			query := fmt.Sprintf("select count(*) from subscription where subname = '%s'",
				sub.Spec.Name)
			Eventually(func(g Gomega) {
				stdout, _, err := env.ExecQueryInInstancePod(
					utils.PodLocator{
						Namespace: namespace,
						PodName:   primaryPod,
					},
					"postgres",
					query)
				g.Expect(err).ToNot(HaveOccurred())
				g.Expect(stdout).Should(ContainSubstring("1"), "expected subscription not found")
			}, 30).Should(Succeed())
		}

		It("can add a declarative database", func() { //nolint:dupl
			By("applying Database CRD manifest", func() {
				CreateResourceFromFile(namespace, databaseManifest)
				databaseObjectName, err = env.GetResourceNameFromYAML(databaseManifest)
				Expect(err).NotTo(HaveOccurred())
			})
			By("ensuring the Database CRD succeeded reconciliation", func() {
				// get database object
				database = &apiv1.Database{}
				databaseNamespacedName := types.NamespacedName{
					Namespace: namespace,
					Name:      databaseObjectName,
				}

				Eventually(func(g Gomega) {
					err := env.Client.Get(env.Ctx, databaseNamespacedName, database)
					Expect(err).ToNot(HaveOccurred())
					g.Expect(database.Status.Ready).Should(BeTrue())
				}, 300).WithPolling(10 * time.Second).Should(Succeed())
			})

			By("verifying new database has been created", func() {
				primaryPodInfo, err := env.GetClusterPrimary(namespace, sourceClusterName)
				Expect(err).ToNot(HaveOccurred())

				assertDatabaseExists(namespace, primaryPodInfo.Name, dbname)
			})
		})
		It("can add a declarative publication", func() { //nolint:dupl
			By("applying Publication CRD manifest", func() {
				CreateResourceFromFile(namespace, pubManifest)
				pubObjectName, err = env.GetResourceNameFromYAML(pubManifest)
				Expect(err).NotTo(HaveOccurred())
			})
			By("ensuring the Publication CRD succeeded reconciliation", func() {
				// get publication object
				pub = &apiv1.Publication{}
				pubNamespacedName := types.NamespacedName{
					Namespace: namespace,
					Name:      pubObjectName,
				}

				Eventually(func(g Gomega) {
					err := env.Client.Get(env.Ctx, pubNamespacedName, pub)
					Expect(err).ToNot(HaveOccurred())
					g.Expect(pub.Status.Ready).Should(BeTrue())
				}, 300).WithPolling(10 * time.Second).Should(Succeed())
			})

			By("verifying new publication has been created", func() {
				primaryPodInfo, err := env.GetClusterPrimary(namespace, sourceClusterName)
				Expect(err).ToNot(HaveOccurred())

				assertPublicationExists(namespace, primaryPodInfo.Name, pub)
			})
		})
		It("can add a declarative subscription", func() { //nolint:dupl
			By("applying Subscription CRD manifest", func() {
				CreateResourceFromFile(namespace, subManifest)
				subObjectName, err = env.GetResourceNameFromYAML(subManifest)
				Expect(err).NotTo(HaveOccurred())
			})
			By("ensuring the Subscription CRD succeeded reconciliation", func() {
				// get subscription object
				sub = &apiv1.Subscription{}
				pubNamespacedName := types.NamespacedName{
					Namespace: namespace,
					Name:      subObjectName,
				}

				Eventually(func(g Gomega) {
					err := env.Client.Get(env.Ctx, pubNamespacedName, sub)
					Expect(err).ToNot(HaveOccurred())
					g.Expect(sub.Status.Ready).Should(BeTrue())
				}, 300).WithPolling(10 * time.Second).Should(Succeed())
			})

			By("verifying new subscription has been created", func() {
				primaryPodInfo, err := env.GetClusterPrimary(namespace, destinationClusterName)
				Expect(err).ToNot(HaveOccurred())

				assertSubscriptionExists(namespace, primaryPodInfo.Name, sub)
			})
		})
	})
})
