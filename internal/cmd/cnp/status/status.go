/*
This file is part of Cloud Native PostgreSQL.

Copyright (C) 2019-2021 EnterpriseDB Corporation.
*/

// Package status implement the kubectl-cnp status command
package status

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/cheynewallace/tabby"
	"github.com/logrusorgru/aurora/v3"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"

	apiv1 "github.com/EnterpriseDB/cloud-native-postgresql/api/v1"
	"github.com/EnterpriseDB/cloud-native-postgresql/internal/cmd/cnp"
	"github.com/EnterpriseDB/cloud-native-postgresql/pkg/management/log"
	"github.com/EnterpriseDB/cloud-native-postgresql/pkg/postgres"
	"github.com/EnterpriseDB/cloud-native-postgresql/pkg/specs"
	"github.com/EnterpriseDB/cloud-native-postgresql/pkg/utils"
)

// PostgresqlStatus contains the status of the Cluster and of all its instances
type PostgresqlStatus struct {
	// Cluster is the Cluster we are investigating
	Cluster *apiv1.Cluster `json:"cluster"`

	// InstanceStatus is the status of each instance, extracted directly
	// from the instance manager running into each Pod
	InstanceStatus *postgres.PostgresqlStatusList `json:"instanceStatus"`
}

// Status implement the "status" subcommand
func Status(ctx context.Context, clusterName string, verbose bool, format cnp.OutputFormat) error {
	status, err := ExtractPostgresqlStatus(ctx, clusterName)
	if err != nil {
		return err
	}

	err = cnp.Print(status, format)
	if err != nil {
		return err
	}

	if format != cnp.OutputFormatText {
		return nil
	}

	status.printBasicInfo()
	if verbose {
		err = status.printPostgresConfiguration()
		if err != nil {
			log.Log.Error(err, "Cannot extract configuration from cluster!")
		}
	}
	status.printInstancesStatus()

	return nil
}

// ExtractPostgresqlStatus get the PostgreSQL status using the Kubernetes API
func ExtractPostgresqlStatus(ctx context.Context, clusterName string) (*PostgresqlStatus, error) {
	// Get the Cluster object
	object, err := cnp.DynamicClient.Resource(apiv1.ClusterGVK).Namespace(
		cnp.Namespace).Get(ctx, clusterName, metav1.GetOptions{})
	if err != nil {
		log.Log.Error(err, "Cannot find PostgreSQL Cluster",
			"namespace", cnp.Namespace,
			"name", clusterName)
		return nil, err
	}

	var cluster apiv1.Cluster
	err = runtime.DefaultUnstructuredConverter.FromUnstructured(object.Object, &cluster)
	if err != nil {
		log.Log.Error(err, "Error decoding Cluster resource")
		return nil, err
	}

	// Get the list of Pods created by this Cluster
	var instancesStatus postgres.PostgresqlStatusList
	pods, err := cnp.GoClient.CoreV1().Pods(cnp.Namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		log.Log.Error(err, "Cannot find PostgreSQL Pods",
			"namespace", cnp.Namespace,
			"name", clusterName)
		return nil, err
	}

	var managedPods []corev1.Pod
	for idx := range pods.Items {
		for _, owner := range pods.Items[idx].ObjectMeta.OwnerReferences {
			if owner.Kind == apiv1.ClusterKind && owner.Name == clusterName {
				managedPods = append(managedPods, pods.Items[idx])
			}
		}
	}

	instancesStatus = extractInstancesStatus(
		ctx,
		cnp.Config,
		managedPods,
		specs.PostgresContainerName)

	// Extract the status from the instances
	status := PostgresqlStatus{
		Cluster:        &cluster,
		InstanceStatus: &instancesStatus,
	}
	return &status, nil
}

func (fullStatus *PostgresqlStatus) printBasicInfo() {
	cluster := fullStatus.Cluster

	primaryInstance := cluster.Status.CurrentPrimary
	if cluster.Status.CurrentPrimary != cluster.Status.TargetPrimary {
		primaryInstance = fmt.Sprintf("%v (switching to %v)",
			cluster.Status.CurrentPrimary, cluster.Status.TargetPrimary)
	}

	switch cluster.Status.Phase {
	case apiv1.PhaseHealthy, apiv1.PhaseFirstPrimary, apiv1.PhaseCreatingReplica:
		fmt.Println(aurora.Green(cluster.Status.Phase), " ", cluster.Status.PhaseReason)

	case apiv1.PhaseUpgrade, apiv1.PhaseWaitingForUser:
		fmt.Println(aurora.Yellow(cluster.Status.Phase), " ", cluster.Status.PhaseReason)

	default:
		fmt.Println(aurora.Red(cluster.Status.Phase), " ", cluster.Status.PhaseReason)
	}

	summary := tabby.New()
	summary.AddLine("Name:", cluster.Name)
	summary.AddLine("Namespace:", cluster.Namespace)
	summary.AddLine("PostgreSQL Image:", cluster.GetImageName())
	summary.AddLine("Primary instance:", primaryInstance)
	if cluster.Spec.Instances == cluster.Status.Instances {
		summary.AddLine("Instances:", aurora.Green(cluster.Spec.Instances))
	} else {
		summary.AddLine("Instances:", aurora.Red(cluster.Spec.Instances))
	}
	if cluster.Spec.Instances == cluster.Status.ReadyInstances {
		summary.AddLine("Ready instances:", aurora.Green(cluster.Status.ReadyInstances))
	} else {
		summary.AddLine("Ready instances:", aurora.Red(cluster.Status.ReadyInstances))
	}

	if cluster.Status.CurrentPrimary != cluster.Status.TargetPrimary {
		if cluster.Status.CurrentPrimary == "" {
			fmt.Println(aurora.Red("Primary server is initializing"))
		} else {
			fmt.Println(aurora.Red("Switchover in progress"))
		}
	}
	summary.Print()
	fmt.Println()
}

func (fullStatus *PostgresqlStatus) printPostgresConfiguration() error {
	// TODO this is not the real configuration. It must be removed or taken from the primary.
	configuration, err := fullStatus.Cluster.CreatePostgresqlConfiguration()
	if err != nil {
		return err
	}

	hba := fullStatus.Cluster.CreatePostgresqlHBA()

	fmt.Println(aurora.Green("PostgreSQL Configuration"))
	fmt.Println(configuration)
	fmt.Println()

	fmt.Println(aurora.Green("PostgreSQL HBA Rules"))
	fmt.Println(hba)
	fmt.Println()

	return nil
}

func (fullStatus *PostgresqlStatus) printInstancesStatus() {
	instanceStatus := fullStatus.InstanceStatus

	status := tabby.New()
	fmt.Println(aurora.Green("Instances status"))
	status.AddHeader(
		"Pod name",
		"Current LSN",
		"Received LSN",
		"Replay LSN",
		"System ID",
		"Primary",
		"Replicating",
		"Replay paused",
		"Pending restart",
		"Status")
	for _, instance := range instanceStatus.Items {
		if instance.ExecError != nil {
			status.AddLine(
				instance.PodName,
				"-",
				"-",
				"-",
				"-",
				"-",
				"-",
				"-",
				"-",
				instance.ExecError.Error())
		} else {
			status.AddLine(
				instance.PodName,
				instance.CurrentLsn,
				instance.ReceivedLsn,
				instance.ReplayLsn,
				instance.SystemID,
				boolToCheck(instance.IsPrimary),
				boolToCheck(instance.IsWalReceiverActive),
				boolToCheck(instance.ReplayPaused),
				boolToCheck(instance.PendingRestart),
				"OK")
		}
	}
	status.Print()
}

func boolToCheck(val bool) string {
	if val {
		return "\u2713"
	}
	return "\u2717"
}

func extractInstancesStatus(
	ctx context.Context,
	config *rest.Config,
	filteredPods []corev1.Pod,
	postgresContainerName string,
) postgres.PostgresqlStatusList {
	var result postgres.PostgresqlStatusList

	for idx := range filteredPods {
		instanceStatus := getReplicaStatusFromPodViaExec(
			ctx, config, filteredPods[idx], postgresContainerName)
		instanceStatus.IsReady = utils.IsPodReady(filteredPods[idx])
		result.Items = append(result.Items, instanceStatus)
	}

	return result
}

func getReplicaStatusFromPodViaExec(
	ctx context.Context,
	config *rest.Config,
	pod corev1.Pod,
	postgresContainerName string) postgres.PostgresqlStatus {
	result := postgres.PostgresqlStatus{
		PodName: pod.Name,
	}

	timeout := time.Second * 2
	clientInterface := kubernetes.NewForConfigOrDie(config)
	stdout, _, err := utils.ExecCommand(
		ctx,
		clientInterface,
		config,
		pod,
		postgresContainerName,
		&timeout,
		"/controller/manager", "instance", "status")

	if err != nil {
		result.PodName = pod.Name
		result.ExecError = err
		return result
	}

	err = json.Unmarshal([]byte(stdout), &result)
	if err != nil {
		result.PodName = pod.Name
		result.ExecError = err
		return result
	}

	return result
}
