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

	clientv3 "go.etcd.io/etcd/client/v3"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	kstonev1alpha2 "tkestack.io/kstone/pkg/apis/kstone/v1alpha2"
	"tkestack.io/kstone/pkg/controllers/util"
	"tkestack.io/kstone/pkg/etcd"
	"tkestack.io/kstone/pkg/featureprovider"
	featureutil "tkestack.io/kstone/pkg/featureprovider/util"
	clientset "tkestack.io/kstone/pkg/generated/clientset/versioned"
	platformscheme "tkestack.io/kstone/pkg/generated/clientset/versioned/scheme"
)

const (
	DefaultInspectionInterval = 300 * time.Second
	DefaultInspectionPath     = ""
)

type Server struct {
	cli                *clientset.Clientset
	kubeCli            kubernetes.Interface
	client             map[string]*clientv3.Client
	wchan              map[string]clientv3.WatchChan
	watcher            map[string]clientv3.Watcher
	eventCh            map[string]chan *clientv3.Event
	mux                sync.Mutex
	clientConfigGetter etcd.ClientConfigGetter
}

// NewInspectionServer generates the server of inspection
func NewInspectionServer(ctx *featureprovider.FeatureContext) (*Server, error) {
	cli, err := clientset.NewForConfig(ctx.ClientBuilder.ConfigOrDie())
	if err != nil {
		klog.Errorf("failed to init etcdinspection client, err is %v", err)
		return nil, err
	}
	return &Server{
		kubeCli:            ctx.ClientBuilder.ClientOrDie(),
		cli:                cli,
		client:             make(map[string]*clientv3.Client),
		wchan:              make(map[string]clientv3.WatchChan),
		watcher:            make(map[string]clientv3.Watcher),
		eventCh:            make(map[string]chan *clientv3.Event),
		clientConfigGetter: ctx.ClientConfigGetter,
	}, nil
}

// GetEtcdCluster gets etcdcluster
func (c *Server) GetEtcdCluster(namespace, name string) (*kstonev1alpha2.EtcdCluster, error) {
	return c.cli.KstoneV1alpha2().EtcdClusters(namespace).Get(context.TODO(), name, metav1.GetOptions{})
}

// GetEtcdInspection gets etcdinspection
func (c *Server) GetEtcdInspection(namespace, name string) (*kstonev1alpha2.EtcdInspection, error) {
	inspectionTask, err := c.cli.KstoneV1alpha2().EtcdInspections(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		if !apierrors.IsNotFound(err) {
			klog.Errorf("failed to get etcdinspection, err: %v, namespace is %s, name is %s", err, namespace, name)
		}
		return nil, err
	}
	return inspectionTask, nil
}

// DeleteEtcdInspection deletes etcdinspection
func (c *Server) DeleteEtcdInspection(namespace, name string) error {
	err := c.cli.KstoneV1alpha2().EtcdInspections(namespace).
		Delete(context.TODO(), name, metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		klog.Errorf(
			"failed to delete etcdinspection, namespace is %s, name is %s, err is %v",
			namespace,
			name,
			err,
		)
		return err
	}
	return nil
}

// CreateEtcdInspection creates etcdinspection
func (c *Server) CreateEtcdInspection(inspection *kstonev1alpha2.EtcdInspection) (*kstonev1alpha2.EtcdInspection, error) {
	newinspectionTask, err := c.cli.KstoneV1alpha2().EtcdInspections(inspection.Namespace).
		Create(context.TODO(), inspection, metav1.CreateOptions{})
	if err != nil && !apierrors.IsAlreadyExists(err) {
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
	cluster *kstonev1alpha2.EtcdCluster,
	inspectionFeatureName kstonev1alpha2.KStoneFeature,
) (*kstonev1alpha2.EtcdInspection, error) {
	name := cluster.Name + "-" + string(inspectionFeatureName)
	inspectionTask := &kstonev1alpha2.EtcdInspection{}
	inspectionTask.ObjectMeta = metav1.ObjectMeta{
		Name:      name,
		Namespace: cluster.Namespace,
		Labels:    cluster.Labels,
	}
	inspectionTask.Spec = kstonev1alpha2.EtcdInspectionSpec{
		InspectionType: string(inspectionFeatureName),
		ClusterName:    cluster.Name,
	}
	inspectionTask.Status = kstonev1alpha2.EtcdInspectionStatus{
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

func (c *Server) GetEtcdClusterInfo(namespace, name string) (*kstonev1alpha2.EtcdCluster, *etcd.ClientConfig, error) {
	cluster, err := c.GetEtcdCluster(namespace, name)
	if err != nil {
		klog.Errorf("failed to get cluster info, namespace is %s, name is %s, err is %v", namespace, name, err)
		return nil, nil, err
	}

	annotations := cluster.ObjectMeta.Annotations
	secretName := ""
	if annotations != nil {
		if _, found := annotations[util.ClusterTLSSecretName]; found {
			secretName = annotations[util.ClusterTLSSecretName]
		}
	}
	clientConfig, err := c.clientConfigGetter.New(cluster.Name, secretName)
	if err != nil {
		klog.Errorf("failed to get cluster, namespace is %s, name is %s, err is %v", namespace, name, err)
		return nil, nil, err
	}
	return cluster, clientConfig, nil
}

func (c *Server) Equal(cluster *kstonev1alpha2.EtcdCluster, inspectionFeatureName kstonev1alpha2.KStoneFeature) bool {
	if !featureutil.IsFeatureGateEnabled(cluster.ObjectMeta.Annotations, inspectionFeatureName) {
		if cluster.Status.FeatureGatesStatus[inspectionFeatureName] != featureutil.FeatureStatusDisabled {
			return c.checkEqualIfDisabled(cluster, inspectionFeatureName)
		}
		return true
	}
	return c.checkEqualIfEnabled(cluster, inspectionFeatureName)
}

func (c *Server) Sync(cluster *kstonev1alpha2.EtcdCluster, inspectionFeatureName kstonev1alpha2.KStoneFeature) error {
	if !featureutil.IsFeatureGateEnabled(cluster.ObjectMeta.Annotations, inspectionFeatureName) {
		return c.cleanInspectionTask(cluster, inspectionFeatureName)
	}
	return c.syncInspectionTask(cluster, inspectionFeatureName)
}

// CheckEqualIfDisabled Checks whether the inspection task has been deleted if the inspection feature is disabled.
func (c *Server) checkEqualIfDisabled(cluster *kstonev1alpha2.EtcdCluster, inspectionFeatureName kstonev1alpha2.KStoneFeature) bool {
	name := cluster.Name + "-" + string(inspectionFeatureName)
	if _, err := c.GetEtcdInspection(cluster.Namespace, name); apierrors.IsNotFound(err) {
		return true
	}
	return false
}

// CheckEqualIfEnabled check whether the desired inspection task are consistent with the actual task,
// if the inspection feature is enabled.
func (c *Server) checkEqualIfEnabled(cluster *kstonev1alpha2.EtcdCluster, inspectionFeatureName kstonev1alpha2.KStoneFeature) bool {
	name := cluster.Name + "-" + string(inspectionFeatureName)
	if _, err := c.GetEtcdInspection(cluster.Namespace, name); err == nil {
		return true
	}
	return false
}

// CleanInspectionTask cleans inspection task
func (c *Server) cleanInspectionTask(cluster *kstonev1alpha2.EtcdCluster, inspectionFeatureName kstonev1alpha2.KStoneFeature) error {
	name := cluster.Name + "-" + string(inspectionFeatureName)
	return c.DeleteEtcdInspection(cluster.Namespace, name)
}

// SyncInspectionTask syncs inspection task
func (c *Server) syncInspectionTask(cluster *kstonev1alpha2.EtcdCluster, inspectionFeatureName kstonev1alpha2.KStoneFeature) error {
	task, err := c.initInspectionTask(cluster, inspectionFeatureName)
	if err != nil {
		return err
	}
	_, err = c.CreateEtcdInspection(task)
	if err != nil {
		return err
	}
	return nil
}
