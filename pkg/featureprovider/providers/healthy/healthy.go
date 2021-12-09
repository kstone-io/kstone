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

type FeatureHealthy struct {
	name       string
	inspection *inspection.Server
	once       sync.Once
	ctx        *featureprovider.FeatureContext
}

const (
	ProviderName = string(kstoneapiv1.KStoneFeatureHealthy)
)

func init() {
	featureprovider.RegisterFeatureFactory(
		ProviderName,
		func(ctx *featureprovider.FeatureContext) (featureprovider.Feature, error) {
			return NewFeatureHealty(ctx)
		},
	)
}

func NewFeatureHealty(ctx *featureprovider.FeatureContext) (featureprovider.Feature, error) {
	return &FeatureHealthy{
		name: ProviderName,
		ctx:  ctx,
	}, nil
}

func (c *FeatureHealthy) Init() error {
	var err error
	c.once.Do(func() {
		c.inspection = &inspection.Server{
			Clientbuilder: c.ctx.Clientbuilder,
		}
		err = c.inspection.Init()
	})
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
