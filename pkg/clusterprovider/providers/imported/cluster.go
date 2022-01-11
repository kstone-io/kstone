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
	kstonev1alpha2 "tkestack.io/kstone/pkg/apis/kstone/v1alpha2"
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
	name kstonev1alpha2.EtcdClusterType
	ctx  *clusterprovider.ClusterContext
}

// init registers an imported etcd cluster provider
func init() {
	clusterprovider.RegisterEtcdClusterFactory(
		kstonev1alpha2.EtcdClusterImported,
		func(ctx *clusterprovider.ClusterContext) (clusterprovider.Cluster, error) {
			return initEtcdClusterImportedInstance(ctx)
		},
	)
}

func initEtcdClusterImportedInstance(ctx *clusterprovider.ClusterContext) (clusterprovider.Cluster, error) {
	once.Do(func() {
		instance = &EtcdClusterImported{
			name: kstonev1alpha2.EtcdClusterImported,
			ctx: &clusterprovider.ClusterContext{
				Clientbuilder: ctx.Clientbuilder,
				Client:        ctx.Clientbuilder.DynamicClientOrDie(),
			},
		}
	})
	return instance, nil
}

func (c *EtcdClusterImported) BeforeCreate(cluster *kstonev1alpha2.EtcdCluster) error {
	return nil
}

func (c *EtcdClusterImported) Create(cluster *kstonev1alpha2.EtcdCluster) error {
	return nil
}

func (c *EtcdClusterImported) AfterCreate(cluster *kstonev1alpha2.EtcdCluster) error {
	return nil
}

func (c *EtcdClusterImported) BeforeUpdate(cluster *kstonev1alpha2.EtcdCluster) error {
	return nil
}

func (c *EtcdClusterImported) Update(cluster *kstonev1alpha2.EtcdCluster) error {
	return nil
}

func (c *EtcdClusterImported) AfterUpdate(cluster *kstonev1alpha2.EtcdCluster) error {
	return nil
}

func (c *EtcdClusterImported) BeforeDelete(cluster *kstonev1alpha2.EtcdCluster) error {
	return nil
}

func (c *EtcdClusterImported) Delete(cluster *kstonev1alpha2.EtcdCluster) error {
	return nil
}

func (c *EtcdClusterImported) AfterDelete(cluster *kstonev1alpha2.EtcdCluster) error {
	return nil
}

func (c *EtcdClusterImported) Equal(cluster *kstonev1alpha2.EtcdCluster) (bool, error) {
	return true, nil
}

// Status gets the imported etcd cluster status
func (c *EtcdClusterImported) Status(tlsConfig *transport.TLSInfo, cluster *kstonev1alpha2.EtcdCluster) (kstonev1alpha2.EtcdClusterStatus, error) {
	status := cluster.Status

	annotations := cluster.ObjectMeta.Annotations
	if annotations == nil {
		annotations = make(map[string]string)
	}

	endpoints := clusterprovider.GetStorageMemberEndpoints(cluster)

	if len(endpoints) == 0 {
		if addr, found := annotations[AnnoImportedURI]; found {
			endpoints = append(endpoints, addr)
			status.ServiceName = addr
		} else {
			status.Phase = kstonev1alpha2.EtcdClusterUnknown
			return status, nil
		}
	}

	members, err := clusterprovider.GetRuntimeEtcdMembers(
		endpoints,
		cluster.Annotations[util.ClusterExtensionClientURL],
		tlsConfig,
	)
	if err != nil && len(members) == 0 {
		status.Phase = kstonev1alpha2.EtcdClusterUnknown
		return status, err
	}

	status.Members, status.Phase = clusterprovider.GetEtcdClusterMemberStatus(members, tlsConfig)
	return status, err
}
