/*
This file is part of Cloud Native PostgreSQL.

Copyright (C) 2019-2021 EnterpriseDB Corporation.
*/

// Package specs contains the specification of the K8s resources
// generated by the Cloud Native PostgreSQL operator
package specs

import (
	"fmt"
	"strconv"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	apiv1 "github.com/EnterpriseDB/cloud-native-postgresql/api/v1"
	"github.com/EnterpriseDB/cloud-native-postgresql/pkg/management/url"
	"github.com/EnterpriseDB/cloud-native-postgresql/pkg/postgres"
	"github.com/EnterpriseDB/cloud-native-postgresql/pkg/utils"
)

const (
	// MetadataNamespace is the annotation and label namespace used by the operator
	MetadataNamespace = "k8s.enterprisedb.io"

	// ClusterSerialAnnotationName is the name of the annotation containing the
	// serial number of the node
	ClusterSerialAnnotationName = MetadataNamespace + "/nodeSerial"

	// ClusterRestartAnnotationName is the name of the annotation containing the
	// latest required restart time
	ClusterRestartAnnotationName = "kubectl.kubernetes.io/restartedAt"

	// ClusterReloadAnnotationName is the name of the annotation containing the
	// latest required restart time
	ClusterReloadAnnotationName = MetadataNamespace + "/reloadedAt"

	// ClusterRoleLabelName label is applied to Pods to mark primary ones
	ClusterRoleLabelName = "role"

	// ClusterRoleLabelPrimary is written in labels to represent primary servers
	ClusterRoleLabelPrimary = "primary"

	// ClusterRoleLabelReplica is written in labels to represent replica servers
	ClusterRoleLabelReplica = "replica"

	// WatchedLabelName label is for Secrets or ConfigMaps that needs to be reloaded
	WatchedLabelName = MetadataNamespace + "/reload"

	// ClusterLabelName label is applied to Pods to link them to the owning
	// cluster.
	//
	// Deprecated.
	//
	// utils.ClusterLabelName should be used instead where possible.
	ClusterLabelName = "postgresql"

	// PostgresContainerName is the name of the container executing PostgreSQL
	// inside one Pod
	PostgresContainerName = "postgres"

	// BootstrapControllerContainerName is the name of the container copying the bootstrap
	// controller inside the Pod file system
	BootstrapControllerContainerName = "bootstrap-controller"

	// PgDataPath is the path to PGDATA variable
	PgDataPath = "/var/lib/postgresql/data/pgdata"

	// PgWalPath is the path to the pg_wal directory
	PgWalPath = PgDataPath + "/pg_wal"

	// PgWalArchiveStatusPath is the path to the archive status directory
	PgWalArchiveStatusPath = PgWalPath + "/archive_status"
)

func createEnvVarPostgresContainer(cluster apiv1.Cluster, podName string) []corev1.EnvVar {
	envVar := []corev1.EnvVar{
		{
			Name:  "PGDATA",
			Value: PgDataPath,
		},
		{
			Name:  "POD_NAME",
			Value: podName,
		},
		{
			Name:  "NAMESPACE",
			Value: cluster.Namespace,
		},
		{
			Name:  "CLUSTER_NAME",
			Value: cluster.Name,
		},
		{
			Name:  "PGPORT",
			Value: strconv.Itoa(postgres.ServerPort),
		},
		{
			Name:  "PGHOST",
			Value: postgres.SocketDirectory,
		},
	}

	return envVar
}

// createPostgresContainers create the PostgreSQL containers that are
// used for every instance
func createPostgresContainers(
	cluster apiv1.Cluster,
	podName string,
) []corev1.Container {
	containers := []corev1.Container{
		{
			Name:            PostgresContainerName,
			Image:           cluster.GetImageName(),
			ImagePullPolicy: cluster.Spec.ImagePullPolicy,
			Env:             createEnvVarPostgresContainer(cluster, podName),
			VolumeMounts:    createPostgresVolumeMounts(cluster),
			ReadinessProbe: &corev1.Probe{
				TimeoutSeconds: 5,
				Handler: corev1.Handler{
					HTTPGet: &corev1.HTTPGetAction{
						Path: url.PathReady,
						Port: intstr.FromInt(url.StatusPort),
					},
				},
			},
			// From K8s 1.17 and newer, startup probes will be available for
			// all users and not just protected from feature gates. For now
			// let's use the LivenessProbe. When we will drop support for K8s
			// 1.16, we'll configure a StartupProbe and this will lead to a
			// better LivenessProbe (without InitialDelaySeconds).
			LivenessProbe: &corev1.Probe{
				InitialDelaySeconds: cluster.GetMaxStartDelay(),
				TimeoutSeconds:      5,
				Handler: corev1.Handler{
					HTTPGet: &corev1.HTTPGetAction{
						Path: url.PathHealth,
						Port: intstr.FromInt(url.StatusPort),
					},
				},
			},
			Command: []string{
				"/controller/manager",
				"instance",
				"run",
			},
			Resources: cluster.Spec.Resources,
			Ports: []corev1.ContainerPort{
				{
					Name:          "postgresql",
					ContainerPort: postgres.ServerPort,
					Protocol:      "TCP",
				},
				{
					Name:          "metrics",
					ContainerPort: int32(url.PostgresMetricsPort),
					Protocol:      "TCP",
				},
				{
					Name:          "status",
					ContainerPort: int32(url.StatusPort),
					Protocol:      "TCP",
				},
			},
			SecurityContext: CreateContainerSecurityContext(),
		},
	}

	addManagerLoggingOptions(cluster, &containers[0])

	if cluster.Spec.Backup.IsBarmanEndpointCASet() {
		containers[0].Env = append(containers[0].Env, corev1.EnvVar{
			Name:  "AWS_CA_BUNDLE",
			Value: postgres.BarmanEndpointCACertificateLocation,
		})
	}

	return containers
}

// CreateAffinitySection creates the affinity sections for Pods, given the configuration
// from the user
func CreateAffinitySection(clusterName string, config apiv1.AffinityConfiguration) *corev1.Affinity {
	// Initialize affinity
	affinity := CreateGeneratedAntiAffinity(clusterName, config)

	if config.AdditionalPodAffinity == nil &&
		config.AdditionalPodAntiAffinity == nil {
		return affinity
	}

	if affinity == nil {
		affinity = &corev1.Affinity{}
	}

	if config.AdditionalPodAffinity != nil {
		affinity.PodAffinity = config.AdditionalPodAffinity
	}

	if config.AdditionalPodAntiAffinity != nil {
		if affinity.PodAntiAffinity == nil {
			affinity.PodAntiAffinity = &corev1.PodAntiAffinity{}
		}
		affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution = append(
			affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution,
			config.AdditionalPodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution...)
		affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution = append(
			affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution,
			config.AdditionalPodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution...)
	}
	return affinity
}

// CreateGeneratedAntiAffinity generates the affinity terms the operator is in charge for if enabled,
// return nil if disabled or an error occurred, as invalid values should be validated before this method is called
func CreateGeneratedAntiAffinity(clusterName string, config apiv1.AffinityConfiguration) *corev1.Affinity {
	// We have no anti affinity section if the user don't have it configured
	if config.EnablePodAntiAffinity != nil && !(*config.EnablePodAntiAffinity) {
		return nil
	}
	affinity := &corev1.Affinity{PodAntiAffinity: &corev1.PodAntiAffinity{}}
	topologyKey := config.TopologyKey
	if len(topologyKey) == 0 {
		topologyKey = "kubernetes.io/hostname"
	}

	podAffinityTerm := corev1.PodAffinityTerm{
		LabelSelector: &metav1.LabelSelector{
			MatchExpressions: []metav1.LabelSelectorRequirement{
				{
					Key:      ClusterLabelName,
					Operator: metav1.LabelSelectorOpIn,
					Values: []string{
						clusterName,
					},
				},
			},
		},
		TopologyKey: topologyKey,
	}

	// Switch pod anti-affinity type:
	// - if it is "required", 'RequiredDuringSchedulingIgnoredDuringExecution' will be properly set.
	// - if it is "preferred",'PreferredDuringSchedulingIgnoredDuringExecution' will be properly set.
	// - by default, return nil.
	switch config.PodAntiAffinityType {
	case apiv1.PodAntiAffinityTypeRequired:
		affinity.PodAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution =
			[]corev1.PodAffinityTerm{podAffinityTerm}
	case apiv1.PodAntiAffinityTypePreferred:
		affinity.PodAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution =
			[]corev1.WeightedPodAffinityTerm{
				{
					Weight:          100,
					PodAffinityTerm: podAffinityTerm,
				},
			}
	default:
		return nil
	}
	return affinity
}

// CreatePostgresSecurityContext defines the security context under which
// the PostgreSQL containers are running
func CreatePostgresSecurityContext(postgresUser, postgresGroup int64) *corev1.PodSecurityContext {
	// Under Openshift we inherit SecurityContext from the restricted security context constraint
	if utils.HaveSecurityContextConstraints() {
		return nil
	}

	trueValue := true
	return &corev1.PodSecurityContext{
		RunAsNonRoot: &trueValue,
		RunAsUser:    &postgresUser,
		RunAsGroup:   &postgresGroup,
		FSGroup:      &postgresGroup,
	}
}

// PodWithExistingStorage create a new instance with an existing storage
func PodWithExistingStorage(cluster apiv1.Cluster, nodeSerial int32) *corev1.Pod {
	podName := fmt.Sprintf("%s-%v", cluster.Name, nodeSerial)
	gracePeriod := int64(cluster.GetMaxStopDelay())

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Labels: map[string]string{
				ClusterLabelName:       cluster.Name,
				utils.ClusterLabelName: cluster.Name,
			},
			Annotations: map[string]string{
				ClusterSerialAnnotationName: strconv.Itoa(int(nodeSerial)),
			},
			Name:      podName,
			Namespace: cluster.Namespace,
		},
		Spec: corev1.PodSpec{
			Hostname:  podName,
			Subdomain: cluster.GetServiceAnyName(),
			InitContainers: []corev1.Container{
				createBootstrapContainer(cluster),
			},
			Containers:                    createPostgresContainers(cluster, podName),
			Volumes:                       createPostgresVolumes(cluster, podName),
			SecurityContext:               CreatePostgresSecurityContext(cluster.GetPostgresUID(), cluster.GetPostgresGID()),
			Affinity:                      CreateAffinitySection(cluster.Name, cluster.Spec.Affinity),
			Tolerations:                   cluster.Spec.Affinity.Tolerations,
			ServiceAccountName:            cluster.Name,
			NodeSelector:                  cluster.Spec.Affinity.NodeSelector,
			TerminationGracePeriodSeconds: &gracePeriod,
		},
	}

	return pod
}
