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
	"tkestack.io/kstone/pkg/apis/kstone/v1alpha1"
	"tkestack.io/kstone/pkg/controllers/util"
	"tkestack.io/kstone/pkg/etcd"
)

// Feature is an abstract, pluggable interface for cluster features.
type Feature interface {
	// Equal checks whether the feature needs to be updated
	Equal(cluster *v1alpha1.EtcdCluster) bool

	// Sync synchronizes the latest feature configuration
	Sync(cluster *v1alpha1.EtcdCluster) error

	// Do executes inspection tasks.
	Do(task *v1alpha1.EtcdInspection) error
}
type FeatureContext struct {
	ClientBuilder util.ClientBuilder
	TLSGetter     etcd.TLSGetter
}
