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
	"k8s.io/klog/v2"

	kstonev1alpha1 "tkestack.io/kstone/pkg/apis/kstone/v1alpha1"
	"tkestack.io/kstone/pkg/etcd"
	"tkestack.io/kstone/pkg/inspection/metrics"
)

// CollectMemberHealthy collects the health of etcd, and
// transfer them to prometheus metrics
func (c *Server) CollectMemberHealthy(inspection *kstonev1alpha1.EtcdInspection) error {
	namespace, name := inspection.Namespace, inspection.Spec.ClusterName
	cluster, tlsConfig, err := c.GetEtcdClusterInfo(namespace, name)
	if err != nil {
		klog.Errorf("load tlsConfig failed, namespace is %s, name is %s, err is %v", namespace, name, err)
		return err
	}

	for _, m := range cluster.Status.Members {
		healthy, hErr := etcd.MemberHealthy(m.ExtensionClientUrl, tlsConfig)
		labels := map[string]string{
			"clusterName": cluster.Name,
			"endpoint":    m.Endpoint,
		}
		if hErr != nil || !healthy {
			metrics.EtcdEndpointHealthy.With(labels).Set(0)
		} else {
			metrics.EtcdEndpointHealthy.With(labels).Set(1)
		}
	}
	return nil
}
