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

package healthy

import (
	"sync"

	kstoneapiv1 "tkestack.io/kstone/pkg/apis/kstone/v1alpha1"
	"tkestack.io/kstone/pkg/featureprovider"
	"tkestack.io/kstone/pkg/inspection"
)

var (
	once     sync.Once
	instance *FeatureHealthy
)

type FeatureHealthy struct {
	name       string
	inspection *inspection.Server
	ctx        *featureprovider.FeatureContext
}

const (
	ProviderName = string(kstoneapiv1.KStoneFeatureHealthy)
)

func init() {
	featureprovider.RegisterFeatureFactory(
		ProviderName,
		func(ctx *featureprovider.FeatureContext) (featureprovider.Feature, error) {
			return initFeatureHealthyInstance(ctx)
		},
	)
}

func initFeatureHealthyInstance(ctx *featureprovider.FeatureContext) (featureprovider.Feature, error) {
	var err error
	once.Do(func() {
		instance = &FeatureHealthy{
			name: ProviderName,
			ctx:  ctx,
		}
		err = instance.init()
	})
	return instance, err
}

func (c *FeatureHealthy) init() error {
	var err error
	c.inspection = &inspection.Server{
		Clientbuilder: c.ctx.Clientbuilder,
	}
	err = c.inspection.Init()
	return err
}

func (c *FeatureHealthy) Equal(cluster *kstoneapiv1.EtcdCluster) bool {
	return c.inspection.IsNotFound(cluster, ProviderName)
}

func (c *FeatureHealthy) Sync(cluster *kstoneapiv1.EtcdCluster) error {
	return c.inspection.AddHealthyTask(cluster, ProviderName)
}

func (c *FeatureHealthy) Do(inspection *kstoneapiv1.EtcdInspection) error {
	return c.inspection.CollectMemberHealthy(inspection)
}
