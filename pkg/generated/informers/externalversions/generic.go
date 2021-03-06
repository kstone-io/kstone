/*
 * Tencent is pleased to support the open source community by making TKEStack
 * available.
 *
 * Copyright (C) 2012-2023 Tencent. All Rights Reserved.
 *
 * Licensed under the Apache License, Version 2.0 (the "License"); you may not use
 * this file except in compliance with the License. You may obtain a copy of the
 * License at
 *
 * https://opensource.org/licenses/Apache-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
 * WARRANTIES OF ANY KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations under the License.
 */

// Code generated by informer-gen. DO NOT EDIT.

package externalversions

import (
	"fmt"

	schema "k8s.io/apimachinery/pkg/runtime/schema"
	cache "k8s.io/client-go/tools/cache"
	v1alpha1 "tkestack.io/kstone/pkg/apis/kstone/v1alpha1"
	v1alpha2 "tkestack.io/kstone/pkg/apis/kstone/v1alpha2"
)

// GenericInformer is type of SharedIndexInformer which will locate and delegate to other
// sharedInformers based on type
type GenericInformer interface {
	Informer() cache.SharedIndexInformer
	Lister() cache.GenericLister
}

type genericInformer struct {
	informer cache.SharedIndexInformer
	resource schema.GroupResource
}

// Informer returns the SharedIndexInformer.
func (f *genericInformer) Informer() cache.SharedIndexInformer {
	return f.informer
}

// Lister returns the GenericLister.
func (f *genericInformer) Lister() cache.GenericLister {
	return cache.NewGenericLister(f.Informer().GetIndexer(), f.resource)
}

// ForResource gives generic access to a shared informer of the matching type
// TODO extend this to unknown resources with a client pool
func (f *sharedInformerFactory) ForResource(resource schema.GroupVersionResource) (GenericInformer, error) {
	switch resource {
	// Group=kstone.tkestack.io, Version=v1alpha1
	case v1alpha1.SchemeGroupVersion.WithResource("etcdclusters"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Kstone().V1alpha1().EtcdClusters().Informer()}, nil
	case v1alpha1.SchemeGroupVersion.WithResource("etcdinspections"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Kstone().V1alpha1().EtcdInspections().Informer()}, nil

		// Group=kstone.tkestack.io, Version=v1alpha2
	case v1alpha2.SchemeGroupVersion.WithResource("etcdclusters"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Kstone().V1alpha2().EtcdClusters().Informer()}, nil
	case v1alpha2.SchemeGroupVersion.WithResource("etcdinspections"):
		return &genericInformer{resource: resource.GroupResource(), informer: f.Kstone().V1alpha2().EtcdInspections().Informer()}, nil

	}

	return nil, fmt.Errorf("no informer found for %v", resource)
}
