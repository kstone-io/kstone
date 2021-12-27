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

package monitor

import (
	"sync"

	kstonev1alpha1 "tkestack.io/kstone/pkg/apis/kstone/v1alpha1"
	"tkestack.io/kstone/pkg/featureprovider"
	featureutil "tkestack.io/kstone/pkg/featureprovider/util"
	"tkestack.io/kstone/pkg/monitor"
)

var (
	once     sync.Once
	instance *FeaturePrometheus
)

type FeaturePrometheus struct {
	name string
	prom *monitor.PrometheusMonitor
	ctx  *featureprovider.FeatureContext
}

const (
	ProviderName = string(kstonev1alpha1.KStoneFeatureMonitor)
)

func init() {
	featureprovider.RegisterFeatureFactory(
		ProviderName,
		func(cfg *featureprovider.FeatureContext) (featureprovider.Feature, error) {
			return initFeaturePrometheusInstance(cfg)
		},
	)
}

func initFeaturePrometheusInstance(ctx *featureprovider.FeatureContext) (featureprovider.Feature, error) {
	var err error
	once.Do(func() {
		instance = &FeaturePrometheus{
			name: ProviderName,
			ctx:  ctx,
		}
		instance.prom, err = monitor.NewPrometheusMonitor(ctx.ClientBuilder)
	})
	return instance, err
}

func (p *FeaturePrometheus) Equal(cluster *kstonev1alpha1.EtcdCluster) bool {
	if !featureutil.IsFeatureGateEnabled(cluster.ObjectMeta.Annotations, kstonev1alpha1.KStoneFeatureMonitor) {
		if cluster.Status.FeatureGatesStatus[kstonev1alpha1.KStoneFeatureMonitor] != featureutil.FeatureStatusDisabled {
			return p.prom.CheckEqualIfDisabled(cluster)
		}
		return true
	}
	return p.prom.CheckEqualIfEnabled(cluster)
}

func (p *FeaturePrometheus) Sync(cluster *kstonev1alpha1.EtcdCluster) error {
	if !featureutil.IsFeatureGateEnabled(cluster.ObjectMeta.Annotations, kstonev1alpha1.KStoneFeatureMonitor) {
		return p.prom.CleanPrometheusMonitor(cluster)
	}
	return p.prom.SyncPrometheusMonitor(cluster)
}

func (p *FeaturePrometheus) Do(inspection *kstonev1alpha1.EtcdInspection) error {
	return nil
}
