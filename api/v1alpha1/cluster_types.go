/*
This file is part of Cloud Native PostgreSQL.

Copyright (C) 2019-2021 EnterpriseDB Corporation.
*/

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/EnterpriseDB/cloud-native-postgresql/pkg/utils"
)

// ClusterSpec defines the desired state of Cluster
type ClusterSpec struct {
	// Description of this PostgreSQL cluster
	Description string `json:"description,omitempty"`

	// Name of the container image
	ImageName string `json:"imageName,omitempty"`

	// Image pull policy.
	// One of `Always`, `Never` or `IfNotPresent`.
	// If not defined, it defaults to `IfNotPresent`.
	// Cannot be updated.
	// More info: https://kubernetes.io/docs/concepts/containers/images#updating-images
	// +optional
	ImagePullPolicy corev1.PullPolicy `json:"imagePullPolicy,omitempty"`

	// The UID of the `postgres` user inside the image, defaults to `26`
	PostgresUID int64 `json:"postgresUID,omitempty"`

	// The GID of the `postgres` user inside the image, defaults to `26`
	PostgresGID int64 `json:"postgresGID,omitempty"`

	// Number of instances required in the cluster
	// +kubebuilder:validation:Minimum=1
	Instances int32 `json:"instances"`

	// Minimum number of instances required in synchronous replication with the
	// primary. Undefined or 0 allow writes to complete when no standby is
	// available.
	MinSyncReplicas int32 `json:"minSyncReplicas,omitempty"`

	// The target value for the synchronous replication quorum, that can be
	// decreased if the number of ready standbys is lower than this.
	// Undefined or 0 disable synchronous replication.
	MaxSyncReplicas int32 `json:"maxSyncReplicas,omitempty"`

	// Configuration of the PostgreSQL server
	// +optional
	PostgresConfiguration PostgresConfiguration `json:"postgresql,omitempty"`

	// Instructions to bootstrap this cluster
	// +optional
	Bootstrap *BootstrapConfiguration `json:"bootstrap,omitempty"`

	// Replica cluster configuration
	// +optional
	ReplicaCluster *ReplicaClusterConfiguration `json:"replica,omitempty"`

	// The secret containing the superuser password. If not defined, a new
	// secret will be created with a randomly generated password
	// +optional
	SuperuserSecret *LocalObjectReference `json:"superuserSecret,omitempty"`

	// The configuration for the CA and related certificates
	// +optional
	Certificates *CertificatesConfiguration `json:"certificates,omitempty"`

	// The list of pull secrets to be used to pull the images
	ImagePullSecrets []LocalObjectReference `json:"imagePullSecrets,omitempty"`

	// Configuration of the storage of the instances
	// +optional
	StorageConfiguration StorageConfiguration `json:"storage,omitempty"`

	// The time in seconds that is allowed for a PostgreSQL instance to
	// successfully start up (default 30)
	MaxStartDelay int32 `json:"startDelay,omitempty"`

	// The time in seconds that is allowed for a PostgreSQL instance node to
	// gracefully shutdown (default 30)
	MaxStopDelay int32 `json:"stopDelay,omitempty"`

	// Affinity/Anti-affinity rules for Pods
	// +optional
	Affinity AffinityConfiguration `json:"affinity,omitempty"`

	// Resources requirements of every generated Pod. Please refer to
	// https://kubernetes.io/docs/concepts/configuration/manage-resources-containers/
	// for more information.
	// +optional
	Resources corev1.ResourceRequirements `json:"resources,omitempty"`

	// Strategy to follow to upgrade the primary server during a rolling
	// update procedure, after all replicas have been successfully updated:
	// it can be automated (`unsupervised` - default) or manual (`supervised`)
	PrimaryUpdateStrategy PrimaryUpdateStrategy `json:"primaryUpdateStrategy,omitempty"`

	// The configuration to be used for backups
	Backup *BackupConfiguration `json:"backup,omitempty"`

	// Define a maintenance window for the Kubernetes nodes
	NodeMaintenanceWindow *NodeMaintenanceWindow `json:"nodeMaintenanceWindow,omitempty"`

	// The configuration of the monitoring infrastructure of this cluster
	Monitoring *MonitoringConfiguration `json:"monitoring,omitempty"`

	// The list of external server which are used in the cluster configuration
	ExternalClusters []ExternalCluster `json:"externalClusters,omitempty"`
}

// ClusterStatus defines the observed state of Cluster
type ClusterStatus struct {
	// Total number of instances in the cluster
	Instances int32 `json:"instances,omitempty"`

	// Total number of ready instances in the cluster
	ReadyInstances int32 `json:"readyInstances,omitempty"`

	// Instances status
	InstancesStatus map[utils.PodStatus][]string `json:"instancesStatus,omitempty"`

	// ID of the latest generated node (used to avoid node name clashing)
	LatestGeneratedNode int32 `json:"latestGeneratedNode,omitempty"`

	// Current primary instance
	CurrentPrimary string `json:"currentPrimary,omitempty"`

	// Target primary instance, this is different from the previous one
	// during a switchover or a failover
	TargetPrimary string `json:"targetPrimary,omitempty"`

	// How many PVCs have been created by this cluster
	PVCCount int32 `json:"pvcCount,omitempty"`

	// How many Jobs have been created by this cluster
	JobCount int32 `json:"jobCount,omitempty"`

	// List of all the PVCs created by this cluster and still available
	// which are not attached to a Pod
	DanglingPVC []string `json:"danglingPVC,omitempty"`

	// List of all the PVCs that are being initialized by this cluster
	InitializingPVC []string `json:"initializingPVC,omitempty"`

	// List of all the PVCs not dangling nor initializing
	HealthyPVC []string `json:"healthyPVC,omitempty"`

	// Current write pod
	WriteService string `json:"writeService,omitempty"`

	// Current list of read pods
	ReadService string `json:"readService,omitempty"`

	// Current phase of the cluster
	Phase string `json:"phase,omitempty"`

	// Reason for the current phase
	PhaseReason string `json:"phaseReason,omitempty"`

	// The list of resource versions of the secrets
	// managed by the operator. Every change here is done in the
	// interest of the instance manager, which will refresh the
	// secret data
	SecretsResourceVersion SecretsResourceVersion `json:"secretsResourceVersion,omitempty"`

	// The list of resource versions of the configmaps
	// managed by the operator. Every change here is done in the
	// interest of the instance manager, which will refresh the
	// configmap data
	ConfigMapResourceVersion ConfigMapResourceVersion `json:"configMapResourceVersion,omitempty"`

	// The configuration for the CA and related certificates, initialized with defaults.
	Certificates CertificatesStatus `json:"certificates,omitempty"`
}

// ReplicaClusterConfiguration encapsulates the configuration of a replica
// cluster
type ReplicaClusterConfiguration struct {
	// If replica mode is enabled, this cluster will be a replica of an
	// existing cluster. A cluster of such type can be created only
	// using bootstrap via pg_basebackup
	//+optional
	Enabled bool `json:"enabled"`

	// The name of the external server which is the replication origin
	// +kubebuilder:validation:MinLength=1
	Source string `json:"source"`
}

// PostgresConfiguration defines the PostgreSQL configuration
type PostgresConfiguration struct {
	// PostgreSQL configuration options (postgresql.conf)
	Parameters map[string]string `json:"parameters,omitempty"`

	// PostgreSQL Host Based Authentication rules (lines to be appended
	// to the pg_hba.conf file)
	// +optional
	PgHBA []string `json:"pg_hba,omitempty"`

	// Specifies the maximum number of seconds to wait when promoting an instance to primary
	// +optional
	PgCtlTimeoutForPromotion int32 `json:"promotionTimeout,omitempty"`

	// Lists of shared preload libraries to add to the default ones
	// +optional
	AdditionalLibraries []string `json:"shared_preload_libraries,omitempty"`
}

// BootstrapConfiguration contains information about how to create the PostgreSQL
// cluster. Only a single bootstrap method can be defined among the supported
// ones. `initdb` will be used as the bootstrap method if left
// unspecified. Refer to the Bootstrap page of the documentation for more
// information.
type BootstrapConfiguration struct {
	// Bootstrap the cluster via initdb
	InitDB *BootstrapInitDB `json:"initdb,omitempty"`

	// Bootstrap the cluster from a backup
	Recovery *BootstrapRecovery `json:"recovery,omitempty"`

	// Bootstrap the cluster taking a physical backup of another compatible
	// PostgreSQL instance
	PgBaseBackup *BootstrapPgBaseBackup `json:"pg_basebackup,omitempty"`
}

// CertificatesConfiguration contains the needed configurations to handle server certificates.
type CertificatesConfiguration struct {
	// The secret containing the Server CA certificate. If not defined, a new secret will be created
	// with a self-signed CA and will be used to generate the TLS certificate ServerTLSSecret.<br />
	// <br />
	// Contains:<br />
	// <br />
	// - `ca.crt`: CA that should be used to validate the server certificate,
	// used as `sslrootcert` in client connection strings.<br />
	// - `ca.key`: key used to generate Server SSL certs, if ServerTLSSecret is provided,
	// this can be omitted.<br />
	ServerCASecret string `json:"serverCASecret,omitempty"`

	// The secret of type kubernetes.io/tls containing the server TLS certificate and key that will be set as
	// `ssl_cert_file` and `ssl_key_file` so that clients can connect to postgres securely.
	// If not defined, ServerCASecret must provide also `ca.key` and a new secret will be
	// created using the provided CA.
	ServerTLSSecret string `json:"serverTLSSecret,omitempty"`

	// The secret of type kubernetes.io/tls containing the client certificate to authenticate as
	// the `streaming_replica` user
	// If not defined, ClientCASecret must provide also `ca.key` and a new secret will be
	// created using the provided CA.
	ReplicationTLSSecret string `json:"replicationTLSSecret,omitempty"`

	// The secret containing the Client CA certificate. If not defined, a new secret will be created
	// with a self-signed CA and will be used to generate all the client certificates.<br />
	// <br />
	// Contains:<br />
	// <br />
	// - `ca.crt`: CA that should be used to validate the client certificates,
	// used as `ssl_ca_file` of all the instances.<br />
	// - `ca.key`: key used to generate client certificates, if ReplicationTLSSecret is provided,
	// this can be omitted.<br />
	ClientCASecret string `json:"clientCASecret,omitempty"`

	// The list of the server alternative DNS names to be added to the generated server TLS certificates, when required.
	ServerAltDNSNames []string `json:"serverAltDNSNames,omitempty"`
}

// BootstrapRecovery contains the configuration required to restore
// the backup with the specified name and, after having changed the password
// with the one chosen for the superuser, will use it to bootstrap a full
// cluster cloning all the instances from the restored primary.
// Refer to the Bootstrap page of the documentation for more information.
type BootstrapRecovery struct {
	// The backup we need to restore
	Backup LocalObjectReference `json:"backup"`

	// By default, the recovery will end as soon as a consistent state is
	// reached: in this case, that means at the end of a backup.
	// This option allows to fine tune the recovery process
	// +optional
	RecoveryTarget *RecoveryTarget `json:"recoveryTarget,omitempty"`
}

// CertificatesStatus contains configuration certificates and related expiration dates.
type CertificatesStatus struct {
	// Needed configurations to handle server certificates, initialized with default values, if needed.
	CertificatesConfiguration `json:",inline"`

	// Expiration dates for all certificates.
	Expirations map[string]string `json:"expirations,omitempty"`
}

// BootstrapInitDB is the configuration of the bootstrap process when
// initdb is used.
// Refer to the Bootstrap page of the documentation for more information.
type BootstrapInitDB struct {
	// Name of the database used by the application. Default: `app`.
	// +optional
	Database string `json:"database"`

	// Name of the owner of the database in the instance to be used
	// by applications. Defaults to the value of the `database` key.
	// +optional
	Owner string `json:"owner"`

	// Name of the secret containing the initial credentials for the
	// owner of the user database. If empty a new secret will be
	// created from scratch
	// +optional
	Secret *LocalObjectReference `json:"secret,omitempty"`

	// The list of options that must be passed to initdb
	// when creating the cluster
	Options []string `json:"options,omitempty"`

	// List of SQL queries to be executed as a superuser immediately
	// after the cluster has been created - to be used with extreme care
	// (by default empty)
	PostInitSQL []string `json:"postInitSQL,omitempty"`
}

// BootstrapPgBaseBackup represent the configuration needed to bootstrap
// a new cluster from an existing PostgreSQL database
type BootstrapPgBaseBackup struct {
	// The name of the server of which we need to take a physical backup
	// +kubebuilder:validation:MinLength=1
	Source string `json:"source"`
}

// StorageConfiguration is the configuration of the storage of the PostgreSQL instances
type StorageConfiguration struct {
	// StorageClass to use for database data (`PGDATA`). Applied after
	// evaluating the PVC template, if available.
	// If not specified, generated PVCs will be satisfied by the
	// default storage class
	// +optional
	StorageClass *string `json:"storageClass,omitempty"`

	// Size of the storage. Required if not already specified in the PVC template.
	// Changes to this field are automatically reapplied to the created PVCs.
	// Size cannot be decreased.
	Size string `json:"size"`

	// Resize existent PVCs, defaults to true
	// +optional
	ResizeInUseVolumes *bool `json:"resizeInUseVolumes,omitempty"`

	// Template to be used to generate the Persistent Volume Claim
	// +optional
	PersistentVolumeClaimTemplate *corev1.PersistentVolumeClaimSpec `json:"pvcTemplate,omitempty"`
}

// AffinityConfiguration contains the info we need to create the
// affinity rules for Pods
type AffinityConfiguration struct {
	// Activates anti-affinity for the pods. The operator will define pods
	// anti-affinity unless this field is explicitly set to false
	// +optional
	EnablePodAntiAffinity *bool `json:"enablePodAntiAffinity,omitempty"`

	// TopologyKey to use for anti-affinity configuration. See k8s documentation
	// for more info on that
	// +optional
	TopologyKey string `json:"topologyKey"`

	// NodeSelector is map of key-value pairs used to define the nodes on which
	// the pods can run.
	// More info: https://kubernetes.io/docs/concepts/configuration/assign-pod-node/
	// +optional
	NodeSelector map[string]string `json:"nodeSelector,omitempty"`

	// Tolerations is a list of Tolerations that should be set to all the pods for this cluster.
	// More info: https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/
	// +optional
	Tolerations []corev1.Toleration `json:"tolerations,omitempty"`

	// PodAntiAffinityType allows the user to decide whether pod anti-affinity between cluster instance has to be
	// considered a strong requirement during scheduling or not. Allowed values are: "preferred" (default if empty) or
	// "required". Setting it to "required", could lead to instances remaining pending until new kubernetes nodes are
	// added if all the existing nodes don't match the required pod anti-affinity rule.
	// More info:
	// https://kubernetes.io/docs/concepts/scheduling-eviction/assign-pod-node/#inter-pod-affinity-and-anti-affinity
	// +optional
	PodAntiAffinityType string `json:"podAntiAffinityType,omitempty"`

	// AdditionalPodAntiAffinity allows to specify pod anti-affinity terms to be added to the ones generated
	// by the operator if EnablePodAntiAffinity is set to true (default) or to be used exclusively if set to false.
	// +optional
	AdditionalPodAntiAffinity *corev1.PodAntiAffinity `json:"additionalPodAntiAffinity,omitempty"`

	// AdditionalPodAffinity allows to specify pod affinity terms to be passed to all the cluster's pods.
	// +optional
	AdditionalPodAffinity *corev1.PodAffinity `json:"additionalPodAffinity,omitempty"`
}

// PrimaryUpdateStrategy contains the strategy to follow when upgrading
// the primary server of the cluster as part of rolling updates
type PrimaryUpdateStrategy string

const (
	// PrimaryUpdateStrategySupervised means that the operator need to wait for the
	// user to manually issue a switchover request before updating the primary
	// server (`supervised`)
	PrimaryUpdateStrategySupervised = "supervised"

	// PrimaryUpdateStrategyUnsupervised means that the operator will switchover
	// to another updated replica and then automatically update the primary server
	// (`unsupervised`, default)
	PrimaryUpdateStrategyUnsupervised = "unsupervised"
)

// BackupConfiguration defines how the backup of the cluster are taken.
// Currently the only supported backup method is barmanObjectStore.
// For details and examples refer to the Backup and Recovery section of the
// documentation
type BackupConfiguration struct {
	// The configuration for the barman-cloud tool suite
	BarmanObjectStore *BarmanObjectStoreConfiguration `json:"barmanObjectStore,omitempty"`
}

// BarmanObjectStoreConfiguration contains the backup configuration
// using Barman against an S3-compatible object storage
type BarmanObjectStoreConfiguration struct {
	// The credentials to use to upload data to S3
	S3Credentials *S3Credentials `json:"s3Credentials,omitempty"`

	// The credentials to use to upload data to S3
	AzureCredentials *AzureCredentials `json:"azureCredentials,omitempty"`

	// Endpoint to be used to upload data to the cloud,
	// overriding the automatic endpoint discovery
	EndpointURL string `json:"endpointURL,omitempty"`

	// EndpointCA store the CA bundle of the barman endpoint.
	// Useful when using self-signed certificates to avoid
	// errors with certificate issuer and barman-cloud-wal-archive
	EndpointCA *SecretKeySelector `json:"endpointCA,omitempty"`

	// The path where to store the backup (i.e. s3://bucket/path/to/folder)
	// this path, with different destination folders, will be used for WALs
	// and for data
	// +kubebuilder:validation:MinLength=1
	DestinationPath string `json:"destinationPath"`

	// The server name on S3, the cluster name is used if this
	// parameter is omitted
	ServerName string `json:"serverName,omitempty"`

	// The configuration for the backup of the WAL stream.
	// When not defined, WAL files will be stored uncompressed and may be
	// unencrypted in the object store, according to the bucket default policy.
	Wal *WalBackupConfiguration `json:"wal,omitempty"`

	// The configuration to be used to backup the data files
	// When not defined, base backups files will be stored uncompressed and may
	// be unencrypted in the object store, according to the bucket default
	// policy.
	Data *DataBackupConfiguration `json:"data,omitempty"`
}

// NodeMaintenanceWindow contains information that the operator
// will use while upgrading the underlying node.
//
// This option is only useful when the chosen storage prevents the Pods
// from being freely moved across nodes.
type NodeMaintenanceWindow struct {
	// Is there a node maintenance activity in progress?
	InProgress bool `json:"inProgress"`

	// Reuse the existing PVC (wait for the node to come
	// up again) or not (recreate it elsewhere)
	// +optional
	ReusePVC *bool `json:"reusePVC"`
}

// RecoveryTarget allows to configure the moment where the recovery process
// will stop. All the target options except TargetTLI are mutually exclusive.
type RecoveryTarget struct {
	// The target timeline ("latest", "current" or a positive integer)
	// +optional
	TargetTLI string `json:"targetTLI,omitempty"`

	// The target transaction ID
	// +optional
	TargetXID string `json:"targetXID,omitempty"`

	// The target name (to be previously created
	// with `pg_create_restore_point`)
	// +optional
	TargetName string `json:"targetName,omitempty"`

	// The target LSN (Log Sequence Number)
	// +optional
	TargetLSN string `json:"targetLSN,omitempty"`

	// The target time, in any unambiguous representation
	// allowed by PostgreSQL
	TargetTime string `json:"targetTime,omitempty"`

	// End recovery as soon as a consistent state is reached
	TargetImmediate *bool `json:"targetImmediate,omitempty"`

	// Set the target to be exclusive (defaults to true)
	Exclusive *bool `json:"exclusive,omitempty"`
}

// S3Credentials is the type for the credentials to be used to upload
// files to S3
type S3Credentials struct {
	// The reference to the access key id
	AccessKeyIDReference SecretKeySelector `json:"accessKeyId"`

	// The reference to the secret access key
	SecretAccessKeyReference SecretKeySelector `json:"secretAccessKey"`
}

// AzureCredentials is the type for the credentials to be used to upload
// files to Azure Blob Storage. The connection string contains every needed
// information. If the connection string is not specified, we'll need the
// storage account name and also one (and only one) of:
//
// - storageKey
// - storageSasToken
type AzureCredentials struct {
	// The connection string to be used
	ConnectionString *SecretKeySelector `json:"connectionString,omitempty"`

	// The storage account where to upload data
	StorageAccount *SecretKeySelector `json:"storageAccount,omitempty"`

	// The storage account key to be used in conjunction
	// with the storage account name
	StorageKey *SecretKeySelector `json:"storageKey,omitempty"`

	// A shared-access-signature to be used in conjunction with
	// the storage account name
	StorageSasToken *SecretKeySelector `json:"storageSasToken,omitempty"`
}

// WalBackupConfiguration is the configuration of the backup of the
// WAL stream
type WalBackupConfiguration struct {
	// Compress a WAL file before sending it to the object store. Available
	// options are empty string (no compression, default), `gzip` or `bzip2`.
	Compression CompressionType `json:"compression,omitempty"`

	// Whenever to force the encryption of files (if the bucket is
	// not already configured for that).
	// Allowed options are empty string (use the bucket policy, default),
	// `AES256` and `aws:kms`
	Encryption EncryptionType `json:"encryption,omitempty"`
}

// DataBackupConfiguration is the configuration of the backup of
// the data directory
type DataBackupConfiguration struct {
	// Compress a backup file (a tar file per tablespace) while streaming it
	// to the object store. Available options are empty string (no
	// compression, default), `gzip` or `bzip2`.
	Compression CompressionType `json:"compression,omitempty"`

	// Whenever to force the encryption of files (if the bucket is
	// not already configured for that).
	// Allowed options are empty string (use the bucket policy, default),
	// `AES256` and `aws:kms`
	Encryption EncryptionType `json:"encryption,omitempty"`

	// Control whether the I/O workload for the backup initial checkpoint will
	// be limited, according to the `checkpoint_completion_target` setting on
	// the PostgreSQL server. If set to true, an immediate checkpoint will be
	// used, meaning PostgreSQL will complete the checkpoint as soon as
	// possible. `false` by default.
	ImmediateCheckpoint bool `json:"immediateCheckpoint,omitempty"`

	// The number of parallel jobs to be used to upload the backup, defaults
	// to 2
	Jobs *int32 `json:"jobs,omitempty"`
}

// CompressionType encapsulates the available types of compression
type CompressionType string

const (
	// CompressionTypeNone means no compression is performed
	CompressionTypeNone = ""

	// CompressionTypeGzip means gzip compression is performed
	CompressionTypeGzip = "gzip"

	// CompressionTypeBzip2 means bzip2 compression is performed
	CompressionTypeBzip2 = "bzip2"
)

// EncryptionType encapsulated the available types of encryption
type EncryptionType string

const (
	// EncryptionTypeNone means just use the bucket configuration
	EncryptionTypeNone = ""

	// EncryptionTypeAES256 means to use AES256 encryption
	EncryptionTypeAES256 = "AES256"

	// EncryptionTypeNoneAWSKMS means to use aws:kms encryption
	EncryptionTypeNoneAWSKMS = "aws:kms"
)

// MonitoringConfiguration is the type containing all the monitoring
// configuration for a certain cluster
type MonitoringConfiguration struct {
	// The list of config maps containing the custom queries
	CustomQueriesConfigMap []ConfigMapKeySelector `json:"customQueriesConfigMap,omitempty"`

	// The list of secrets containing the custom queries
	CustomQueriesSecret []SecretKeySelector `json:"customQueriesSecret,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status
// +kubebuilder:subresource:scale:specpath=.spec.instances,statuspath=.status.instances
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="Instances",type="integer",JSONPath=".status.instances",description="Number of instances"
// +kubebuilder:printcolumn:name="Ready",type="integer",JSONPath=".status.readyInstances",description="Number of ready instances"
// +kubebuilder:printcolumn:name="Status",type="string",JSONPath=".status.phase",description="Cluster current status"
// +kubebuilder:printcolumn:name="Primary",type="string",JSONPath=".status.currentPrimary",description="Primary pod"

// Cluster is the Schema for the PostgreSQL API
type Cluster struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Specification of the desired behavior of the cluster.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status
	Spec ClusterSpec `json:"spec,omitempty"`
	// Most recently observed status of the cluster. This data may not be up
	// to date. Populated by the system. Read-only.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status
	Status ClusterStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ClusterList contains a list of Cluster
type ClusterList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds
	metav1.ListMeta `json:"metadata,omitempty"`
	// List of clusters
	Items []Cluster `json:"items"`
}

// SecretsResourceVersion is the resource versions of the secrets
// managed by the operator
type SecretsResourceVersion struct {
	// The resource version of the "postgres" user secret
	SuperuserSecretVersion string `json:"superuserSecretVersion,omitempty"`

	// The resource version of the "streaming_replica" user secret
	ReplicationSecretVersion string `json:"replicationSecretVersion,omitempty"`

	// The resource version of the "app" user secret
	ApplicationSecretVersion string `json:"applicationSecretVersion,omitempty"`

	// Unused. Retained for compatibility with old versions.
	CASecretVersion string `json:"caSecretVersion,omitempty"`

	// The resource version of the PostgreSQL client-side CA secret version
	ClientCASecretVersion string `json:"clientCaSecretVersion,omitempty"`

	// The resource version of the PostgreSQL server-side CA secret version
	ServerCASecretVersion string `json:"serverCaSecretVersion,omitempty"`

	// The resource version of the PostgreSQL server-side secret version
	ServerSecretVersion string `json:"serverSecretVersion,omitempty"`

	// The resource version of the Barman Endpoint CA if provided
	BarmanEndpointCA string `json:"barmanEndpointCA,omitempty"`

	// The versions of all the secrets used to pass metrics
	Metrics map[string]string `json:"metrics,omitempty"`
}

// ConfigMapResourceVersion is the resource versions of the secrets
// managed by the operator
type ConfigMapResourceVersion struct {
	// The versions of all the configmaps used to pass metrics
	Metrics map[string]string `json:"metrics,omitempty"`
}

// ExternalCluster represent the connection parameters to an
// external server which is used in the cluster configuration
type ExternalCluster struct {
	// The server name, required
	Name string `json:"name"`

	// The list of connection parameters, such as dbname, host, username, etc
	ConnectionParameters map[string]string `json:"connectionParameters,omitempty"`

	// The reference to an SSL certificate to be used to connect to this
	// instance
	SSLCert *corev1.SecretKeySelector `json:"sslCert,omitempty"`

	// The reference to an SSL private key to be used to connect to this
	// instance
	SSLKey *corev1.SecretKeySelector `json:"sslKey,omitempty"`

	// The reference to an SSL CA public key to be used to connect to this
	// instance
	SSLRootCert *corev1.SecretKeySelector `json:"sslRootCert,omitempty"`

	// The reference to the password to be used to connect to the server
	Password *corev1.SecretKeySelector `json:"password,omitempty"`
}

func init() {
	SchemeBuilder.Register(&Cluster{}, &ClusterList{})
}
