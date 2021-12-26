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

package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	EtcdNodeDiffTotal = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "kstone",
		Subsystem: "inspection",
		Name:      "etcd_node_diff_total",
		Help:      "total etcd node diff key",
	}, []string{"clusterName"})

	EtcdEndpointHealthy = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "kstone",
		Subsystem: "inspection",
		Name:      "etcd_endpoint_healthy",
		Help:      "The healthy of etcd member",
	}, []string{"clusterName", "endpoint"})

	EtcdRequestTotal = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "kstone",
		Subsystem: "inspection",
		Name:      "etcd_request_total",
		Help:      "The total number of etcd requests",
	}, []string{"clusterName", "grpcMethod", "etcdPrefix", "resourceName"})

	EtcdKeyTotal = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "kstone",
		Subsystem: "inspection",
		Name:      "etcd_key_total",
		Help:      "The total number of etcd key",
	}, []string{"clusterName", "etcdPrefix", "resourceName"})

	EtcdEndpointAlarm = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "kstone",
		Subsystem: "inspection",
		Name:      "etcd_endpoint_alarm",
		Help:      "The alarm of etcd member",
	}, []string{"clusterName", "endpoint", "alarmType"})
)

func init() {
	prometheus.MustRegister(EtcdNodeDiffTotal)
	prometheus.MustRegister(EtcdEndpointHealthy)
	prometheus.MustRegister(EtcdRequestTotal)
	prometheus.MustRegister(EtcdKeyTotal)
	prometheus.MustRegister(EtcdEndpointAlarm)
}
