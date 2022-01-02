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
	"encoding/json"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/klog/v2"

	kstonev1alpha1 "tkestack.io/kstone/pkg/apis/kstone/v1alpha1"
	"tkestack.io/kstone/pkg/backup"
	// import backup provider
	_ "tkestack.io/kstone/pkg/backup/providers"
	featureutil "tkestack.io/kstone/pkg/featureprovider/util"
	"tkestack.io/kstone/pkg/inspection/metrics"
)

// StatBackupFiles counts the number of backup files in the last day and
// transfer it to prometheus metrics
func (c *Server) StatBackupFiles(inspection *kstonev1alpha1.EtcdInspection) error {
	namespace, name := inspection.Namespace, inspection.Spec.ClusterName
	labels := map[string]string{
		"clusterName": name,
	}
	cluster, err := c.cli.KstoneV1alpha1().EtcdClusters(namespace).Get(context.TODO(), name, metav1.GetOptions{})

	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil
		}
		return err
	}

	// generate backup config
	strCfg, found := cluster.Annotations[backup.AnnoBackupConfig]
	if !found || strCfg == "" {
		err = fmt.Errorf(
			"backup config not found, annotation key %s not exists, namespace is %s, name is %s",
			backup.AnnoBackupConfig,
			cluster.Namespace,
			cluster.Name,
		)
		klog.Errorf("failed to get backup config,cluster %s,err is %v", inspection.ClusterName, err)
		return err
	}
	backupConfig := &backup.Config{}
	err = json.Unmarshal([]byte(strCfg), backupConfig)
	if err != nil {
		klog.Errorf("failed to unmarshal backup config,cluster %s,err is %v", inspection.ClusterName, err)
		return err
	}

	// generate backup provider
	backupProvider, err := backup.GetBackupProvider(string(backupConfig.StorageType), &backup.ProviderConfig{
		Kubeconfig: "",
	})
	if err != nil {
		klog.Errorf("failed to get backup provider,cluster %s,err is %v", inspection.ClusterName, err)
		return err
	}
	resp, err := backupProvider.List(cluster)
	if err != nil {
		klog.Errorf("failed to list backup files,cluster %s,err is %v", inspection.ClusterName, err)
		return err
	}

	actualFiles, err := backupProvider.StatBackupFiles(resp)
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
