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

package backup

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"tkestack.io/kstone/pkg/etcd"

	backupapiv2 "github.com/coreos/etcd-operator/pkg/apis/etcd/v1beta2"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	apiYaml "k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/yaml"

	kstonev1alpha2 "tkestack.io/kstone/pkg/apis/kstone/v1alpha2"
	"tkestack.io/kstone/pkg/controllers/util"
	platformscheme "tkestack.io/kstone/pkg/generated/clientset/versioned/scheme"
)

type Config struct {
	StorageType              backupapiv2.BackupStorageType `json:"storageType"`
	StoragePolicy            *backupapiv2.BackupPolicy     `json:"backupPolicy,omitempty"`
	backupapiv2.BackupSource `json:",inline"`
}

type Server struct {
	cli     dynamic.Interface
	kubeCli kubernetes.Interface
}

const (
	AnnoBackupConfig = "backup"
	BackupGroup      = "etcd.database.coreos.com"
	BackupVersion    = "v1beta2"
	BackupResource   = "etcdbackups"
	BackupKind       = "EtcdBackup"
)

var (
	BackupSchema = schema.GroupVersionResource{
		Group:    BackupGroup,
		Version:  BackupVersion,
		Resource: BackupResource,
	}
	backupSchemaKind = schema.GroupVersionKind{
		Group:   BackupGroup,
		Version: BackupVersion,
		Kind:    BackupKind,
	}
)

// NewBackupServer generates backup provider
func NewBackupServer(clientBuilder util.ClientBuilder) (*Server, error) {
	cli, err := dynamic.NewForConfig(clientBuilder.ConfigOrDie())
	if err != nil {
		klog.Errorf("failed to init backup client,err is %v", err)
		return nil, err
	}

	return &Server{
		cli:     cli,
		kubeCli: clientBuilder.ClientOrDie(),
	}, err
}

// encodeBackupObj encodes backup object
func (bak *Server) encodeBackupObj(obj *unstructured.Unstructured) (*backupapiv2.EtcdBackup, error) {
	data, err := obj.MarshalJSON()
	if err != nil {
		return nil, err
	}
	var newBackup backupapiv2.EtcdBackup
	if err := json.Unmarshal(data, &newBackup); err != nil {
		return nil, err
	}
	return &newBackup, nil
}

// decodeBackupObj decodes backup object
func (bak *Server) decodeBackupObj(backup *backupapiv2.EtcdBackup) (*unstructured.Unstructured, error) {
	data, err := json.Marshal(backup)
	if err != nil {
		return nil, fmt.Errorf("transfer json failed, err is %v", err)
	}
	yml, er := yaml.JSONToYAML(data)
	if er != nil {
		return nil, fmt.Errorf("transfer yaml failed, err is %v", err)
	}
	decoder := apiYaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
	obj := &unstructured.Unstructured{}
	if _, _, err := decoder.Decode(yml, &backupSchemaKind, obj); err != nil {
		return nil, err
	}
	return obj, nil
}

// CreateEtcdBackupByYaml creates etcd backup by yaml
func (bak *Server) CreateEtcdBackupByYaml(backup *backupapiv2.EtcdBackup) error {
	obj, err := bak.decodeBackupObj(backup)
	if err != nil {
		return fmt.Errorf("decode backup failed, err is %v", err)
	}
	_, err = bak.cli.Resource(BackupSchema).
		Namespace(backup.Namespace).
		Create(context.TODO(), obj, metav1.CreateOptions{})
	if err != nil && !apierrors.IsAlreadyExists(err) {
		return fmt.Errorf("create etcdbackup failed, err is %v", err)
	}
	return nil
}

// GetEtcdBackup gets etcd backup
func (bak *Server) GetEtcdBackup(name, namespace string) (*backupapiv2.EtcdBackup, error) {
	obj, err := bak.cli.Resource(BackupSchema).Namespace(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return bak.encodeBackupObj(obj)
}

// DeleteEtcdBackup deletes etcd backup
func (bak *Server) DeleteEtcdBackup(name, namespace string) error {
	err := bak.cli.Resource(BackupSchema).Namespace(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
	if err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	return nil
}

// UpdateEtcdBackup updates etcd backup
func (bak *Server) UpdateEtcdBackup(backup *backupapiv2.EtcdBackup) (*backupapiv2.EtcdBackup, error) {
	obj, err := bak.decodeBackupObj(backup)
	if err != nil {
		return nil, fmt.Errorf("decode backup failed, err is %v", err)
	}

	oldObj, err := bak.cli.Resource(BackupSchema).
		Namespace(backup.Namespace).
		Get(context.TODO(), backup.GetName(), metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	obj.SetResourceVersion(oldObj.GetResourceVersion())
	obj, err = bak.cli.Resource(BackupSchema).
		Namespace(backup.Namespace).
		Update(context.TODO(), obj, metav1.UpdateOptions{})
	if err != nil {
		return nil, err
	}
	return bak.encodeBackupObj(obj)
}

// parseBackupConfig parses backup config
func (bak *Server) parseBackupConfig(cluster *kstonev1alpha2.EtcdCluster) (*Config, string, error) {
	annotations := cluster.ObjectMeta.Annotations
	if annotations == nil {
		annotations = make(map[string]string)
	}

	strCfg, found := annotations[AnnoBackupConfig]
	if !found {
		klog.Errorf(
			"not found backup config, annotation key %s not exists, namespace is %s, name is %s",
			AnnoBackupConfig,
			cluster.Namespace,
			cluster.Name,
		)
		return nil, "", errors.New("backup config not found")
	}

	secretName := ""
	secretName, found = annotations[util.ClusterTLSSecretName]
	if strings.Contains(secretName, "/") {
		secretName = strings.Split(secretName, "/")[1]
	}

	// If enables tls and secretName cannot be empty
	authConfig := cluster.Spec.AuthConfig
	if authConfig.EnableTLS && !found {
		klog.Errorf("tlsEnabled cluster no secret, namespace is %s, name is %s", cluster.Namespace, cluster.Name)
		return nil, "", errors.New("secretName not found")
	}

	cfg := &Config{}
	err := json.Unmarshal([]byte(strCfg), cfg)
	if err != nil {
		klog.Errorf(
			"failed to parse backup config, namespace is %s, name is %s, err is %v",
			cluster.Namespace,
			cluster.Name,
			err,
		)
		return nil, "", errors.New("backup config parse failed")
	}
	return cfg, secretName, nil
}

// initEtcdBackup generates etcd backup
func (bak *Server) initEtcdBackup(cluster *kstonev1alpha2.EtcdCluster) (*backupapiv2.EtcdBackup, error) {
	backupCfg, secretName, err := bak.parseBackupConfig(cluster)
	if err != nil {
		return nil, err
	}

	backup := &backupapiv2.EtcdBackup{
		TypeMeta: metav1.TypeMeta{
			Kind:       BackupKind,
			APIVersion: BackupGroup + "/" + BackupVersion,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.Name,
			Namespace: cluster.Namespace,
			Labels:    cluster.ObjectMeta.Labels,
		},
		Spec: backupapiv2.BackupSpec{
			EtcdEndpoints: []string{cluster.Status.ServiceName},
			StorageType:   backupCfg.StorageType,
			//ClientTLSSecret: secretName,
			//InsecureSkipVerify: true,
			BackupPolicy: backupCfg.StoragePolicy,
			BackupSource: backupCfg.BackupSource,
			//BasicAuthSecret: secretName,
		},
	}
	//load secretConfig
	clientConfigGetter := etcd.NewClientConfigSecretGetter(util.NewSimpleClientBuilder(""))
	klog.Infof("secretName: %s", secretName)
	path := fmt.Sprintf("%s/%s", cluster.Namespace, cluster.Name)
	config, err := clientConfigGetter.New(path, secretName)
	if err != nil {
		klog.Errorf("failed to get cluster, namespace is %s, name is %s, err is %v", cluster.Namespace, cluster.Name, err)
		return nil, err
	}
	if config.Username != "" {
		backup.Spec.BasicAuthSecret = secretName
	}
	if config.CaCert != "" {
		backup.Spec.ClientTLSSecret = secretName
		backup.Spec.InsecureSkipVerify = true
	}

	return backup, nil
}

// Equal checks whether the backup resource needs to be updated
func (bak *Server) Equal(cluster *kstonev1alpha2.EtcdCluster) bool {
	namespace, name := cluster.Namespace, cluster.Name

	backup, err := bak.GetEtcdBackup(name, namespace)
	if err != nil {
		return apierrors.IsNotFound(err)
	}

	newBackup, err := bak.initEtcdBackup(cluster)
	if err != nil {
		klog.Errorf("init etcd backup failed, namespace is %s, name is %s, err is %v", namespace, name, err)
		return false
	}

	return !reflect.DeepEqual(backup, &newBackup)
}

// CheckEqualIfDisabled checks whether the backup resource is not found if it is disabled
func (bak *Server) CheckEqualIfDisabled(cluster *kstonev1alpha2.EtcdCluster) bool {
	if _, err := bak.GetEtcdBackup(cluster.Name, cluster.Namespace); apierrors.IsNotFound(err) {
		return true
	}
	return false
}

// CheckEqualIfEnabled checks whether the backup resource is equal if it is enabled
func (bak *Server) CheckEqualIfEnabled(cluster *kstonev1alpha2.EtcdCluster) bool {
	namespace, name := cluster.Namespace, cluster.Name
	backup, err := bak.GetEtcdBackup(name, namespace)
	if err != nil && apierrors.IsNotFound(err) {
		return false
	}

	newBackup, err := bak.initEtcdBackup(cluster)
	if err != nil {
		klog.Errorf("init etcd backup failed, namespace is %s, name is %s, err is %v", namespace, name, err)
		return false
	}

	return reflect.DeepEqual(backup, &newBackup)
}

// CleanBackup cleans the etcdbackup if it is disabled.
func (bak *Server) CleanBackup(cluster *kstonev1alpha2.EtcdCluster) error {
	err := bak.DeleteEtcdBackup(cluster.Name, cluster.Namespace)
	if err != nil {
		klog.Errorf("failed to delete etcd backup, namespace is %s, name is %s, err is %v", cluster.Namespace, cluster.Name, err)
	}
	return err
}

// SyncBackup synchronizes the etcdbackup if it is enabled.
func (bak *Server) SyncBackup(cluster *kstonev1alpha2.EtcdCluster) error {
	namespace, name := cluster.Namespace, cluster.Name
	newBackup, err := bak.initEtcdBackup(cluster)
	if err != nil {
		klog.Errorf("failed to init etcd backup, namespace is %s, name is %s, err is %v", namespace, name, err)
		return err
	}
	err = controllerutil.SetOwnerReference(cluster, newBackup, platformscheme.Scheme)
	if err != nil {
		klog.Errorf("set reference failed, err is %s, backup is %s", newBackup.Name)
		return err
	}
	_, err = bak.GetEtcdBackup(name, namespace)
	if err != nil && !apierrors.IsNotFound(err) {
		klog.Errorf("failed to get etcd backup, namespace is %s, name is %s, err is %v", namespace, name, err)
		return err
	}
	if apierrors.IsNotFound(err) {
		err = bak.CreateEtcdBackupByYaml(newBackup)
		if err != nil {
			klog.Errorf("failed to create etcd backup, namespace is %s, name is %s, err is %v", namespace, name, err)
		}
		return err
	}
	_, err = bak.UpdateEtcdBackup(newBackup)
	if err != nil {
		klog.Errorf("failed to update etcd backup, namespace is %s, name is %s, err is %v", namespace, name, err)
	}
	return err
}
