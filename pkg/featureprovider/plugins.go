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

package featureprovider

import (
	"errors"
	"sync"

	"k8s.io/klog/v2"
)

var (
	mutex                sync.Mutex
	EtcdFeatureProviders = make(map[string]FeatureFactory)
)

type FeatureFactory func(cfg *FeatureContext) (Feature, error)

// RegisterFeatureFactory registers the specified feature provider
func RegisterFeatureFactory(name string, factory FeatureFactory) {
	mutex.Lock()
	defer mutex.Unlock()

	if _, found := EtcdFeatureProviders[name]; found {
		klog.V(2).Infof("feature provider:%s was registered twice", name)
	}

	klog.V(2).Infof("feature provider:%s", name)
	EtcdFeatureProviders[name] = factory
}

// GetFeatureProvider gets the specified feature provider
func GetFeatureProvider(name string, ctx *FeatureContext) (Feature, error) {
	mutex.Lock()
	defer mutex.Unlock()
	f, found := EtcdFeatureProviders[name]

	klog.V(1).Infof("get provider name %s,status:%t", name, found)
	if !found {
		return nil, errors.New("fatal error,feature provider not found")
	}
	return f(ctx)
}

// ListFeatureProvider lists all feature provider
func ListFeatureProvider() []string {
	var features []string
	mutex.Lock()
	defer mutex.Unlock()
	for feature := range EtcdFeatureProviders {
		features = append(features, feature)
	}
	return features
}
