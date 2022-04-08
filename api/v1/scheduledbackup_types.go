/*
Copyright 2019-2022 The CloudNativePG Contributors

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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/cloudnative-pg/cloudnative-pg/internal/configuration"
	"github.com/cloudnative-pg/cloudnative-pg/pkg/utils"
)

// ScheduledBackupSpec defines the desired state of ScheduledBackup
type ScheduledBackupSpec struct {
	// If this backup is suspended or not
	Suspend *bool `json:"suspend,omitempty"`

	// If the first backup has to be immediately start after creation or not
	Immediate *bool `json:"immediate,omitempty"`

	// The schedule follows the same format used in Kubernetes CronJobs,
	// see https://pkg.go.dev/github.com/robfig/cron#hdr-CRON_Expression_Format
	Schedule string `json:"schedule"`

	// The cluster to backup
	Cluster LocalObjectReference `json:"cluster,omitempty"`
}

// ScheduledBackupStatus defines the observed state of ScheduledBackup
type ScheduledBackupStatus struct {
	// The latest time the schedule
	LastCheckTime *metav1.Time `json:"lastCheckTime,omitempty"`

	// Information when was the last time that backup was successfully scheduled.
	// +optional
	LastScheduleTime *metav1.Time `json:"lastScheduleTime,omitempty"`

	// Next time we will run a backup
	// +optional
	NextScheduleTime *metav1.Time `json:"nextScheduleTime,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:storageversion
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"
// +kubebuilder:printcolumn:name="Cluster",type="string",JSONPath=".spec.cluster.name"
// +kubebuilder:printcolumn:name="Last Backup",type="date",JSONPath=".status.lastScheduleTime"

// ScheduledBackup is the Schema for the scheduledbackups API
type ScheduledBackup struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	// Specification of the desired behavior of the ScheduledBackup.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status
	Spec ScheduledBackupSpec `json:"spec,omitempty"`
	// Most recently observed status of the ScheduledBackup. This data may not be up
	// to date. Populated by the system. Read-only.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#spec-and-status
	Status ScheduledBackupStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ScheduledBackupList contains a list of ScheduledBackup
type ScheduledBackupList struct {
	metav1.TypeMeta `json:",inline"`
	// Standard list metadata.
	// More info: https://git.k8s.io/community/contributors/devel/sig-architecture/api-conventions.md#types-kinds
	metav1.ListMeta `json:"metadata,omitempty"`
	// List of clusters
	Items []ScheduledBackup `json:"items"`
}

// IsSuspended check if a scheduled backup has been suspended or not
func (scheduledBackup ScheduledBackup) IsSuspended() bool {
	if scheduledBackup.Spec.Suspend == nil {
		return false
	}

	return *scheduledBackup.Spec.Suspend
}

// IsImmediate check if a backup has to be issued immediately upon creation or not
func (scheduledBackup ScheduledBackup) IsImmediate() bool {
	if scheduledBackup.Spec.Immediate == nil {
		return false
	}

	return *scheduledBackup.Spec.Immediate
}

// GetName gets the scheduled backup name
func (scheduledBackup *ScheduledBackup) GetName() string {
	return scheduledBackup.Name
}

// GetNamespace gets the scheduled backup name
func (scheduledBackup *ScheduledBackup) GetNamespace() string {
	return scheduledBackup.Namespace
}

// GetSchedule get the cron-like schedule of this scheduled backup
func (scheduledBackup *ScheduledBackup) GetSchedule() string {
	return scheduledBackup.Spec.Schedule
}

// GetStatus gets the status that the caller may update
func (scheduledBackup *ScheduledBackup) GetStatus() *ScheduledBackupStatus {
	return &scheduledBackup.Status
}

// GetKubernetesObject gets the kubernetes object
func (scheduledBackup *ScheduledBackup) GetKubernetesObject() client.Object {
	return scheduledBackup
}

// CreateBackup creates a backup from this scheduled backup
func (scheduledBackup *ScheduledBackup) CreateBackup(name string) *Backup {
	backup := Backup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: scheduledBackup.Namespace,
		},
		Spec: BackupSpec{
			Cluster: scheduledBackup.Spec.Cluster,
		},
	}
	utils.InheritAnnotations(&backup.ObjectMeta, scheduledBackup.Annotations, nil, configuration.Current)
	return &backup
}

func init() {
	SchemeBuilder.Register(&ScheduledBackup{}, &ScheduledBackupList{})
}
