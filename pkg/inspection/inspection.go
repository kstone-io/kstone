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
	"sync"
	"time"

	"go.etcd.io/etcd/client/pkg/v3/transport"
	clientv3 "go.etcd.io/etcd/client/v3"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	kstoneapiv1 "tkestack.io/kstone/pkg/apis/kstone/v1alpha1"
	"tkestack.io/kstone/pkg/controllers/util"
	"tkestack.io/kstone/pkg/etcd"
	clientset "tkestack.io/kstone/pkg/generated/clientset/versioned"
	platformscheme "tkestack.io/kstone/pkg/generated/clientset/versioned/scheme"
)

const (
	DefaultInspectionInterval = 300 * time.Second
	DefaultInspectionPath     = ""
)

type Server struct {
	Clientbuilder util.ClientBuilder
	cli           *clientset.Clientset
	kubeCli       kubernetes.Interface
	tlsGetter     etcd.TLSGetter
	client        map[string]*clientv3.Client
	wchan         map[string]clientv3.WatchChan
	watcher       map[string]clientv3.Watcher
	eventCh       map[string]chan *clientv3.Event
	mux           sync.Mutex
}

// Init inits the server of inspection
func (c *Server) Init() error {
	var err error
	c.kubeCli = c.Clientbuilder.ClientOrDie()
	c.cli, err = clientset.NewForConfig(c.Clientbuilder.ConfigOrDie())
	if err != nil {
		klog.Errorf("failed to init etcdinspection client, err is %v", err)
		return err
	}
	c.tlsGetter = etcd.NewTLSSecretGetter(c.Clientbuilder)
	c.client = make(map[string]*clientv3.Client)
	c.wchan = make(map[string]clientv3.WatchChan)
	c.watcher = make(map[string]clientv3.Watcher)
	c.eventCh = make(map[string]chan *clientv3.Event)

	return nil
}

// GetEtcdCluster gets etcdcluster
func (c *Server) GetEtcdCluster(namespace, name string) (*kstoneapiv1.EtcdCluster, error) {
	return c.cli.KstoneV1alpha1().EtcdClusters(namespace).Get(context.TODO(), name, metav1.GetOptions{})
}

// GetEtcdInspection gets etcdinspection
func (c *Server) GetEtcdInspection(namespace, name string) (*kstoneapiv1.EtcdInspection, error) {
	inspectionTask, err := c.cli.KstoneV1alpha1().EtcdInspections(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		klog.Errorf("failed to get etcdinspection, err: %v, namespace is %s, name is %s", err, namespace, name)
		return inspectionTask, err
	}
	return inspectionTask, nil
}

// CreateEtcdInspection creates etcdinspection
func (c *Server) CreateEtcdInspection(inspection *kstoneapiv1.EtcdInspection) (*kstoneapiv1.EtcdInspection, error) {
	newinspectionTask, err := c.cli.KstoneV1alpha1().EtcdInspections(inspection.Namespace).
		Create(context.TODO(), inspection, metav1.CreateOptions{})
	if err != nil {
		klog.Errorf(
			"failed to create etcdinspection, namespace is %s, name is %s, err is %v",
			inspection.Namespace,
			inspection.Name,
			err,
		)
		return newinspectionTask, err
	}
	return newinspectionTask, nil
}

func (c *Server) initInspectionTask(
	cluster *kstoneapiv1.EtcdCluster,
	inspectionType string,
) (*kstoneapiv1.EtcdInspection, error) {
	name := cluster.Name + "-" + inspectionType
	inspectionTask := &kstoneapiv1.EtcdInspection{}
	inspectionTask.ObjectMeta = metav1.ObjectMeta{
		Name:      name,
		Namespace: cluster.Namespace,
		Labels:    cluster.Labels,
	}
	inspectionTask.Spec = kstoneapiv1.EtcdInspectionSpec{
		InspectionType: inspectionType,
		ClusterName:    cluster.Name,
	}
	inspectionTask.Status = kstoneapiv1.EtcdInspectionStatus{
		LastUpdatedTime: metav1.Time{
			Time: time.Now(),
		},
	}

	err := controllerutil.SetOwnerReference(cluster, inspectionTask, platformscheme.Scheme)
	if err != nil {
		klog.Errorf("set inspection task's owner failed, err is %v", err)
		return inspectionTask, err
	}
	return inspectionTask, nil
}

func (c *Server) IsNotFound(cluster *kstoneapiv1.EtcdCluster, inspectionType string) bool {
	name := cluster.Name + "-" + inspectionType
	_, err := c.GetEtcdInspection(cluster.Namespace, name)
	if err != nil {
		return apierrors.IsNotFound(err)
	}
	return false
}

func (c *Server) GetEtcdClusterInfo(namespace, name string) (*kstoneapiv1.EtcdCluster, *transport.TLSInfo, error) {
	cluster, err := c.GetEtcdCluster(namespace, name)
	if err != nil {
		klog.Errorf("faild to get cluster info, namespace is %s, name is %s, err is %v", namespace, name, err)
		return nil, nil, err
	}

	annotations := cluster.ObjectMeta.Annotations
	secretName := ""
	if annotations != nil {
		if _, found := annotations[util.ClusterTLSSecretName]; found {
			secretName = annotations[util.ClusterTLSSecretName]
		}
	}
	tlsConfig, err := c.tlsGetter.Config(cluster.Name, secretName)
	if err != nil {
		klog.Errorf("failed to get cluster, namespace is %s, name is %s, err is %v", namespace, name, err)
		return nil, nil, err
	}
	return cluster, tlsConfig, nil
}
