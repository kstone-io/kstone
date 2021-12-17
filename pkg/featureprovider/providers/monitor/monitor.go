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

	kstoneapiv1 "tkestack.io/kstone/pkg/apis/kstone/v1alpha1"
	"tkestack.io/kstone/pkg/featureprovider"
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
	ProviderName = string(kstoneapiv1.KStoneFeatureMonitor)
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
		err = instance.init()
	})
	return instance, err
}

func (p *FeaturePrometheus) init() error {
	var err error
	p.prom = &monitor.PrometheusMonitor{
		ClientBuilder: p.ctx.Clientbuilder,
	}
	err = p.prom.Init()
	return err
}

func (p *FeaturePrometheus) Equal(cluster *kstoneapiv1.EtcdCluster) bool {
	if len(cluster.Status.Members) == 0 {
		return false
	}
	return p.prom.Equal(cluster)
}

func (p *FeaturePrometheus) Sync(cluster *kstoneapiv1.EtcdCluster) error {
	return p.prom.SyncPrometheusMonitor(cluster)
}

func (p *FeaturePrometheus) Do(inspection *kstoneapiv1.EtcdInspection) error {
	return nil
}
