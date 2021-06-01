/*
This file is part of Cloud Native PostgreSQL.

Copyright (C) 2019-2021 EnterpriseDB Corporation.
*/

package sequential

import (
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrlclient "sigs.k8s.io/controller-runtime/pkg/client"

	clusterapiv1 "github.com/EnterpriseDB/cloud-native-postgresql/api/v1"
	"github.com/EnterpriseDB/cloud-native-postgresql/tests"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

// Set of tests for config map for the operator. It is useful to configure the operator globally to survive
// the upgrades (especially in OLM installation like OpenShift).
var _ = Describe("ConfigMap support", func() {
	const clusterName = "configmap-support"
	const sampleFile = fixturesDir + "/configmap-support/configmap-support.yaml"
	const configMapFile = fixturesDir + "/configmap-support/configmap.yaml"
	const configMapName = "postgresql-operator-controller-manager-config"
	const namespace = "configmap-support-e2e"
	var operatorNamespace string
	var err error

	AssertReloadOperatorDeployment := func(operatorNamespace string, env *tests.TestingEnvironment) {
		By("reload the configmap by restarting the operator deployment", func() {
			operatorPod, err := env.GetOperatorPod()
			Expect(err).ToNot(HaveOccurred())

			// Restart operator deployment
			cmd := fmt.Sprintf("kubectl delete pod %v -n %v --force", operatorPod.Name, operatorNamespace)
			_, _, err = tests.Run(cmd)
			Expect(err).ToNot(HaveOccurred())

			// verify new operator pod is up and running
			// TODO write as an assert
			Eventually(env.IsOperatorReady, 120).Should(BeTrue(), "Operator pod is not ready")
		})
	}

	BeforeEach(func() {
		operatorDeployment, err := env.GetOperatorDeployment()
		Expect(err).ToNot(HaveOccurred())

		operatorNamespace = operatorDeployment.GetNamespace()
	})
	JustAfterEach(func() {
		if CurrentGinkgoTestDescription().Failed {
			env.DumpClusterEnv(namespace, clusterName,
				"out/"+CurrentGinkgoTestDescription().FullTestText+".log")
		}
	})
	AfterEach(func() {
		err = env.DeleteNamespace(namespace)
		Expect(err).ToNot(HaveOccurred())

		// Delete the configmap and restore the previous behaviour
		cmd := fmt.Sprintf("kubectl delete -n %v -f %v", operatorNamespace, configMapFile)
		_, _, err = tests.Run(cmd)
		Expect(err).ToNot(HaveOccurred())

		AssertReloadOperatorDeployment(operatorNamespace, env)
	})

	It("verify label's and annotation's inheritance support", func() {
		By("creating configmap", func() {
			// create a config map where operator is deployed
			cmd := fmt.Sprintf("kubectl apply -n %v -f %v", operatorNamespace, configMapFile)
			_, _, err = tests.Run(cmd)
			Expect(err).ToNot(HaveOccurred())
			// Check if configmap is created
			Eventually(func() ([]corev1.ConfigMap, error) {
				tempConfigMapList := &corev1.ConfigMapList{}
				err := env.Client.List(
					env.Ctx, tempConfigMapList, ctrlclient.InNamespace(operatorNamespace),
					ctrlclient.MatchingFields{"metadata.name": configMapName},
				)
				return tempConfigMapList.Items, err
			}, 60).Should(HaveLen(1))
		})

		AssertReloadOperatorDeployment(operatorNamespace, env)

		// Create the cluster namespace
		err = env.CreateNamespace(namespace)
		Expect(err).ToNot(HaveOccurred())
		AssertCreateCluster(namespace, clusterName, sampleFile, env)
		By("verify labels inherited on cluster and pods", func() {
			// Gathers the cluster list using labels
			clusterList := &clusterapiv1.ClusterList{}
			err = env.Client.List(env.Ctx,
				clusterList, ctrlclient.InNamespace(namespace),
				ctrlclient.MatchingLabels{
					"environment": "qaEnv",
				},
			)
			Expect(len(clusterList.Items)).Should(BeEquivalentTo(1),
				"label is not inherited on cluster")

			// Gathers the pod list using labels
			Eventually(func() int32 {
				podList := &corev1.PodList{}
				err = env.Client.List(
					env.Ctx, podList, ctrlclient.InNamespace(namespace),
					ctrlclient.MatchingLabels{
						"environment": "qaEnv",
					},
				)
				return int32(len(podList.Items))
			}, 180).Should(BeEquivalentTo(3), "label is not inherited on pod")
		})
		By("verify wildcard labels inherited", func() {
			// Gathers pod list using wildcard labels
			Eventually(func() int32 {
				podList := &corev1.PodList{}
				err = env.Client.List(
					env.Ctx, podList, ctrlclient.InNamespace(namespace),
					ctrlclient.MatchingLabels{
						"example.com/qa":   "qa",
						"example.com/prod": "prod",
					},
				)
				return int32(len(podList.Items))
			}, 60).Should(BeEquivalentTo(3),
				"wildcard labels are not inherited on pods")
		})
		By("verify annotations inherited on cluster and pods", func() {
			expectedAnnotationValue := "DatabaseApplication"
			// Gathers the cluster list using annotations
			cluster := &clusterapiv1.Cluster{}
			namespacedName := types.NamespacedName{
				Namespace: namespace,
				Name:      clusterName,
			}
			err = env.Client.Get(env.Ctx, namespacedName, cluster)
			Expect(err).ShouldNot(HaveOccurred())
			annotation := cluster.ObjectMeta.Annotations["categories"]
			Expect(annotation).ShouldNot(BeEmpty(),
				"annotation key is not inherited on cluster")
			Expect(annotation).Should(BeEquivalentTo(expectedAnnotationValue),
				"annotation value is not inherited on cluster")
			// Gathers the pod list using annotations
			podList, _ := env.GetClusterPodList(namespace, clusterName)
			for _, pod := range podList.Items {
				annotation = pod.ObjectMeta.Annotations["categories"]
				Expect(annotation).ShouldNot(BeEmpty(),
					fmt.Sprintf("annotation key is not inherited on pod %v", pod.ObjectMeta.Name))
				Expect(annotation).Should(BeEquivalentTo(expectedAnnotationValue),
					fmt.Sprintf("annotation value is not inherited on pod %v", pod.ObjectMeta.Name))
			}
		})
		By("verify wildcard annotation inherited", func() {
			// Gathers pod list using wildcard labels
			podList, _ := env.GetClusterPodList(namespace, clusterName)
			for _, pod := range podList.Items {
				wildcardAnnotationOne := pod.ObjectMeta.Annotations["example.com/qa"]
				wildcardAnnotationTwo := pod.ObjectMeta.Annotations["example.com/prod"]

				Expect(wildcardAnnotationOne).ShouldNot(BeEmpty(),
					fmt.Sprintf("wildcard annotaioon key %v is not inherited on pod %v", wildcardAnnotationOne,
						pod.ObjectMeta.Name))
				Expect(wildcardAnnotationTwo).ShouldNot(BeEmpty(),
					fmt.Sprintf("wildcard annotation key %v is not inherited on pod %v", wildcardAnnotationTwo,
						pod.ObjectMeta.Name))
				Expect(wildcardAnnotationOne).Should(BeEquivalentTo("qa"),
					fmt.Sprintf("wildcard annotation value %v is not inherited on pod %v", wildcardAnnotationOne,
						pod.ObjectMeta.Name))
				Expect(wildcardAnnotationTwo).Should(BeEquivalentTo("prod"),
					fmt.Sprintf("wildcard annotation value %v is not inherited on pod %v", wildcardAnnotationTwo,
						pod.ObjectMeta.Name))
			}
		})
	})
})
