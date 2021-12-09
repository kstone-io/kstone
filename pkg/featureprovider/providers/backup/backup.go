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

package backup

import (
	"sync"

	kstoneapiv1 "tkestack.io/kstone/pkg/apis/kstone/v1alpha1"
	"tkestack.io/kstone/pkg/backup"
	"tkestack.io/kstone/pkg/featureprovider"
)

const (
	ProviderName = string(kstoneapiv1.KStoneFeatureBackup)
)

type Feature struct {
	name      string
	backupSvr *backup.Server
	once      sync.Once
	ctx       *featureprovider.FeatureContext
}

func init() {
	featureprovider.RegisterFeatureFactory(
		ProviderName,
		func(ctx *featureprovider.FeatureContext) (featureprovider.Feature, error) {
			return NewProviderBackup(ctx), nil
		},
	)
}

func NewProviderBackup(ctx *featureprovider.FeatureContext) featureprovider.Feature {
	return &Feature{
		name: ProviderName,
		ctx:  ctx,
	}
}

func (bak *Feature) Init() error {
	var err error
	bak.once.Do(func() {
		bak.backupSvr = &backup.Server{
			Clientbuilder: bak.ctx.Clientbuilder,
		}
		err = bak.backupSvr.Init()
	})
	return err
}

func (bak *Feature) Equal(cluster *kstoneapiv1.EtcdCluster) bool {
	return bak.backupSvr.Equal(cluster)
}

func (bak *Feature) Sync(cluster *kstoneapiv1.EtcdCluster) error {
	return bak.backupSvr.SyncEtcdBackup(cluster)
}

func (bak *Feature) Do(inspection *kstoneapiv1.EtcdInspection) error {
	return nil
}
