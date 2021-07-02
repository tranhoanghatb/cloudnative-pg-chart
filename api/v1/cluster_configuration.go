/*
This file is part of Cloud Native PostgreSQL.

Copyright (C) 2019-2021 EnterpriseDB Corporation.
*/

package v1

import (
	"sort"

	"github.com/EnterpriseDB/cloud-native-postgresql/pkg/postgres"
	"github.com/EnterpriseDB/cloud-native-postgresql/pkg/utils"
)

// CreatePostgresqlConfiguration creates the PostgreSQL configuration to be
// used for this cluster and return it and its sha256 checksum
func (cluster *Cluster) CreatePostgresqlConfiguration() (string, string, error) {
	// Extract the PostgreSQL major version
	imageName := cluster.GetImageName()
	tag := utils.GetImageTag(imageName)
	fromVersion, err := postgres.GetPostgresVersionFromTag(tag)
	if err != nil {
		return "", "", err
	}

	info := postgres.ConfigurationInfo{
		Settings:                         postgres.CnpConfigurationSettings,
		MajorVersion:                     fromVersion,
		UserSettings:                     cluster.Spec.PostgresConfiguration.Parameters,
		IncludingMandatory:               true,
		PgAuditEnabled:                   postgres.IsPgAuditEnabled(cluster.Spec.PostgresConfiguration.Parameters),
		AdditionalSharedPreloadLibraries: cluster.Spec.PostgresConfiguration.AdditionalLibraries,
	}

	// We need to include every replica inside the list of possible synchronous standbys
	info.Replicas = nil
	for _, instances := range cluster.Status.InstancesStatus {
		info.Replicas = append(info.Replicas, instances...)
	}

	// Ensure a consistent ordering to avoid spurious configuration changes
	sort.Strings(info.Replicas)

	// We start with the number of healthy replicas (healthy pods minus one)
	// and verify it is between minSyncReplicas and maxSyncReplicas
	info.SyncReplicas = len(cluster.Status.InstancesStatus[utils.PodHealthy]) - 1
	if info.SyncReplicas > int(cluster.Spec.MaxSyncReplicas) {
		info.SyncReplicas = int(cluster.Spec.MaxSyncReplicas)
	}
	if info.SyncReplicas < int(cluster.Spec.MinSyncReplicas) {
		info.SyncReplicas = int(cluster.Spec.MinSyncReplicas)
	}

	conf, sha256 := postgres.CreatePostgresqlConfFile(postgres.CreatePostgresqlConfiguration(info))
	return conf, sha256, nil
}

// CreatePostgresqlHBA create the HBA rules for this cluster
func (cluster *Cluster) CreatePostgresqlHBA() string {
	return postgres.CreateHBARules(cluster.Spec.PostgresConfiguration.PgHBA)
}
