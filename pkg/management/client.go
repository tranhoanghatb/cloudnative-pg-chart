/*
This file is part of Cloud Native PostgreSQL.

Copyright (C) 2019-2021 EnterpriseDB Corporation.
*/

// Package management contains all the features needed by the instance
// manager that runs in each Pod as PID 1
package management

import (
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	clientgoscheme "k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"

	apiv1 "github.com/EnterpriseDB/cloud-native-postgresql/api/v1"
)

// Scheme used for the instance manager
var Scheme = runtime.NewScheme()

func init() {
	_ = clientgoscheme.AddToScheme(Scheme)
	_ = apiv1.AddToScheme(Scheme)
}

// NewControllerRuntimeClient creates a new typed K8s client where
// the PostgreSQL CRD has already been registered
func NewControllerRuntimeClient() (client.Client, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	return client.New(config, client.Options{
		Scheme: Scheme,
		Mapper: nil,
	})
}

// newClientGoClient creates a new client-go kubernetes interface.
// It is used only to create event recorders, as controller-runtime do.
func newClientGoClient() (kubernetes.Interface, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	return kubernetes.NewForConfig(config)
}

// NewDynamicClient creates a new dynamic kubernetes interface
func NewDynamicClient() (dynamic.Interface, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}

	return dynamic.NewForConfig(config)
}

// NewEventRecorder creates a new event recorder
func NewEventRecorder() (record.EventRecorder, error) {
	kubeClient, err := newClientGoClient()
	if err != nil {
		return nil, err
	}

	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartRecordingToSink(
		&typedcorev1.EventSinkImpl{
			Interface: kubeClient.CoreV1().Events(""),
		})
	recorder := eventBroadcaster.NewRecorder(
		Scheme,
		v1.EventSource{Component: "instance-manager"},
	)

	return recorder, nil
}
