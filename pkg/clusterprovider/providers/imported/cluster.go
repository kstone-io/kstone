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
	"go.etcd.io/etcd/client/pkg/v3/transport"
	kstoneapiv1 "tkestack.io/kstone/pkg/apis/kstone/v1alpha1"
	"tkestack.io/kstone/pkg/clusterprovider"
	"tkestack.io/kstone/pkg/controllers/util"
)

const (
	AnnoImportedURI = "importedAddr"
)

// EtcdClusterImported is the etcd cluster imported from kstone-dashboard
type EtcdClusterImported struct {
	name    kstoneapiv1.EtcdClusterType
	cluster *kstoneapiv1.EtcdCluster
}

// init registers an imported etcd cluster provider
func init() {
	clusterprovider.RegisterEtcdClusterFactory(
		kstoneapiv1.EtcdClusterImported,
		nil,
		nil,
		func(cluster *kstoneapiv1.EtcdCluster) (clusterprovider.Cluster, error) {
			return NewEtcdClusterImported(cluster)
		},
	)
}

// NewEtcdClusterImported generates imported etcd provider
func NewEtcdClusterImported(cluster *kstoneapiv1.EtcdCluster) (clusterprovider.Cluster, error) {
	return &EtcdClusterImported{
		name:    kstoneapiv1.EtcdClusterImported,
		cluster: cluster,
	}, nil
}

func (c *EtcdClusterImported) BeforeCreate() error {
	return nil
}

func (c *EtcdClusterImported) Create() error {
	return nil
}

func (c *EtcdClusterImported) AfterCreate() error {
	return nil
}

func (c *EtcdClusterImported) BeforeUpdate() error {
	return nil
}

func (c *EtcdClusterImported) Update() error {
	return nil
}

func (c *EtcdClusterImported) AfterUpdate() error {
	return nil
}

func (c *EtcdClusterImported) BeforeDelete() error {
	return nil
}

func (c *EtcdClusterImported) Delete() error {
	return nil
}

func (c *EtcdClusterImported) AfterDelete() error {
	return nil
}

func (c *EtcdClusterImported) Equal() (bool, error) {
	return true, nil
}

// Status gets the imported etcd cluster status
func (c *EtcdClusterImported) Status(tlsConfig *transport.TLSInfo) (kstoneapiv1.EtcdClusterStatus, error) {
	status := c.cluster.Status

	annotations := c.cluster.ObjectMeta.Annotations
	if annotations == nil {
		annotations = make(map[string]string)
	}

	endpoints := clusterprovider.GetStorageMemberEndpoints(c.cluster)

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
		c.cluster.Annotations[util.ClusterExtensionClientURL],
		tlsConfig,
	)
	if err != nil && len(members) == 0 {
		status.Phase = kstoneapiv1.EtcdClusterUnknown
		return status, err
	}

	status.Members, status.Phase = clusterprovider.GetEtcdClusterMemberStatus(members, tlsConfig)
	return status, err
}
