// +build !ignore_autogenerated

/*
This file is part of Cloud Native PostgreSQL.

Copyright (C) 2019-2021 EnterpriseDB Corporation.
*/

// Code generated by controller-gen. DO NOT EDIT.

package v1

import (
	"github.com/EnterpriseDB/cloud-native-postgresql/pkg/utils"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *AffinityConfiguration) DeepCopyInto(out *AffinityConfiguration) {
	*out = *in
	if in.EnablePodAntiAffinity != nil {
		in, out := &in.EnablePodAntiAffinity, &out.EnablePodAntiAffinity
		*out = new(bool)
		**out = **in
	}
	if in.NodeSelector != nil {
		in, out := &in.NodeSelector, &out.NodeSelector
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new AffinityConfiguration.
func (in *AffinityConfiguration) DeepCopy() *AffinityConfiguration {
	if in == nil {
		return nil
	}
	out := new(AffinityConfiguration)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Backup) DeepCopyInto(out *Backup) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Backup.
func (in *Backup) DeepCopy() *Backup {
	if in == nil {
		return nil
	}
	out := new(Backup)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Backup) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BackupConfiguration) DeepCopyInto(out *BackupConfiguration) {
	*out = *in
	if in.BarmanObjectStore != nil {
		in, out := &in.BarmanObjectStore, &out.BarmanObjectStore
		*out = new(BarmanObjectStoreConfiguration)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BackupConfiguration.
func (in *BackupConfiguration) DeepCopy() *BackupConfiguration {
	if in == nil {
		return nil
	}
	out := new(BackupConfiguration)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BackupList) DeepCopyInto(out *BackupList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Backup, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BackupList.
func (in *BackupList) DeepCopy() *BackupList {
	if in == nil {
		return nil
	}
	out := new(BackupList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *BackupList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BackupSpec) DeepCopyInto(out *BackupSpec) {
	*out = *in
	out.Cluster = in.Cluster
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BackupSpec.
func (in *BackupSpec) DeepCopy() *BackupSpec {
	if in == nil {
		return nil
	}
	out := new(BackupSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BackupStatus) DeepCopyInto(out *BackupStatus) {
	*out = *in
	in.S3Credentials.DeepCopyInto(&out.S3Credentials)
	if in.StartedAt != nil {
		in, out := &in.StartedAt, &out.StartedAt
		*out = (*in).DeepCopy()
	}
	if in.StoppedAt != nil {
		in, out := &in.StoppedAt, &out.StoppedAt
		*out = (*in).DeepCopy()
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BackupStatus.
func (in *BackupStatus) DeepCopy() *BackupStatus {
	if in == nil {
		return nil
	}
	out := new(BackupStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BarmanObjectStoreConfiguration) DeepCopyInto(out *BarmanObjectStoreConfiguration) {
	*out = *in
	in.S3Credentials.DeepCopyInto(&out.S3Credentials)
	if in.Wal != nil {
		in, out := &in.Wal, &out.Wal
		*out = new(WalBackupConfiguration)
		**out = **in
	}
	if in.Data != nil {
		in, out := &in.Data, &out.Data
		*out = new(DataBackupConfiguration)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BarmanObjectStoreConfiguration.
func (in *BarmanObjectStoreConfiguration) DeepCopy() *BarmanObjectStoreConfiguration {
	if in == nil {
		return nil
	}
	out := new(BarmanObjectStoreConfiguration)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BootstrapConfiguration) DeepCopyInto(out *BootstrapConfiguration) {
	*out = *in
	if in.InitDB != nil {
		in, out := &in.InitDB, &out.InitDB
		*out = new(BootstrapInitDB)
		(*in).DeepCopyInto(*out)
	}
	if in.Recovery != nil {
		in, out := &in.Recovery, &out.Recovery
		*out = new(BootstrapRecovery)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BootstrapConfiguration.
func (in *BootstrapConfiguration) DeepCopy() *BootstrapConfiguration {
	if in == nil {
		return nil
	}
	out := new(BootstrapConfiguration)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BootstrapInitDB) DeepCopyInto(out *BootstrapInitDB) {
	*out = *in
	if in.Secret != nil {
		in, out := &in.Secret, &out.Secret
		*out = new(corev1.LocalObjectReference)
		**out = **in
	}
	if in.Options != nil {
		in, out := &in.Options, &out.Options
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BootstrapInitDB.
func (in *BootstrapInitDB) DeepCopy() *BootstrapInitDB {
	if in == nil {
		return nil
	}
	out := new(BootstrapInitDB)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *BootstrapRecovery) DeepCopyInto(out *BootstrapRecovery) {
	*out = *in
	out.Backup = in.Backup
	if in.RecoveryTarget != nil {
		in, out := &in.RecoveryTarget, &out.RecoveryTarget
		*out = new(RecoveryTarget)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new BootstrapRecovery.
func (in *BootstrapRecovery) DeepCopy() *BootstrapRecovery {
	if in == nil {
		return nil
	}
	out := new(BootstrapRecovery)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Cluster) DeepCopyInto(out *Cluster) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Cluster.
func (in *Cluster) DeepCopy() *Cluster {
	if in == nil {
		return nil
	}
	out := new(Cluster)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Cluster) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterList) DeepCopyInto(out *ClusterList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Cluster, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterList.
func (in *ClusterList) DeepCopy() *ClusterList {
	if in == nil {
		return nil
	}
	out := new(ClusterList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ClusterList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterSpec) DeepCopyInto(out *ClusterSpec) {
	*out = *in
	in.PostgresConfiguration.DeepCopyInto(&out.PostgresConfiguration)
	if in.Bootstrap != nil {
		in, out := &in.Bootstrap, &out.Bootstrap
		*out = new(BootstrapConfiguration)
		(*in).DeepCopyInto(*out)
	}
	if in.SuperuserSecret != nil {
		in, out := &in.SuperuserSecret, &out.SuperuserSecret
		*out = new(corev1.LocalObjectReference)
		**out = **in
	}
	if in.ImagePullSecrets != nil {
		in, out := &in.ImagePullSecrets, &out.ImagePullSecrets
		*out = make([]corev1.LocalObjectReference, len(*in))
		copy(*out, *in)
	}
	in.StorageConfiguration.DeepCopyInto(&out.StorageConfiguration)
	in.Affinity.DeepCopyInto(&out.Affinity)
	in.Resources.DeepCopyInto(&out.Resources)
	if in.Backup != nil {
		in, out := &in.Backup, &out.Backup
		*out = new(BackupConfiguration)
		(*in).DeepCopyInto(*out)
	}
	if in.NodeMaintenanceWindow != nil {
		in, out := &in.NodeMaintenanceWindow, &out.NodeMaintenanceWindow
		*out = new(NodeMaintenanceWindow)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterSpec.
func (in *ClusterSpec) DeepCopy() *ClusterSpec {
	if in == nil {
		return nil
	}
	out := new(ClusterSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterStatus) DeepCopyInto(out *ClusterStatus) {
	*out = *in
	if in.InstancesStatus != nil {
		in, out := &in.InstancesStatus, &out.InstancesStatus
		*out = make(map[utils.PodStatus][]string, len(*in))
		for key, val := range *in {
			var outVal []string
			if val == nil {
				(*out)[key] = nil
			} else {
				in, out := &val, &outVal
				*out = make([]string, len(*in))
				copy(*out, *in)
			}
			(*out)[key] = outVal
		}
	}
	if in.DanglingPVC != nil {
		in, out := &in.DanglingPVC, &out.DanglingPVC
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterStatus.
func (in *ClusterStatus) DeepCopy() *ClusterStatus {
	if in == nil {
		return nil
	}
	out := new(ClusterStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DataBackupConfiguration) DeepCopyInto(out *DataBackupConfiguration) {
	*out = *in
	if in.Jobs != nil {
		in, out := &in.Jobs, &out.Jobs
		*out = new(int32)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DataBackupConfiguration.
func (in *DataBackupConfiguration) DeepCopy() *DataBackupConfiguration {
	if in == nil {
		return nil
	}
	out := new(DataBackupConfiguration)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NodeMaintenanceWindow) DeepCopyInto(out *NodeMaintenanceWindow) {
	*out = *in
	if in.ReusePVC != nil {
		in, out := &in.ReusePVC, &out.ReusePVC
		*out = new(bool)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NodeMaintenanceWindow.
func (in *NodeMaintenanceWindow) DeepCopy() *NodeMaintenanceWindow {
	if in == nil {
		return nil
	}
	out := new(NodeMaintenanceWindow)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PostgresConfiguration) DeepCopyInto(out *PostgresConfiguration) {
	*out = *in
	if in.Parameters != nil {
		in, out := &in.Parameters, &out.Parameters
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
	if in.PgHBA != nil {
		in, out := &in.PgHBA, &out.PgHBA
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PostgresConfiguration.
func (in *PostgresConfiguration) DeepCopy() *PostgresConfiguration {
	if in == nil {
		return nil
	}
	out := new(PostgresConfiguration)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RecoveryTarget) DeepCopyInto(out *RecoveryTarget) {
	*out = *in
	if in.TargetImmediate != nil {
		in, out := &in.TargetImmediate, &out.TargetImmediate
		*out = new(bool)
		**out = **in
	}
	if in.Exclusive != nil {
		in, out := &in.Exclusive, &out.Exclusive
		*out = new(bool)
		**out = **in
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RecoveryTarget.
func (in *RecoveryTarget) DeepCopy() *RecoveryTarget {
	if in == nil {
		return nil
	}
	out := new(RecoveryTarget)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *S3Credentials) DeepCopyInto(out *S3Credentials) {
	*out = *in
	in.AccessKeyIDReference.DeepCopyInto(&out.AccessKeyIDReference)
	in.SecretAccessKeyReference.DeepCopyInto(&out.SecretAccessKeyReference)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new S3Credentials.
func (in *S3Credentials) DeepCopy() *S3Credentials {
	if in == nil {
		return nil
	}
	out := new(S3Credentials)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ScheduledBackup) DeepCopyInto(out *ScheduledBackup) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ScheduledBackup.
func (in *ScheduledBackup) DeepCopy() *ScheduledBackup {
	if in == nil {
		return nil
	}
	out := new(ScheduledBackup)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ScheduledBackup) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ScheduledBackupList) DeepCopyInto(out *ScheduledBackupList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]ScheduledBackup, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ScheduledBackupList.
func (in *ScheduledBackupList) DeepCopy() *ScheduledBackupList {
	if in == nil {
		return nil
	}
	out := new(ScheduledBackupList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ScheduledBackupList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ScheduledBackupSpec) DeepCopyInto(out *ScheduledBackupSpec) {
	*out = *in
	if in.Suspend != nil {
		in, out := &in.Suspend, &out.Suspend
		*out = new(bool)
		**out = **in
	}
	out.Cluster = in.Cluster
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ScheduledBackupSpec.
func (in *ScheduledBackupSpec) DeepCopy() *ScheduledBackupSpec {
	if in == nil {
		return nil
	}
	out := new(ScheduledBackupSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ScheduledBackupStatus) DeepCopyInto(out *ScheduledBackupStatus) {
	*out = *in
	if in.LastCheckTime != nil {
		in, out := &in.LastCheckTime, &out.LastCheckTime
		*out = (*in).DeepCopy()
	}
	if in.LastScheduleTime != nil {
		in, out := &in.LastScheduleTime, &out.LastScheduleTime
		*out = (*in).DeepCopy()
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ScheduledBackupStatus.
func (in *ScheduledBackupStatus) DeepCopy() *ScheduledBackupStatus {
	if in == nil {
		return nil
	}
	out := new(ScheduledBackupStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *StorageConfiguration) DeepCopyInto(out *StorageConfiguration) {
	*out = *in
	if in.StorageClass != nil {
		in, out := &in.StorageClass, &out.StorageClass
		*out = new(string)
		**out = **in
	}
	if in.ResizeInUseVolumes != nil {
		in, out := &in.ResizeInUseVolumes, &out.ResizeInUseVolumes
		*out = new(bool)
		**out = **in
	}
	if in.PersistentVolumeClaimTemplate != nil {
		in, out := &in.PersistentVolumeClaimTemplate, &out.PersistentVolumeClaimTemplate
		*out = new(corev1.PersistentVolumeClaimSpec)
		(*in).DeepCopyInto(*out)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new StorageConfiguration.
func (in *StorageConfiguration) DeepCopy() *StorageConfiguration {
	if in == nil {
		return nil
	}
	out := new(StorageConfiguration)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *WalBackupConfiguration) DeepCopyInto(out *WalBackupConfiguration) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new WalBackupConfiguration.
func (in *WalBackupConfiguration) DeepCopy() *WalBackupConfiguration {
	if in == nil {
		return nil
	}
	out := new(WalBackupConfiguration)
	in.DeepCopyInto(out)
	return out
}
