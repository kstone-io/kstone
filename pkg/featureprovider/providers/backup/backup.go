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

var (
	once     sync.Once
	instance *FeatureBackup
)

type FeatureBackup struct {
	name      string
	backupSvr *backup.Server
	ctx       *featureprovider.FeatureContext
}

func init() {
	featureprovider.RegisterFeatureFactory(
		ProviderName,
		func(ctx *featureprovider.FeatureContext) (featureprovider.Feature, error) {
			return initFeatureBackupInstance(ctx)
		},
	)
}

func initFeatureBackupInstance(ctx *featureprovider.FeatureContext) (featureprovider.Feature, error) {
	var err error
	once.Do(func() {
		instance = &FeatureBackup{
			name: ProviderName,
			ctx:  ctx,
		}
		err = instance.init()
	})
	return instance, err
}

func (bak *FeatureBackup) init() error {
	var err error
	bak.backupSvr = &backup.Server{
		Clientbuilder: bak.ctx.Clientbuilder,
	}
	err = bak.backupSvr.Init()
	return err
}

func (bak *FeatureBackup) Equal(cluster *kstoneapiv1.EtcdCluster) bool {
	return bak.backupSvr.Equal(cluster)
}

func (bak *FeatureBackup) Sync(cluster *kstoneapiv1.EtcdCluster) error {
	return bak.backupSvr.SyncEtcdBackup(cluster)
}

func (bak *FeatureBackup) Do(inspection *kstoneapiv1.EtcdInspection) error {
	return nil
}
