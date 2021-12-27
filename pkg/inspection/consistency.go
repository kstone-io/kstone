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

package inspection

import (
	"context"
	"sort"
	"strings"
	"sync"

	"go.etcd.io/etcd/client/pkg/v3/transport"
	"golang.org/x/sync/errgroup"
	"k8s.io/klog/v2"

	kstonev1alpha1 "tkestack.io/kstone/pkg/apis/kstone/v1alpha1"
	"tkestack.io/kstone/pkg/etcd"
	featureutil "tkestack.io/kstone/pkg/featureprovider/util"
	"tkestack.io/kstone/pkg/inspection/metrics"
)

type ConsistencyInfo struct {
	Path     string `json:"path,omitempty"`
	Interval int    `json:"interval,omitempty"`
}

// getEtcdConsistentMetadata gets the etcd consistent metadata.
func (c *Server) getEtcdConsistentMetadata(
	cluster *kstonev1alpha1.EtcdCluster,
	keyPrefix string,
	tls *transport.TLSInfo,
) (map[featureutil.ConsistencyType][]uint64, error) {

	var mu sync.Mutex
	endpointMetadata := make(map[featureutil.ConsistencyType][]uint64)

	ctx, cancel := context.WithTimeout(context.Background(), etcd.DefaultDialTimeout)
	g, ctx := errgroup.WithContext(ctx)
	defer cancel()

	for _, member := range cluster.Status.Members {
		member := member
		g.Go(func() error {
			ca, cert, key := "", "", ""
			if tls != nil {
				ca, cert, key = tls.TrustedCAFile, tls.CertFile, tls.KeyFile
			}
			backendStorage, extensionClientURL := etcd.EtcdV3Backend, member.ExtensionClientUrl
			if strings.HasPrefix(member.Version, "2") {
				backendStorage = etcd.EtcdV2Backend
			}
			backend, err := etcd.NewEtcdStatBackend(backendStorage)
			if err != nil {
				klog.Errorf("failed to get etcd stat backend,backend %s,err is %v", extensionClientURL, err)
				return err
			}
			err = backend.Init(ca, cert, key, extensionClientURL)
			if err != nil {
				klog.Errorf(
					"failed to get new etcd clientv3,cluster name is %s,endpoint is %s,err is %v",
					cluster.Name,
					extensionClientURL,
					err,
				)
				return err
			}
			defer backend.Close()
			metadata, err := backend.GetIndex(ctx, extensionClientURL)
			if err != nil {
				return err
			}
			totalKey, err := backend.GetTotalKeyNum(ctx, keyPrefix)
			if err != nil {
				return err
			}
			metadata[featureutil.ConsistencyKeyTotal] = totalKey

			mu.Lock()
			for t, v := range metadata {
				endpointMetadata[t] = append(endpointMetadata[t], v)
			}
			mu.Unlock()
			return nil
		})
	}
	if err := g.Wait(); err != nil {
		return endpointMetadata, err
	}
	return endpointMetadata, nil
}

// CollectClusterConsistentData collects the etcd metadata info, calculate the difference, and
// transfer them to prometheus metrics
func (c *Server) CollectClusterConsistentData(inspection *kstonev1alpha1.EtcdInspection) error {
	namespace, name := inspection.Namespace, inspection.Spec.ClusterName
	cluster, tlsConfig, err := c.GetEtcdClusterInfo(namespace, name)
	if err != nil {
		klog.Errorf("failed to load tls config, namespace is %s, name is %s, err is %v", namespace, name, err)
		return err
	}
	endpointMetadataDiff := make(map[featureutil.ConsistencyType]uint64)
	endpointMetadata, err := c.getEtcdConsistentMetadata(cluster, DefaultInspectionPath, tlsConfig)
	if err != nil {
		klog.Errorf("failed to getEtcdConsistentMetadata, etcd cluster %s, err is %v", cluster.Name, err)
	} else {
		for t, values := range endpointMetadata {
			sort.Slice(values, func(i, j int) bool { return values[i] < values[j] })
			endpointMetadataDiff[t] = values[len(values)-1] - values[0]
		}
	}
	labels := map[string]string{
		"clusterName": cluster.Name,
	}
	for t, v := range endpointMetadataDiff {
		switch t {
		case featureutil.ConsistencyKeyTotal:
			metrics.EtcdNodeDiffTotal.With(labels).Set(float64(v))
		case featureutil.ConsistencyRevision:
			metrics.EtcdNodeRevisionDiff.With(labels).Set(float64(v))
		case featureutil.ConsistencyIndex:
			metrics.EtcdNodeIndexDiff.With(labels).Set(float64(v))
		case featureutil.ConsistencyRaftRaftAppliedIndex:
			metrics.EtcdNodeRaftAppliedIndexDiff.With(labels).Set(float64(v))
		case featureutil.ConsistencyRaftIndex:
			metrics.EtcdNodeRaftIndexDiff.With(labels).Set(float64(v))
		}
	}
	return nil
}
