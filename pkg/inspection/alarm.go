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
	"strconv"

	"k8s.io/klog/v2"

	kstonev1alpha2 "tkestack.io/kstone/pkg/apis/kstone/v1alpha2"
	"tkestack.io/kstone/pkg/clusterprovider"
	featureutil "tkestack.io/kstone/pkg/featureprovider/util"
	"tkestack.io/kstone/pkg/inspection/metrics"
)

var alarmTypeList = []string{"NOSPACE", "CORRUPT"}

// CollectAlarmList collects the alarms of etcd, and
// transfer them to prometheus metrics
func (c *Server) CollectAlarmList(inspection *kstonev1alpha2.EtcdInspection) error {
	namespace, name := inspection.Namespace, inspection.Spec.ClusterName
	cluster, tlsConfig, err := c.GetEtcdClusterInfo(namespace, name)
	defer func() {
		if err != nil {
			featureutil.IncrFailedInspectionCounter(name, kstonev1alpha2.KStoneFeatureAlarm)
		}
	}()
	if err != nil {
		klog.Errorf("load tlsConfig failed, namespace is %s, name is %s, err is %v", namespace, name, err)
		return err
	}

	alarms, err := clusterprovider.GetEtcdAlarms([]string{cluster.Status.ServiceName}, tlsConfig)
	if err != nil {
		return err
	}

	for _, m := range cluster.Status.Members {
		if len(alarms) == 0 {
			cleanAllAlarmMetrics(cluster.Name, m.Endpoint)
		}
		for _, a := range alarms {
			if m.MemberId == strconv.FormatUint(a.MemberID, 10) {
				labels := map[string]string{
					"clusterName": cluster.Name,
					"endpoint":    m.Endpoint,
					"alarmType":   a.AlarmType,
				}
				metrics.EtcdEndpointAlarm.With(labels).Set(1)
			}
		}
	}
	return nil
}

// cleanAllAlarmMetrics clear all alarm metrics by cluster
func cleanAllAlarmMetrics(clusterName, endpoint string) {
	for _, t := range alarmTypeList {
		metrics.EtcdEndpointAlarm.With(map[string]string{
			"clusterName": clusterName,
			"endpoint":    endpoint,
			"alarmType":   t,
		}).Set(0)
	}
}
