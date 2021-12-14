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

package imported

import (
	"sync"

	"go.etcd.io/etcd/client/pkg/v3/transport"
	kstoneapiv1 "tkestack.io/kstone/pkg/apis/kstone/v1alpha1"
	"tkestack.io/kstone/pkg/clusterprovider"
	"tkestack.io/kstone/pkg/controllers/util"
)

const (
	AnnoImportedURI = "importedAddr"
)

var (
	once     sync.Once
	instance *EtcdClusterImported
)

// EtcdClusterImported is the etcd cluster imported from kstone-dashboard
type EtcdClusterImported struct {
	name kstoneapiv1.EtcdClusterType
	ctx  *clusterprovider.ClusterContext
}

// init registers an imported etcd cluster provider
func init() {
	clusterprovider.RegisterEtcdClusterFactory(
		kstoneapiv1.EtcdClusterImported,
		func(ctx *clusterprovider.ClusterContext) (clusterprovider.Cluster, error) {
			return initEtcdClusterImportedInstance(ctx)
		},
	)
}

func initEtcdClusterImportedInstance(ctx *clusterprovider.ClusterContext) (clusterprovider.Cluster, error) {
	once.Do(func() {
		instance = &EtcdClusterImported{
			name: kstoneapiv1.EtcdClusterImported,
			ctx: &clusterprovider.ClusterContext{
				Clientbuilder: ctx.Clientbuilder,
				Client:        ctx.Clientbuilder.DynamicClientOrDie(),
			},
		}
	})
	return instance, nil
}

func (c *EtcdClusterImported) BeforeCreate(etcdCluster *kstoneapiv1.EtcdCluster) error {
	return nil
}

func (c *EtcdClusterImported) Create(etcdCluster *kstoneapiv1.EtcdCluster) error {
	return nil
}

func (c *EtcdClusterImported) AfterCreate(etcdCluster *kstoneapiv1.EtcdCluster) error {
	return nil
}

func (c *EtcdClusterImported) BeforeUpdate(etcdCluster *kstoneapiv1.EtcdCluster) error {
	return nil
}

func (c *EtcdClusterImported) Update(etcdCluster *kstoneapiv1.EtcdCluster) error {
	return nil
}

func (c *EtcdClusterImported) AfterUpdate(etcdCluster *kstoneapiv1.EtcdCluster) error {
	return nil
}

func (c *EtcdClusterImported) BeforeDelete(etcdCluster *kstoneapiv1.EtcdCluster) error {
	return nil
}

func (c *EtcdClusterImported) Delete(etcdCluster *kstoneapiv1.EtcdCluster) error {
	return nil
}

func (c *EtcdClusterImported) AfterDelete(etcdCluster *kstoneapiv1.EtcdCluster) error {
	return nil
}

func (c *EtcdClusterImported) Equal(etcdCluster *kstoneapiv1.EtcdCluster) (bool, error) {
	return true, nil
}

// Status gets the imported etcd cluster status
func (c *EtcdClusterImported) Status(tlsConfig *transport.TLSInfo, etcdCluster *kstoneapiv1.EtcdCluster) (kstoneapiv1.EtcdClusterStatus, error) {
	status := etcdCluster.Status

	annotations := etcdCluster.ObjectMeta.Annotations
	if annotations == nil {
		annotations = make(map[string]string)
	}

	endpoints := clusterprovider.GetStorageMemberEndpoints(etcdCluster)

	if len(endpoints) == 0 {
		if addr, found := annotations[AnnoImportedURI]; found {
			endpoints = append(endpoints, addr)
			status.ServiceName = addr
		} else {
			status.Phase = kstoneapiv1.EtcdClusterUnknown
			return status, nil
		}
	}

	members, err := clusterprovider.GetRuntimeEtcdMembers(
		endpoints,
		etcdCluster.Annotations[util.ClusterExtensionClientURL],
		tlsConfig,
	)
	if err != nil && len(members) == 0 {
		status.Phase = kstoneapiv1.EtcdClusterUnknown
		return status, err
	}

	status.Members, status.Phase = clusterprovider.GetEtcdClusterMemberStatus(members, tlsConfig)
	return status, err
}
