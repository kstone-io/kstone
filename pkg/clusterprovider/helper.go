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

package clusterprovider

import (
	"fmt"
	"strconv"
	"strings"

	"k8s.io/klog/v2"

	kstonev1alpha2 "tkestack.io/kstone/pkg/apis/kstone/v1alpha2"
	"tkestack.io/kstone/pkg/etcd"
)

type EtcdAlarm struct {
	MemberID  uint64
	AlarmType string
}

// GetStorageMemberEndpoints get member of cluster status
func GetStorageMemberEndpoints(cluster *kstonev1alpha2.EtcdCluster) []string {
	members := cluster.Status.Members
	endpoints := make([]string, 0)
	if len(members) == 0 {
		return endpoints
	}

	for _, m := range members {
		endpoints = append(endpoints, m.ExtensionClientUrl)
	}
	return endpoints
}

// populateExtensionCientURLMap generate extensionClientURLs map
func populateExtensionCientURLMap(extensionClientURLs string) (map[string]string, error) {
	urlMap := make(map[string]string)
	if extensionClientURLs == "" {
		return nil, nil
	}
	items := strings.Split(extensionClientURLs, ",")
	for i := 0; i < len(items); i++ {
		eps := strings.Split(items[i], "->")
		if len(eps) == 2 {
			urlMap[eps[0]] = eps[1]
		} else {
			return urlMap, fmt.Errorf("invalid extensionClientURLs %s", items[i])
		}
	}
	return urlMap, nil
}

// GetRuntimeEtcdMembers get members of etcd
func GetRuntimeEtcdMembers(
	endpoints []string,
	extensionClientURLs string,
	config *etcd.ClientConfig) ([]kstonev1alpha2.MemberStatus, error) {
	etcdMembers := make([]kstonev1alpha2.MemberStatus, 0)

	config.Endpoints = endpoints

	// GetMemberList
	client, err := etcd.NewClientv3(config)
	if err != nil {
		klog.Errorf("failed to get new etcd clientv3,err is %v ", err)
		return etcdMembers, err
	}
	defer client.Close()

	memberRsp, err := etcd.MemberList(client)
	if err != nil {
		klog.Errorf("failed to get member list, endpoints is %s,err is %v", endpoints, err)
		return etcdMembers, err
	}

	extensionClientURLMap, err := populateExtensionCientURLMap(extensionClientURLs)
	if err != nil {
		klog.Errorf("failed to populate extension clientURL,err is %v", err)
		return etcdMembers, err
	}

	for _, m := range memberRsp.Members {
		// parse url
		if m.ClientURLs == nil {
			continue
		}
		items := strings.Split(m.ClientURLs[0], ":")
		endPoint := strings.TrimPrefix(items[1], "//")

		extensionClientURL := m.ClientURLs[0]
		if extensionClientURLs != "" {
			var ep string
			if strings.HasPrefix(m.ClientURLs[0], "https://") {
				ep = strings.TrimPrefix(m.ClientURLs[0], "https://")
				if _, ok := extensionClientURLMap[ep]; ok {
					extensionClientURL = "https://" + extensionClientURLMap[ep]
				}
			} else {
				ep = strings.TrimPrefix(m.ClientURLs[0], "http://")
				if _, ok := extensionClientURLMap[ep]; ok {
					extensionClientURL = "http://" + extensionClientURLMap[ep]
				}
			}
		}

		// default info
		memberVersion, memberStatus, memberRole := "", kstonev1alpha2.MemberPhaseUnStarted, kstonev1alpha2.EtcdMemberUnKnown
		var errors []string
		statusRsp, err := etcd.Status(extensionClientURL, client)
		if err == nil && statusRsp != nil {
			memberStatus = kstonev1alpha2.MemberPhaseRunning
			memberVersion = statusRsp.Version
			if statusRsp.IsLearner {
				memberRole = kstonev1alpha2.EtcdMemberLearner
			} else if statusRsp.Leader == m.ID {
				memberRole = kstonev1alpha2.EtcdMemberLeader
			} else {
				memberRole = kstonev1alpha2.EtcdMemberFollower
			}
			errors = statusRsp.Errors
		} else {
			klog.Errorf("failed to get member %s status,err is %v", extensionClientURL, err)
			errors = append(errors, err.Error())
		}

		etcdMembers = append(etcdMembers, kstonev1alpha2.MemberStatus{
			Name:               m.Name,
			MemberId:           strconv.FormatUint(m.ID, 10),
			ClientUrl:          m.ClientURLs[0],
			ExtensionClientUrl: extensionClientURL,
			Role:               memberRole,
			Status:             memberStatus,
			Endpoint:           endPoint,
			Port:               items[2],
			Version:            memberVersion,
			Errors:             errors,
		})
	}

	return etcdMembers, nil
}

// GetEtcdClusterMemberStatus check healthy of cluster and member
func GetEtcdClusterMemberStatus(
	members []kstonev1alpha2.MemberStatus,
	config *etcd.ClientConfig) ([]kstonev1alpha2.MemberStatus, kstonev1alpha2.EtcdClusterPhase) {
	clusterStatus := kstonev1alpha2.EtcdClusterRunning
	newMembers := make([]kstonev1alpha2.MemberStatus, 0)
	for _, m := range members {
		healthy, err := etcd.MemberHealthy(m.ExtensionClientUrl, config)
		if err != nil {
			m.Status = kstonev1alpha2.MemberPhaseUnKnown
		} else {
			if healthy {
				m.Status = kstonev1alpha2.MemberPhaseRunning
			} else {
				m.Status = kstonev1alpha2.MemberPhaseUnHealthy
			}
		}

		if m.Status != kstonev1alpha2.MemberPhaseRunning && clusterStatus == kstonev1alpha2.EtcdClusterRunning {
			clusterStatus = kstonev1alpha2.EtcdClusterUnhealthy
		}

		newMembers = append(newMembers, m)
	}

	return newMembers, clusterStatus
}

// GetEtcdAlarms get alarm list of etcd
func GetEtcdAlarms(
	endpoints []string,
	config *etcd.ClientConfig) ([]EtcdAlarm, error) {
	etcdAlarms := make([]EtcdAlarm, 0)

	config.Endpoints = endpoints

	client, err := etcd.NewClientv3(config)
	if err != nil {
		klog.Errorf("failed to get new etcd clientv3, err is %v ", err)
		return etcdAlarms, err
	}
	defer client.Close()

	alarmRsp, err := etcd.AlarmList(client)
	if err != nil {
		klog.Errorf("failed to get alarm list, err is %v", err)
		return etcdAlarms, err
	}

	for _, a := range alarmRsp.Alarms {
		etcdAlarms = append(etcdAlarms, EtcdAlarm{
			MemberID:  a.MemberID,
			AlarmType: a.Alarm.String(),
		})
	}
	return etcdAlarms, nil
}
