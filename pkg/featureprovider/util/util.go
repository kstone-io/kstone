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

package util

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"k8s.io/klog/v2"

	kstonev1alpha1 "tkestack.io/kstone/pkg/apis/kstone/v1alpha1"
	"tkestack.io/kstone/pkg/backup"
)

const (
	FeatureStatusEnabled  = "enabled"
	FeatureStatusDisabled = "disabled"
)

type ConsistencyType string

const (
	ConsistencyKeyTotal             ConsistencyType = "keyTotal"
	ConsistencyRevision             ConsistencyType = "revision"
	ConsistencyIndex                ConsistencyType = "index"
	ConsistencyRaftRaftAppliedIndex ConsistencyType = "raftAppliedIndex"
	ConsistencyRaftIndex            ConsistencyType = "raftIndex"
)

const (
	OneDaySeconds = 24 * 60 * 60
)

func IsFeatureGateEnabled(annotations map[string]string, name kstonev1alpha1.KStoneFeature) bool {
	if gates, found := annotations[kstonev1alpha1.KStoneFeatureAnno]; found && gates != "" {
		featurelist := strings.Split(gates, ",")
		for _, item := range featurelist {
			features := strings.Split(item, "=")
			if len(features) != 2 {
				continue
			}

			enabled, _ := strconv.ParseBool(features[1])
			if kstonev1alpha1.KStoneFeature(features[0]) == name && enabled {
				return true
			}
		}
	}
	return false
}

func GetBackupConfig(cluster *kstonev1alpha1.EtcdCluster) (*backup.Config, error) {
	var err error
	cfg, found := cluster.Annotations[backup.AnnoBackupConfig]
	if !found {
		err = fmt.Errorf(
			"backup config not found, annotation key %s not exists, namespace is %s, name is %s",
			backup.AnnoBackupConfig,
			cluster.Namespace,
			cluster.Name,
		)
		klog.Errorf("failed to get backup config,cluster %s,err is %v", cluster.Name, err)
		return nil, err
	}

	backupConfig := &backup.Config{}
	err = json.Unmarshal([]byte(cfg), backupConfig)
	if err != nil {
		klog.Errorf(err.Error())
		return nil, err
	}
	return backupConfig, nil
}
