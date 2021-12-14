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

package consistency

import (
	"sync"

	kstoneapiv1 "tkestack.io/kstone/pkg/apis/kstone/v1alpha1"
	"tkestack.io/kstone/pkg/featureprovider"
	"tkestack.io/kstone/pkg/inspection"
)

const (
	ProviderName = string(kstoneapiv1.KStoneFeatureConsistency)
)

var (
	once     sync.Once
	instance *FeatureConsistency
)

type FeatureConsistency struct {
	name       string
	inspection *inspection.Server
	ctx        *featureprovider.FeatureContext
}

func init() {
	featureprovider.RegisterFeatureFactory(
		ProviderName,
		func(ctx *featureprovider.FeatureContext) (featureprovider.Feature, error) {
			return initFeatureConsistencyInstance(ctx)
		},
	)
}

func initFeatureConsistencyInstance(ctx *featureprovider.FeatureContext) (featureprovider.Feature, error) {
	var err error
	once.Do(func() {
		instance = &FeatureConsistency{
			name: ProviderName,
			ctx:  ctx,
		}
		err = instance.init()
	})
	return instance, err
}

func (c *FeatureConsistency) init() error {
	var err error
	c.inspection = &inspection.Server{
		Clientbuilder: c.ctx.Clientbuilder,
	}
	err = c.inspection.Init()
	return err
}

func (c *FeatureConsistency) Equal(cluster *kstoneapiv1.EtcdCluster) bool {
	return c.inspection.IsNotFound(cluster, ProviderName)
}

func (c *FeatureConsistency) Sync(cluster *kstoneapiv1.EtcdCluster) error {
	return c.inspection.AddConsistencyTask(cluster, ProviderName)
}

func (c *FeatureConsistency) Do(inspection *kstoneapiv1.EtcdInspection) error {
	return c.inspection.CollectMemberConsistency(inspection)
}
