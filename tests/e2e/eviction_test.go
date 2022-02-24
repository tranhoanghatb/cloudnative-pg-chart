/*
This file is part of Cloud Native PostgreSQL.

Copyright (C) 2019-2021 EnterpriseDB Corporation.
*/

package e2e

import (
	"fmt"
	"time"

	"github.com/avast/retry-go/v4"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/EnterpriseDB/cloud-native-postgresql/pkg/specs"
	"github.com/EnterpriseDB/cloud-native-postgresql/pkg/utils"
	"github.com/EnterpriseDB/cloud-native-postgresql/tests"
	testsUtils "github.com/EnterpriseDB/cloud-native-postgresql/tests/utils"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// This test evicts a CNP cluster's pod to simulate out of memory issues.
// Under this condition, the operator will immediately delete that evicted pod
// and a new pod will be created after that. We are using the status API to patch the pod
// status to phase=Failed, reason=Evict to simulate the eviction.
// There are several test cases:
// 1. Eviction of primary pod in a single instance cluster (using patch simulate) -- included
// 2. Eviction of standby pod in a multiple instance cluster (using patch simulate) -- included
// 3. Eviction of primary pod in a multiple instance cluster (using drain simulate) -- included
// Please note that, for manually testing, we can also use the pgbench to simulate the OOM, but given that the
// process of eviction is
// 1. the node is running out of memory
// 2. node started to evict pod, from the lowest priority first so BestEffort -> Burstable
// see: https://kubernetes.io/docs/concepts/scheduling-eviction/node-pressure-eviction/
// #pod-selection-for-kubelet-eviction
// we can not guarantee how much memory is left in node and when it will get OOM and start eviction,
// so we choose to use patch and drain to simulate the eviction. The patch status issued one problem,
// when evicting the primary pod of multiple clusters (see CNP-1851).

var _ = Describe("Pod eviction", Serial, Label(tests.LabelDisruptive), func() {
	const (
		level                    = tests.Low
		singleInstanceSampleFile = fixturesDir + "/eviction/single-instance-cluster.yaml"
		multiInstanceSampleFile  = fixturesDir + "/eviction/multi-instance-cluster.yaml"
	)

	evictPod := func(podName string, namespace string, env *testsUtils.TestingEnvironment, timeoutSeconds uint) error {
		var pod corev1.Pod
		err := env.Client.Get(env.Ctx,
			ctrlclient.ObjectKey{Namespace: namespace, Name: podName},
			&pod)
		if err != nil {
			return err
		}
		oldPodRV := pod.GetResourceVersion()
		GinkgoWriter.Printf("Old resource version for %v pod: %v \n", pod.GetName(), oldPodRV)
		patch := ctrlclient.MergeFrom(pod.DeepCopy())
		pod.Status.Phase = "Failed"
		pod.Status.Reason = "Evicted"
		// Patching the Pod status
		err = env.Client.Status().Patch(env.Ctx, &pod, patch)
		if err != nil {
			return fmt.Errorf("failed to patch status for Pod: %v", pod.Name)
		}

		// Checking the Pod is actually evicted and resource version changed
		err = retry.Do(
			func() error {
				err = env.Client.Get(env.Ctx,
					ctrlclient.ObjectKey{Namespace: namespace, Name: podName},
					&pod)
				if err != nil {
					return err
				}
				// Sometimes the eviction status is too short, we can not see if has been changed.
				// We checked the resource version here
				if oldPodRV != pod.GetResourceVersion() {
					GinkgoWriter.Printf("New resource version for %v pod: %v \n",
						pod.GetName(), pod.GetResourceVersion())
					return nil
				}
				return fmt.Errorf("pod %v has not been evicted", pod.Name)
			},
			retry.Delay(time.Second),
			retry.Attempts(timeoutSeconds),
		)
		return err
	}

	Context("Pod eviction in single instance cluster", Ordered, func() {
		var namespace string

		BeforeEach(func() {
			if testLevelEnv.Depth < int(level) {
				Skip("Test depth is lower than the amount requested for this test")
			}
		})
		JustAfterEach(func() {
			clusterName, err := env.GetResourceNameFromYAML(singleInstanceSampleFile)
			Expect(err).ToNot(HaveOccurred())
			if CurrentSpecReport().Failed() {
				env.DumpClusterEnv(namespace, clusterName,
					"out/"+CurrentSpecReport().LeafNodeText+".log")
			}
		})
		BeforeAll(func() {
			namespace = "single-instance-pod-eviction"
			err := env.CreateNamespace(namespace)
			Expect(err).ToNot(HaveOccurred())
			By("creating a cluster", func() {
				// Create a cluster in a namespace we'll delete after the test
				clusterName, err := env.GetResourceNameFromYAML(singleInstanceSampleFile)
				Expect(err).ToNot(HaveOccurred())
				AssertCreateCluster(namespace, clusterName, singleInstanceSampleFile, env)
			})
		})
		AfterAll(func() {
			err := env.DeleteNamespace(namespace)
			Expect(err).ToNot(HaveOccurred())
		})

		It("evicts the primary pod", func() {
			clusterName, err := env.GetResourceNameFromYAML(singleInstanceSampleFile)
			Expect(err).ToNot(HaveOccurred())
			podName := clusterName + "-1"
			err = evictPod(podName, namespace, env, 60)
			Expect(err).ToNot(HaveOccurred())

			By("waiting for the pod to be ready again", func() {
				namespacedName := types.NamespacedName{
					Namespace: namespace,
					Name:      podName,
				}
				Eventually(func() (bool, error) {
					pod := corev1.Pod{}
					err := env.Client.Get(env.Ctx, namespacedName, &pod)
					return utils.IsPodActive(pod) && utils.IsPodReady(pod), err
				}, 30).Should(BeTrue())
			})

			By("checking the cluster is healthy", func() {
				AssertClusterIsReady(namespace, clusterName, 30, env)
			})
		})
	})

	Context("Pod eviction in a multiple instance cluster", Ordered, func() {
		var (
			namespace       string
			taintNodeName   string
			needRemoveTaint bool
		)

		BeforeEach(func() {
			if testLevelEnv.Depth < int(level) {
				Skip("Test depth is lower than the amount requested for this test")
			}
		})
		JustAfterEach(func() {
			clusterName, err := env.GetResourceNameFromYAML(multiInstanceSampleFile)
			Expect(err).ToNot(HaveOccurred())
			if CurrentSpecReport().Failed() {
				env.DumpClusterEnv(namespace, clusterName,
					"out/"+CurrentSpecReport().LeafNodeText+".log")
			}
		})
		BeforeAll(func() {
			namespace = "multi-instance-pod-eviction"
			err := env.CreateNamespace(namespace)
			Expect(err).ToNot(HaveOccurred())
			By("Creating a cluster with multiple instances", func() {
				// Create a cluster in a namespace and shared in containers, we'll delete after the test
				clusterName, err := env.GetResourceNameFromYAML(multiInstanceSampleFile)
				Expect(err).ToNot(HaveOccurred())
				AssertCreateCluster(namespace, clusterName, multiInstanceSampleFile, env)
			})

			By("retrieving the nodeName for primary pod", func() {
				var primaryPod *corev1.Pod
				clusterName, err := env.GetResourceNameFromYAML(multiInstanceSampleFile)
				Expect(err).ToNot(HaveOccurred())
				primaryPod, err = env.GetClusterPrimary(namespace, clusterName)
				Expect(err).ToNot(HaveOccurred())
				taintNodeName = primaryPod.Spec.NodeName
			})
		})
		AfterAll(func() {
			err := env.DeleteNamespace(namespace)
			Expect(err).ToNot(HaveOccurred())

			if needRemoveTaint {
				By("cleaning the taint on node", func() {
					cmd := fmt.Sprintf("kubectl taint nodes %v node.kubernetes.io/memory-pressure:NoExecute-", taintNodeName)
					_, _, err := testsUtils.Run(cmd)
					Expect(err).ToNot(HaveOccurred())
				})
			}
		})

		It("evicts the replica pod", func() {
			var podName string

			clusterName, err := env.GetResourceNameFromYAML(multiInstanceSampleFile)
			Expect(err).ToNot(HaveOccurred())

			// Find the standby pod
			By("getting standby pod to evict", func() {
				podList, _ := env.GetClusterPodList(namespace, clusterName)
				Expect(len(podList.Items)).To(BeEquivalentTo(3))
				for _, pod := range podList.Items {
					// Avoid parting non ready nodes, non active nodes, or primary nodes
					if specs.IsPodStandby(pod) {
						podName = pod.Name
						break
					}
				}
				Expect(podName).ToNot(BeEmpty())
			})

			err = evictPod(podName, namespace, env, 60)
			Expect(err).ToNot(HaveOccurred())

			By("waiting for the replica to be ready again", func() {
				namespacedName := types.NamespacedName{
					Namespace: namespace,
					Name:      podName,
				}
				Eventually(func() (bool, error) {
					pod := corev1.Pod{}
					err := env.Client.Get(env.Ctx, namespacedName, &pod)
					return utils.IsPodActive(pod) && utils.IsPodReady(pod), err
				}, 30).Should(BeTrue())
			})

			By("checking the cluster is healthy", func() {
				AssertClusterIsReady(namespace, clusterName, 30, env)
			})
		})

		It("evicts the primary pod", func() {
			var primaryPod *corev1.Pod

			clusterName, err := env.GetResourceNameFromYAML(multiInstanceSampleFile)
			Expect(err).ToNot(HaveOccurred())
			primaryPod, err = env.GetClusterPrimary(namespace, clusterName)
			Expect(err).ToNot(HaveOccurred())

			// We can not use patch to simulate the eviction of a primary pod;
			// so that we use taint to simulate the real eviction

			By("tainting the node to make pod been evicted", func() {
				cmd := fmt.Sprintf("kubectl taint nodes %v node.kubernetes.io/memory-pressure:NoExecute", taintNodeName)
				_, _, err = testsUtils.Run(cmd)
				Expect(err).ToNot(HaveOccurred())
				needRemoveTaint = true
			})

			By("checking switchover happens", func() {
				Eventually(func() bool {
					podList, err := env.GetClusterPodList(namespace, clusterName)
					Expect(err).ToNot(HaveOccurred())
					for _, p := range podList.Items {
						if specs.IsPodPrimary(p) && primaryPod.GetName() != p.GetName() {
							return true
						}
					}
					return false
				}, 30).Should(BeTrue())
			})
			By("removing the taint on the node", func() {
				cmd := fmt.Sprintf("kubectl taint nodes %v node.kubernetes.io/memory-pressure:NoExecute-", taintNodeName)
				_, _, err = testsUtils.Run(cmd)
				Expect(err).ToNot(HaveOccurred())
				needRemoveTaint = false
			})

			// Pod need rejoin, need more time
			By("checking the cluster is healthy", func() {
				AssertClusterIsReady(namespace, clusterName, 120, env)
			})
		})
	})
})
