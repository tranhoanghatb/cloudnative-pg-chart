/*
This file is part of Cloud Native PostgreSQL.

Copyright (C) 2019-2022 EnterpriseDB Corporation.
*/

// Package multicache implements a cache that is able to work on multiple namespaces but also able to
// read data from a namespace which is beside the specified ones. This is different from the
// MultiNamespacedCache implementation that is inside the controller-runtime library.
package multicache

import (
	"context"
	"fmt"

	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/EnterpriseDB/cloud-native-postgresql/pkg/management/log"
	"github.com/EnterpriseDB/cloud-native-postgresql/pkg/stringset"
)

type multiNamespaceCache struct {
	namespaces    *stringset.Data
	multiCache    cache.Cache
	externalCache cache.Cache
}

// Just to ensure we respect the interface
var _ cache.Cache = &multiNamespaceCache{}

// DelegatingMultiNamespacedCacheBuilder returns a cache creation function. The
// created cache is able to work on multiple namespaces but also to respond, as a
// plain client, to requests belonging to namespaces different from the specified
// ones.
func DelegatingMultiNamespacedCacheBuilder(namespaces []string, operatorNamespace string) cache.NewCacheFunc {
	return func(config *rest.Config, opts cache.Options) (cache.Cache, error) {
		multiCache, err := cache.MultiNamespacedCacheBuilder(namespaces)(config, opts)
		if err != nil {
			return nil, err
		}

		// create a cache for external resources
		externalOpts := opts
		externalOpts.Namespace = operatorNamespace
		externalCache, err := cache.New(config, externalOpts)
		if err != nil {
			return nil, fmt.Errorf("error creating global cache %v", err)
		}

		return &multiNamespaceCache{
			namespaces:    stringset.From(namespaces),
			multiCache:    multiCache,
			externalCache: externalCache,
		}, nil
	}
}

// Methods for multiNamespaceCache to conform to the cache.Informers interface.

func (c *multiNamespaceCache) GetInformer(ctx context.Context, obj client.Object) (cache.Informer, error) {
	return c.multiCache.GetInformer(ctx, obj)
}

func (c *multiNamespaceCache) GetInformerForKind(
	ctx context.Context, gvk schema.GroupVersionKind) (cache.Informer, error) {
	return c.multiCache.GetInformerForKind(ctx, gvk)
}

func (c *multiNamespaceCache) Start(ctx context.Context) error {
	// start global cache
	go func() {
		err := c.multiCache.Start(ctx)
		if err != nil {
			log.Error(err, "multi-cache failed to start")
		}
	}()

	go func() {
		err := c.externalCache.Start(ctx)
		if err != nil {
			log.Error(err, "external cache failed to start")
		}
	}()

	<-ctx.Done()
	return nil
}

func (c *multiNamespaceCache) WaitForCacheSync(ctx context.Context) bool {
	synced := true

	if !c.multiCache.WaitForCacheSync(ctx) {
		synced = false
	}

	if !c.externalCache.WaitForCacheSync(ctx) {
		synced = false
	}

	return synced
}

func (c *multiNamespaceCache) IndexField(
	ctx context.Context, obj client.Object, field string, extractValue client.IndexerFunc) error {
	return c.multiCache.IndexField(ctx, obj, field, extractValue)
}

// Methods for multiNamespaceCache to conform to the client.Reader interface.

func (c *multiNamespaceCache) Get(ctx context.Context, key client.ObjectKey, obj client.Object) error {
	// If the object we are looking for is in one of the watched namespaces just use
	// the multi-cache, otherwise we can use the global one
	if key.Namespace != "" && c.namespaces.Has(key.Namespace) {
		return c.multiCache.Get(ctx, key, obj)
	}

	return c.externalCache.Get(ctx, key, obj)
}

func (c *multiNamespaceCache) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	return c.multiCache.List(ctx, list, opts...)
}
