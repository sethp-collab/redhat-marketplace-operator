// Copyright 2020 The redhat-marketplace-operator Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// Code generated by informer-gen. DO NOT EDIT.

package v1alpha1

import (
	"context"
	time "time"

	marketplacev1alpha1 "github.com/redhat-marketplace/redhat-marketplace-operator/pkg/apis/marketplace/v1alpha1"
	versioned "github.com/redhat-marketplace/redhat-marketplace-operator/pkg/generated/clientset/versioned"
	internalinterfaces "github.com/redhat-marketplace/redhat-marketplace-operator/pkg/generated/informers/externalversions/internalinterfaces"
	v1alpha1 "github.com/redhat-marketplace/redhat-marketplace-operator/pkg/generated/listers/marketplace/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
	watch "k8s.io/apimachinery/pkg/watch"
	cache "k8s.io/client-go/tools/cache"
)

// MeterDefinitionInformer provides access to a shared informer and lister for
// MeterDefinitions.
type MeterDefinitionInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() v1alpha1.MeterDefinitionLister
}

type meterDefinitionInformer struct {
	factory          internalinterfaces.SharedInformerFactory
	tweakListOptions internalinterfaces.TweakListOptionsFunc
	namespace        string
}

// NewMeterDefinitionInformer constructs a new informer for MeterDefinition type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewMeterDefinitionInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers) cache.SharedIndexInformer {
	return NewFilteredMeterDefinitionInformer(client, namespace, resyncPeriod, indexers, nil)
}

// NewFilteredMeterDefinitionInformer constructs a new informer for MeterDefinition type.
// Always prefer using an informer factory to get a shared informer instead of getting an independent
// one. This reduces memory footprint and number of connections to the server.
func NewFilteredMeterDefinitionInformer(client versioned.Interface, namespace string, resyncPeriod time.Duration, indexers cache.Indexers, tweakListOptions internalinterfaces.TweakListOptionsFunc) cache.SharedIndexInformer {
	return cache.NewSharedIndexInformer(
		&cache.ListWatch{
			ListFunc: func(options v1.ListOptions) (runtime.Object, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.MarketplaceV1alpha1().MeterDefinitions(namespace).List(context.TODO(), options)
			},
			WatchFunc: func(options v1.ListOptions) (watch.Interface, error) {
				if tweakListOptions != nil {
					tweakListOptions(&options)
				}
				return client.MarketplaceV1alpha1().MeterDefinitions(namespace).Watch(context.TODO(), options)
			},
		},
		&marketplacev1alpha1.MeterDefinition{},
		resyncPeriod,
		indexers,
	)
}

func (f *meterDefinitionInformer) defaultInformer(client versioned.Interface, resyncPeriod time.Duration) cache.SharedIndexInformer {
	return NewFilteredMeterDefinitionInformer(client, f.namespace, resyncPeriod, cache.Indexers{cache.NamespaceIndex: cache.MetaNamespaceIndexFunc}, f.tweakListOptions)
}

func (f *meterDefinitionInformer) Informer() cache.SharedIndexInformer {
	return f.factory.InformerFor(&marketplacev1alpha1.MeterDefinition{}, f.defaultInformer)
}

func (f *meterDefinitionInformer) Lister() v1alpha1.MeterDefinitionLister {
	return v1alpha1.NewMeterDefinitionLister(f.Informer().GetIndexer())
}
