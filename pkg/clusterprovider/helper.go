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
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"
	"strconv"
	"strings"

	"go.etcd.io/etcd/client/pkg/v3/transport"
	"k8s.io/klog/v2"

	kstoneapiv1 "tkestack.io/kstone/pkg/apis/kstone/v1alpha1"
	"tkestack.io/kstone/pkg/etcd"
)

var DynamicClient dynamic.Interface

// Init inits DynamicClient
// TODO: fix me,remove DynamicClient
func Init(config *rest.Config) error {
	client, err := dynamic.NewForConfig(config)
	if err != nil {
		return err
	}
	DynamicClient = client
	return nil
}

// GetStorageMemberEndpoints get member of cluster status
func GetStorageMemberEndpoints(cluster *kstoneapiv1.EtcdCluster) []string {
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
	tls *transport.TLSInfo) ([]kstoneapiv1.MemberStatus, error) {
	etcdMembers := make([]kstoneapiv1.MemberStatus, 0)

	ca, cert, key := "", "", ""
	if tls != nil {
		ca, cert, key = tls.TrustedCAFile, tls.CertFile, tls.KeyFile
	}

	// GetMemberList
	client, err := etcd.NewClientv3(ca, cert, key, endpoints)
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
		memberVersion, memberStatus, memberRole := "", kstoneapiv1.MemberPhaseUnStarted, kstoneapiv1.EtcdMemberUnKnown
		var errors []string
		statusRsp, err := etcd.Status(extensionClientURL, client)
		if err == nil && statusRsp != nil {
			memberStatus = kstoneapiv1.MemberPhaseRunning
			memberVersion = statusRsp.Version
			if statusRsp.IsLearner {
				memberRole = kstoneapiv1.EtcdMemberLearner
			} else if statusRsp.Leader == m.ID {
				memberRole = kstoneapiv1.EtcdMemberLeader
			} else {
				memberRole = kstoneapiv1.EtcdMemberFollower
			}
			errors = statusRsp.Errors
		} else {
			klog.Errorf("failed to get member %s status,err is %v", extensionClientURL, err)
			errors = append(errors, err.Error())
		}

		etcdMembers = append(etcdMembers, kstoneapiv1.MemberStatus{
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
	members []kstoneapiv1.MemberStatus,
	tls *transport.TLSInfo) ([]kstoneapiv1.MemberStatus, kstoneapiv1.EtcdClusterPhase) {
	clusterStatus := kstoneapiv1.EtcdClusterRunning
	newMembers := make([]kstoneapiv1.MemberStatus, 0)
	for _, m := range members {
		healthy, err := etcd.MemberHealthy(m.ExtensionClientUrl, tls)
		if err != nil {
			m.Status = kstoneapiv1.MemberPhaseUnKnown
		} else {
			if healthy {
				m.Status = kstoneapiv1.MemberPhaseRunning
			} else {
				m.Status = kstoneapiv1.MemberPhaseUnHealthy
			}
		}

		if m.Status != kstoneapiv1.MemberPhaseRunning && clusterStatus == kstoneapiv1.EtcdClusterRunning {
			clusterStatus = kstoneapiv1.EtcdClusterUnhealthy
		}

		newMembers = append(newMembers, m)
	}

	return newMembers, clusterStatus
}
