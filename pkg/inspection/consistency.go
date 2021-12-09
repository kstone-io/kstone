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
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"

	"go.etcd.io/etcd/client/pkg/v3/transport"
	"k8s.io/klog/v2"

	kstoneapiv1 "tkestack.io/kstone/pkg/apis/kstone/v1alpha1"
	"tkestack.io/kstone/pkg/etcd"
	"tkestack.io/kstone/pkg/inspection/metrics"
)

const (
	CruiseConsistencyAnno = "cruiseConsistency"
)

type ConsistencyInfo struct {
	Path     string `json:"path,omitempty"`
	Interval int    `json:"interval,omitempty"`
}

// AddConsistencyTask adds consistency inspection task
func (c *Server) AddConsistencyTask(cluster *kstoneapiv1.EtcdCluster, cruiseType string) error {
	task, err := c.initInspectionTask(cluster, cruiseType)
	if err != nil {
		return err
	}

	annotations := cluster.ObjectMeta.Annotations
	if annotations == nil {
		annotations = make(map[string]string)
	}

	if _, found := annotations[CruiseConsistencyAnno]; found {
		taskAnno := task.ObjectMeta.Annotations
		if taskAnno == nil {
			taskAnno = make(map[string]string)
			taskAnno[CruiseConsistencyAnno] = annotations[CruiseConsistencyAnno]
			task.ObjectMeta.Annotations = taskAnno
		}
	}

	_, err = c.CreateEtcdInspection(task)
	if err != nil {
		return err
	}

	return nil
}

// getEtcdNodeKeyDiff checks the diff of node data
func (c *Server) getEtcdNodeKeyDiff(
	cluster *kstoneapiv1.EtcdCluster,
	keyPrefix string,
	tls *transport.TLSInfo,
) (chan uint64, chan error) {
	memberCnt := len(cluster.Status.Members)
	ch := make(chan uint64, memberCnt)
	errch := make(chan error, memberCnt)
	var wg sync.WaitGroup
	wg.Add(memberCnt)
	go func() {
		wg.Wait()
		errch <- nil
		ch <- math.MaxUint64
	}()
	go func() {
		for _, member := range cluster.Status.Members {
			go func(member kstoneapiv1.MemberStatus) {
				defer wg.Done()
				ca, cert, key := "", "", ""
				if tls != nil {
					ca, cert, key = tls.TrustedCAFile, tls.CertFile, tls.KeyFile
				}

				backendStorage, endpoint := etcd.EtcdV3Backend, member.ExtensionClientUrl
				if strings.HasPrefix(member.Version, "2") {
					backendStorage = etcd.EtcdV2Backend
				}
				backend, err := etcd.NewEtcdStatBackend(backendStorage)
				if err != nil {
					klog.Errorf("failed to get etcd stat backend,backend %s,err is %v", endpoint, err)
					errch <- err
					return
				}
				err = backend.Init(ca, cert, key, endpoint)
				if err != nil {
					klog.Errorf(
						"failed to get new etcd clientv3,cluster name is %s,endpoint is %s,err is %v",
						cluster.Name,
						endpoint,
						err,
					)
					ch <- 0
					errch <- err
					return
				}
				defer backend.Close()
				totalKeyNum, err := backend.GetTotalKeyNum(keyPrefix)
				if err != nil {
					klog.Errorf("failed to get etcd cluster %s total key num,endpoint is %s,err is %v", cluster.Name, endpoint, err)
					ch <- 0
					errch <- err
					return
				}

				ch <- totalKeyNum

			}(member)
		}
	}()
	return ch, errch
}

// CollectMemberConsistency collects the consistency info, and
// transfer them to prometheus metrics
func (c *Server) CollectMemberConsistency(inspection *kstoneapiv1.EtcdInspection) error {
	namespace, name := inspection.Namespace, inspection.Spec.ClusterName
	cluster, tlsConfig, err := c.GetEtcdClusterInfo(namespace, name)
	if err != nil {
		klog.Errorf("failed to load tls config, namespace is %s, name is %s, err is %v", namespace, name, err)
		return err
	}

	path := DefaultInspectionPath
	annotations := inspection.ObjectMeta.Annotations
	if annotations != nil {
		if info, found := annotations[CruiseConsistencyAnno]; found {
			consistencyInfo := &ConsistencyInfo{}
			err = json.Unmarshal([]byte(info), consistencyInfo)
			if err != nil {
				klog.Errorf("failed to load inspection info, err is %v", err)
			} else {
				path = consistencyInfo.Path
			}
		}
	}

	var msg string
	var nodeKeyDiff uint64
	ch, errch := c.getEtcdNodeKeyDiff(cluster, path, tlsConfig)
	err = <-errch
	if err != nil {
		msg = fmt.Sprintf("failed to collectEtcdNodeKeyDiff,etcd cluster %s,err is %v", cluster.Name, err)
		klog.Errorf("%s", msg)
	} else {
		var keys []uint64
		for v := range ch {
			if v == math.MaxUint64 {
				break
			}
			keys = append(keys, v)
		}
		sort.Slice(keys, func(i, j int) bool { return keys[i] < keys[j] })
		nodeKeyDiff = keys[len(keys)-1] - keys[0]
	}
	labels := map[string]string{
		"clusterName": cluster.Name,
	}
	metrics.EtcdNodeDiffTotal.With(labels).Set(float64(nodeKeyDiff))
	return nil
}
