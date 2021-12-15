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

package clusterprovider

import (
	"go.etcd.io/etcd/client/pkg/v3/transport"
	"k8s.io/client-go/dynamic"

	kstoneapiv1 "tkestack.io/kstone/pkg/apis/kstone/v1alpha1"
	"tkestack.io/kstone/pkg/controllers/util"
)

// Cluster is an abstract, pluggable interface for etcd clusters.
type Cluster interface {

	// BeforeCreate does some things before creating the cluster
	BeforeCreate(cluster *kstoneapiv1.EtcdCluster) error
	// Create creates the cluster
	Create(cluster *kstoneapiv1.EtcdCluster) error
	// AfterCreate does some things after creating the cluster
	AfterCreate(cluster *kstoneapiv1.EtcdCluster) error

	// BeforeUpdate does some things before updating the cluster
	BeforeUpdate(cluster *kstoneapiv1.EtcdCluster) error
	// Update updates the cluster
	Update(cluster *kstoneapiv1.EtcdCluster) error
	// AfterUpdate does some things after updating the cluster
	AfterUpdate(cluster *kstoneapiv1.EtcdCluster) error

	// BeforeDelete does some things before deleting the cluster
	BeforeDelete(cluster *kstoneapiv1.EtcdCluster) error
	// Delete deletes the cluster
	Delete(cluster *kstoneapiv1.EtcdCluster) error
	// AfterDelete does some things after deleting the cluster
	AfterDelete(cluster *kstoneapiv1.EtcdCluster) error

	// Equal checks whether the cluster needs to be updated
	Equal(cluster *kstoneapiv1.EtcdCluster) (bool, error)

	// Status gets the cluster status
	Status(tlsConfig *transport.TLSInfo, cluster *kstoneapiv1.EtcdCluster) (kstoneapiv1.EtcdClusterStatus, error)
}

type ClusterContext struct {
	Clientbuilder util.ClientBuilder
	Client        dynamic.Interface
}
