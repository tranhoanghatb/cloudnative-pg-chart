/*
This file is part of Cloud Native PostgreSQL.

Copyright (C) 2019-2021 EnterpriseDB Corporation.
*/

package controllers

import (
	corev1 "k8s.io/api/core/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"

	"github.com/EnterpriseDB/cloud-native-postgresql/pkg/specs"
)

var log = ctrl.Log.WithName("cluster_predicates")

var (
	isUsefulConfigMap = func(object client.Object) bool {
		return isOwnedOrSatisfiesPredicate(object, func(object client.Object) bool {
			_, ok := object.(*corev1.ConfigMap)
			return ok || hasReloadLabelSet(object)
		})
	}

	isUsefulSecret = func(object client.Object) bool {
		return isOwnedOrSatisfiesPredicate(object, func(object client.Object) bool {
			_, ok := object.(*corev1.Secret)
			return ok || hasReloadLabelSet(object)
		})
	}

	configMapsPredicate = predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return isUsefulConfigMap(e.Object)
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return isUsefulConfigMap(e.Object)
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return isUsefulConfigMap(e.Object)
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			return isUsefulConfigMap(e.ObjectNew)
		},
	}

	secretsPredicate = predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return isUsefulSecret(e.Object)
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return isUsefulSecret(e.Object)
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return isUsefulSecret(e.Object)
		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			return isUsefulSecret(e.ObjectNew)
		},
	}

	nodesPredicate = predicate.Funcs{
		UpdateFunc: func(e event.UpdateEvent) bool {
			oldNode, oldOk := e.ObjectOld.(*corev1.Node)
			newNode, newOk := e.ObjectNew.(*corev1.Node)
			return oldOk && newOk && oldNode.Spec.Unschedulable != newNode.Spec.Unschedulable
		},
		CreateFunc: func(createEvent event.CreateEvent) bool {
			return false
		},
		DeleteFunc: func(createEvent event.DeleteEvent) bool {
			return false
		},
		GenericFunc: func(genericEvent event.GenericEvent) bool {
			return false
		},
	}
)

func isOwnedOrSatisfiesPredicate(
	object client.Object,
	predicate func(client.Object) bool,
) bool {
	_, owned := isOwnedByCluster(object)
	return owned || predicate(object)
}

func hasReloadLabelSet(obj client.Object) bool {
	_, hasLabel := obj.GetLabels()[specs.WatchedLabelName]
	return hasLabel
}
