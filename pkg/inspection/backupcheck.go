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

package inspection

import (
	"context"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"

	kstonev1alpha2 "tkestack.io/kstone/pkg/apis/kstone/v1alpha2"
	"tkestack.io/kstone/pkg/backup"
	// import backup provider
	_ "tkestack.io/kstone/pkg/backup/providers"
	featureutil "tkestack.io/kstone/pkg/featureprovider/util"
	"tkestack.io/kstone/pkg/inspection/metrics"
)

// StatBackupFiles counts the number of backup files in the last day and
// transfer it to prometheus metrics
func (c *Server) StatBackupFiles(inspection *kstonev1alpha2.EtcdInspection) error {
	namespace, name := inspection.Namespace, inspection.Spec.ClusterName
	labels := map[string]string{
		"clusterName": name,
	}

	cluster, err := c.cli.KstoneV1alpha2().EtcdClusters(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	defer func() {
		if err != nil {
			featureutil.IncrFailedInspectionCounter(name, kstonev1alpha2.KStoneFeatureBackupCheck)
		}
	}()
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}

	// get backup config
	backupConfig, err := featureutil.GetBackupConfig(cluster)
	if err != nil {
		klog.Errorf("failed to get backup config,cluster %s,err is %v", cluster.Name, err)
		return err
	}

	// get specified backup storage provider
	storage, err := backup.GetBackupStorageProvider(string(backupConfig.StorageType), &backup.StorageConfig{
		KubeCli: c.kubeCli,
	})
	if err != nil {
		klog.Errorf("failed to get backup provider,cluster %s,err is %v", inspection.ClusterName, err)
		return err
	}
	objects, err := storage.List(cluster)
	if err != nil {
		klog.Errorf("failed to list backup files,cluster %s,err is %v", inspection.ClusterName, err)
		return err
	}

	actualFiles, err := storage.Stat(objects)
	if err != nil {
		klog.Errorf("failed to stat backup files,cluster %s,err is %v", inspection.ClusterName, err)
		return err
	}
	DesiredFiles := int(featureutil.OneDaySeconds / backupConfig.StoragePolicy.BackupIntervalInSecond)
	if DesiredFiles > backupConfig.StoragePolicy.MaxBackups {
		DesiredFiles = backupConfig.StoragePolicy.MaxBackups
	}
	failedFiles := DesiredFiles - actualFiles
	if failedFiles < 0 {
		failedFiles = 0
	}

	metrics.EtcdBackupFiles.With(labels).Set(float64(actualFiles))
	metrics.EtcdFailedBackupFiles.With(labels).Set(float64(failedFiles))
	return nil
}
