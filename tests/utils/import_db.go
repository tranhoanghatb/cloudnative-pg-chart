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

package utils

import (
	"fmt"
	"os"

	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"

	apiv1 "github.com/cloudnative-pg/cloudnative-pg/api/v1"
)

// ImportDatabaseMicroservice creates a cluster, starting from an external cluster
// using microservice approach
// NOTE: the application user on the source Cluster needs to be granted with
// REPLICATION permissions, which are not set by default
func ImportDatabaseMicroservice(
	namespace,
	sourceClusterName,
	importedClusterName,
	imageName,
	databaseName string,
	env *TestingEnvironment,
) (*apiv1.Cluster, error) {
	if imageName == "" {
		imageName = os.Getenv("POSTGRES_IMG")
	}
	storageClassName := os.Getenv("E2E_DEFAULT_STORAGE_CLASS")
	host, err := GetHostName(namespace, sourceClusterName, env)
	if err != nil {
		return nil, err
	}
	appUserSecretName := sourceClusterName + apiv1.ApplicationUserSecretSuffix
	restoreCluster := &apiv1.Cluster{
		ObjectMeta: v1.ObjectMeta{
			Name:      importedClusterName,
			Namespace: namespace,
		},
		Spec: apiv1.ClusterSpec{
			Instances: 3,
			ImageName: imageName,

			StorageConfiguration: apiv1.StorageConfiguration{
				Size:         "1Gi",
				StorageClass: &storageClassName,
			},

			Bootstrap: &apiv1.BootstrapConfiguration{
				InitDB: &apiv1.BootstrapInitDB{
					Import: &apiv1.Import{
						Type:      "microservice",
						Databases: []string{databaseName},
						Source: apiv1.ImportSource{
							ExternalCluster: sourceClusterName,
						},
						PgDumpExtraOptions:       []string{"--jobs=2"},
						PgRestoreExtraOptions:    []string{"--jobs=2"},
						PostImportApplicationSQL: []string{"SELECT 1"},
					},
				},
			},

			ExternalClusters: []apiv1.ExternalCluster{
				{
					Name: sourceClusterName,
					ConnectionParameters: map[string]string{
						"host":   host,
						"user":   AppUser,
						"dbname": AppDBName,
					},
					Password: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: appUserSecretName,
						},
						Key: "password",
					},
				},
			},
		},
	}

	obj, err := CreateObject(env, restoreCluster)
	if err != nil {
		return nil, err
	}
	cluster, ok := obj.(*apiv1.Cluster)
	if !ok {
		return nil, fmt.Errorf("created object is not of cluster type: %T %v", obj, obj)
	}
	return cluster, nil
}

// ImportDatabasesMonolith creates a new cluster spec importing from a sourceCluster
// using the Monolith approach.
// Imports all the specified `databaseNames` and `roles` from the source cluster
// NOTE: enableSuperuserAccess needs to be enabled
func ImportDatabasesMonolith(
	namespace,
	sourceClusterName,
	importedClusterName,
	imageName string,
	databaseNames []string,
	roles []string,
	env *TestingEnvironment,
) (*apiv1.Cluster, error) {
	if imageName == "" {
		imageName = os.Getenv("POSTGRES_IMG")
	}
	storageClassName := os.Getenv("E2E_DEFAULT_STORAGE_CLASS")
	host, err := GetHostName(namespace, sourceClusterName, env)
	if err != nil {
		return nil, err
	}
	superUserSecretName := sourceClusterName + apiv1.SuperUserSecretSuffix
	targetCluster := &apiv1.Cluster{
		ObjectMeta: v1.ObjectMeta{
			Name:      importedClusterName,
			Namespace: namespace,
		},
		Spec: apiv1.ClusterSpec{
			Instances:             3,
			ImageName:             imageName,
			EnableSuperuserAccess: ptr.To(true),

			StorageConfiguration: apiv1.StorageConfiguration{
				Size:         "1Gi",
				StorageClass: &storageClassName,
			},

			Bootstrap: &apiv1.BootstrapConfiguration{
				InitDB: &apiv1.BootstrapInitDB{
					Import: &apiv1.Import{
						Type:      "monolith",
						Databases: databaseNames,
						Roles:     roles,
						Source: apiv1.ImportSource{
							ExternalCluster: sourceClusterName,
						},
					},
				},
			},
			ExternalClusters: []apiv1.ExternalCluster{
				{
					Name: sourceClusterName,
					ConnectionParameters: map[string]string{
						"host":   host,
						"user":   PostgresUser,
						"dbname": PostgresDBName,
					},
					Password: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: superUserSecretName,
						},
						Key: "password",
					},
				},
			},
		},
	}

	obj, err := CreateObject(env, targetCluster)
	if err != nil {
		return nil, err
	}
	cluster, ok := obj.(*apiv1.Cluster)
	if !ok {
		return nil, fmt.Errorf("created object is not of cluster type: %T %v", obj, obj)
	}
	return cluster, nil
}
