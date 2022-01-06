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

package backupcheck

import (
	"sync"

	kstonev1alpha1 "tkestack.io/kstone/pkg/apis/kstone/v1alpha1"
	"tkestack.io/kstone/pkg/featureprovider"
	"tkestack.io/kstone/pkg/inspection"
)

var (
	once     sync.Once
	instance *FeatureBackupCheck
)

type FeatureBackupCheck struct {
	name       string
	inspection *inspection.Server
	ctx        *featureprovider.FeatureContext
}

const (
	ProviderName = string(kstonev1alpha1.KStoneFeatureBackupCheck)
)

func init() {
	featureprovider.RegisterFeatureFactory(
		ProviderName,
		func(ctx *featureprovider.FeatureContext) (featureprovider.Feature, error) {
			return initFeatureBackupCheckInstance(ctx)
		},
	)
}

func initFeatureBackupCheckInstance(ctx *featureprovider.FeatureContext) (featureprovider.Feature, error) {
	var err error
	once.Do(func() {
		instance = &FeatureBackupCheck{
			name: ProviderName,
			ctx:  ctx,
		}
		instance.inspection, err = inspection.NewInspectionServer(ctx)
	})
	return instance, err
}

func (c *FeatureBackupCheck) Equal(cluster *kstonev1alpha1.EtcdCluster) bool {
	return c.inspection.Equal(cluster, kstonev1alpha1.KStoneFeatureBackupCheck)
}

func (c *FeatureBackupCheck) Sync(cluster *kstonev1alpha1.EtcdCluster) error {
	return c.inspection.Sync(cluster, kstonev1alpha1.KStoneFeatureBackupCheck)
}

func (c *FeatureBackupCheck) Do(inspection *kstonev1alpha1.EtcdInspection) error {
	return c.inspection.StatBackupFiles(inspection)
}
