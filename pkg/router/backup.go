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

package router

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"k8s.io/klog/v2"

	"tkestack.io/kstone/pkg/backup"
)

// BackupList returns backup list
func BackupList(ctx *gin.Context) {
	cluster, _ := GetEtcdClusterInfo(ctx)

	// generate backup config
	strCfg, found := cluster.Annotations[backup.AnnoBackupConfig]
	if !found || strCfg == "" {
		err := fmt.Errorf(
			"backup config not found, annotation key %s not exists, namespace is %s, name is %s",
			backup.AnnoBackupConfig,
			cluster.Namespace,
			cluster.Name,
		)
		klog.Errorf(err.Error())
		ctx.JSON(http.StatusOK, []interface{}{})
		return
	}
	backupConfig := &backup.Config{}
	err := json.Unmarshal([]byte(strCfg), backupConfig)
	if err != nil {
		klog.Errorf(err.Error())
		ctx.JSON(http.StatusInternalServerError, err)
		return
	}

	// generate backup provider
	backupProvider, err := backup.GetBackupProvider(string(backupConfig.StorageType), &backup.ProviderConfig{
		Kubeconfig: "",
	})
	if err != nil {
		klog.Errorf(err.Error())
		ctx.JSON(http.StatusInternalServerError, err)
		return
	}
	resp, err := backupProvider.List(cluster)
	if err != nil {
		klog.Errorf(err.Error())
		ctx.JSON(http.StatusInternalServerError, err)
		return
	}
	ctx.JSON(http.StatusOK, resp)
}
