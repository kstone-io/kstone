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
	"errors"
	"strings"
	"time"

	"go.etcd.io/etcd/api/v3/mvccpb"
	clientv3 "go.etcd.io/etcd/client/v3"
	"k8s.io/klog/v2"

	kstonev1alpha1 "tkestack.io/kstone/pkg/apis/kstone/v1alpha1"
	"tkestack.io/kstone/pkg/clusterprovider"
	"tkestack.io/kstone/pkg/etcd"
	featureutil "tkestack.io/kstone/pkg/featureprovider/util"
	"tkestack.io/kstone/pkg/inspection/metrics"
)

const (
	inspectionRequestAnno = "request"
	eventBuffer           = 40960
)

type RequestInfo struct {
	Path     string `json:"path,omitempty"`
	Interval int    `json:"interval,omitempty"`
	Prefix   bool   `json:"prefix,omitempty"`
}

// CollectEtcdClusterRequest collects request of etcd
func (c *Server) CollectEtcdClusterRequest(inspection *kstonev1alpha1.EtcdInspection) error {
	namespace, name := inspection.Namespace, inspection.Spec.ClusterName
	cluster, tlsConfig, err := c.GetEtcdClusterInfo(namespace, name)
	defer func() {
		if err != nil {
			featureutil.IncrFailedInspectionCounter(name, kstonev1alpha1.KStoneFeatureRequest)
		}
	}()

	if err != nil {
		klog.Errorf("failed to get cluster info, namespace is %s, name is %s, err is %v", namespace, name, err)
		return err
	}

	_, ok := c.watcher[cluster.Name]
	if ok {
		return nil
	}

	annotations := cluster.ObjectMeta.Annotations
	watchKey := DefaultInspectionPath
	if annotations != nil {
		if infoStr, found := annotations[inspectionRequestAnno]; found {
			info := &RequestInfo{}
			err = json.Unmarshal([]byte(infoStr), info)
			if err == nil {
				watchKey = info.Path
			}
		}
	}

	ca, cert, key := "", "", ""
	if tlsConfig != nil {
		ca, cert, key = tlsConfig.TrustedCAFile, tlsConfig.CertFile, tlsConfig.KeyFile
	}

	client, err := etcd.NewClientv3(ca, cert, key, clusterprovider.GetStorageMemberEndpoints(cluster))
	if err != nil {
		klog.Errorf("failed to get new etcd clientv3,err is %v", err)
		return err
	}

	timeoutCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	rsp, rErr := client.Get(
		timeoutCtx,
		watchKey,
		clientv3.WithPrefix(),
		clientv3.WithSort(clientv3.SortByKey, clientv3.SortAscend),
		clientv3.WithKeysOnly(),
	)
	if rErr != nil {
		klog.Errorf("failed to get all etcd cluster keys,err is %v", rErr)
		return rErr
	}

	c.populateClusterTotalKeyMetrics(cluster, rsp.Kvs)
	c.client[cluster.Name] = client
	eventCh := make(chan *clientv3.Event, eventBuffer)
	c.setEventCh(eventCh, cluster.Name)
	err = c.Watch(cluster, client, watchKey)
	if err != nil {
		klog.Errorf("failed to get watch etcdcluster,err is %v", err)
		return err
	}
	go c.processWatchEvent(cluster)
	return err
}

// populateClusterTotalKeyMetrics generates prometheus metrics of the etcd key
func (c *Server) populateClusterTotalKeyMetrics(cluster *kstonev1alpha1.EtcdCluster, nodes []*mvccpb.KeyValue) {
	klog.V(2).Infof("cluster name %s,total node:%d", cluster.Name, len(nodes))
	labels := map[string]string{
		"clusterName": cluster.Name,
	}
	for i := 0; i < len(nodes); i++ {
		c.setEtcdPrefixAndResourceName(labels, string(nodes[i].Key))
		metrics.EtcdKeyTotal.With(labels).Inc()
	}
}

// setEtcdPrefixAndResourceName sets prefix and resource name of metric
func (c *Server) setEtcdPrefixAndResourceName(labels map[string]string, key string) {
	labels["etcdPrefix"] = ""
	labels["resourceName"] = ""
	keys := strings.Split(key, "/")
	if len(keys) < 2 {
		return
	}
	labels["etcdPrefix"] = keys[1]
	if len(keys) > 2 {
		labels["resourceName"] = keys[2]
	}
}

// setEventCh sets event chan
func (c *Server) setEventCh(ch chan *clientv3.Event, clusterName string) {
	c.mux.Lock()
	defer c.mux.Unlock()
	c.eventCh[clusterName] = ch
}

// getEventCh gets event chan
func (c *Server) getEventCh(clusterName string) chan *clientv3.Event {
	c.mux.Lock()
	defer c.mux.Unlock()
	ch, ok := c.eventCh[clusterName]
	if !ok {
		klog.Fatalf("failed to get event ch,cluster name is %s", clusterName)
	}
	return ch
}

// Watch watches etcd event
func (c *Server) Watch(cluster *kstonev1alpha1.EtcdCluster, client *clientv3.Client, keyPrefix string) error {
	watcher := clientv3.NewWatcher(client)
	c.watcher[cluster.Name] = watcher
	go func() {
		for {
			klog.V(2).Infof("cluster name:%s,prefix:%s,start to watch key change", cluster.Name, keyPrefix)
			ch := watcher.Watch(context.Background(), keyPrefix, clientv3.WithPrefix())
			err := c.watch(cluster, ch)
			if err == nil {
				return
			}
			//if failed to watch,just retry
			time.Sleep(time.Second)
		}
	}()
	return nil
}

func (c *Server) watch(cluster *kstonev1alpha1.EtcdCluster, wchan clientv3.WatchChan) error {
	ch := c.getEventCh(cluster.Name)
	for wresp := range wchan {
		if wresp.Canceled {
			klog.V(3).Infof("cluster:%s,watcher is closed", cluster.Name)
			return errors.New("watch failure")
		}
		for _, ev := range wresp.Events {
			switch ev.Type {
			case mvccpb.PUT:
				klog.V(3).Infof("type: put,key:%s,lease:%d,mod version:%d", ev.Kv.Key, ev.Kv.Lease, ev.Kv.ModRevision)
				ch <- ev
			case mvccpb.DELETE:
				klog.V(3).Infof("type: delete,key:%s", ev.Kv.Key)
				ch <- ev
			}
		}
	}
	return nil
}

// processWatchEvent prcoesses the event watched
func (c *Server) processWatchEvent(cluster *kstonev1alpha1.EtcdCluster) {
	ch := c.getEventCh(cluster.Name)
	labels := map[string]string{
		"clusterName": cluster.Name,
	}
	for ev := range ch {
		//fix inconsistent label cardinality,etcdKeyTotal metrics does not have label grpcMethod
		delete(labels, "grpcMethod")
		switch ev.Type {
		case mvccpb.PUT:
			c.setEtcdPrefixAndResourceName(labels, string(ev.Kv.Key))
			if ev.IsCreate() {
				metrics.EtcdKeyTotal.With(labels).Inc()
			}
			labels["grpcMethod"] = "PUT"
			klog.V(3).Infof("cluster:%s,type: PUT,key:%s,lease:%d", cluster.Name, ev.Kv.Key, ev.Kv.Lease)
			metrics.EtcdRequestTotal.With(labels).Inc()
		case mvccpb.DELETE:
			c.setEtcdPrefixAndResourceName(labels, string(ev.Kv.Key))
			metrics.EtcdKeyTotal.With(labels).Dec()
			labels["grpcMethod"] = "Delete"
			metrics.EtcdRequestTotal.With(labels).Inc()
			klog.V(3).Infof("cluster:%s,type: delete,key:%s,lease:%d", cluster.Name, ev.Kv.Key, ev.Kv.Lease)
		}
	}
}
