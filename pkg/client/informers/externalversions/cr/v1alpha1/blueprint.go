/*
Copyright The Kubernetes Authors.

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

// Code generated by informer-gen. DO NOT EDIT.

package v1alpha1

import (
	"context"
	time "time"

	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"

	crv1alpha1 "github.com/kanisterio/kanister/pkg/apis/cr/v1alpha1"
	versioned "github.com/kanisterio/kanister/pkg/client/clientset/versioned"
	internalinterfaces "github.com/kanisterio/kanister/pkg/client/informers/externalversions/internalinterfaces"
	v1alpha1 "github.com/kanisterio/kanister/pkg/client/listers/cr/v1alpha1"
)

// BlueprintInformer provides access to a shared informer and lister for
// Blueprints.
type BlueprintInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1alpha1.BlueprintLister
}

type blueprintInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewBlueprintInformer constructs a new informer for Blueprint type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewBlueprintInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredBlueprintInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredBlueprintInformer constructs a new informer for Blueprint type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredBlueprintInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.CrV1alpha1().Blueprints(namespace).List(context.TODO(), options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.CrV1alpha1().Blueprints(namespace).Watch(context.TODO(), options)
			},
		},
		&crv1alpha1.Blueprint{},
		resyncPeriod,
		indexers,
	)
}

func (f *blueprintInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredBlueprintInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *blueprintInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&crv1alpha1.Blueprint{}, f.defaultInformer)
}

func (f *blueprintInformer) Lister() v1alpha1.BlueprintLister {
	return v1alpha1.NewBlueprintLister(f.Informer().GetIndexer())
}
