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

	kstonev1alpha2 "tkestack.io/kstone/pkg/apis/kstone/v1alpha2"
	"tkestack.io/kstone/pkg/backup"
	"tkestack.io/kstone/pkg/featureprovider"
	featureutil "tkestack.io/kstone/pkg/featureprovider/util"
)

const (
	ProviderName = string(kstonev1alpha2.KStoneFeatureBackup)
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
		instance.backupSvr, err = backup.NewBackupServer(ctx.ClientBuilder)
	})
	return instance, err
}

func (bak *FeatureBackup) Equal(cluster *kstonev1alpha2.EtcdCluster) bool {
	if !featureutil.IsFeatureGateEnabled(cluster.ObjectMeta.Annotations, kstonev1alpha2.KStoneFeatureBackup) {
		if cluster.Status.FeatureGatesStatus[kstonev1alpha2.KStoneFeatureBackup] != featureutil.FeatureStatusDisabled {
			return bak.backupSvr.CheckEqualIfDisabled(cluster)
		}
		return true
	}
	return bak.backupSvr.CheckEqualIfEnabled(cluster)
}

func (bak *FeatureBackup) Sync(cluster *kstonev1alpha2.EtcdCluster) error {
	if !featureutil.IsFeatureGateEnabled(cluster.ObjectMeta.Annotations, kstonev1alpha2.KStoneFeatureBackup) {
		return bak.backupSvr.CleanBackup(cluster)
	}
	return bak.backupSvr.SyncBackup(cluster)
}

func (bak *FeatureBackup) Do(inspection *kstonev1alpha2.EtcdInspection) error {
	return nil
}
