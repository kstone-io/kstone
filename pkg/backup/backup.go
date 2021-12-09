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

	backupapiv2 "github.com/coreos/etcd-operator/pkg/apis/etcd/v1beta2"
	k8serors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	apiYaml "k8s.io/apimachinery/pkg/runtime/serializer/yaml"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/yaml"

	kstoneapiv1 "tkestack.io/kstone/pkg/apis/kstone/v1alpha1"
	"tkestack.io/kstone/pkg/controllers/util"
	platformscheme "tkestack.io/kstone/pkg/generated/clientset/versioned/scheme"
)

type Config struct {
	StorageType              backupapiv2.BackupStorageType `json:"storageType"`
	StoragePolicy            *backupapiv2.BackupPolicy     `json:"backupPolicy,omitempty"`
	backupapiv2.BackupSource `json:",inline"`
}

type Server struct {
	Clientbuilder util.ClientBuilder
	cli           dynamic.Interface
	kubeCli       kubernetes.Interface
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

// Init inits backup provider
func (bak *Server) Init() error {
	var err error
	bak.cli, err = dynamic.NewForConfig(bak.Clientbuilder.ConfigOrDie())
	if err != nil {
		klog.Errorf("failed to init backup client,err is %v", err)
	}
	bak.kubeCli = bak.Clientbuilder.ClientOrDie()
	return err
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
func (bak *Server) CreateEtcdBackupByYaml(backup *backupapiv2.EtcdBackup) (*backupapiv2.EtcdBackup, error) {
	obj, err := bak.decodeBackupObj(backup)
	if err != nil {
		return nil, fmt.Errorf("decode backup failed, err is %v", err)
	}
	retObj, retErr := bak.cli.Resource(BackupSchema).
		Namespace(backup.Namespace).
		Create(context.TODO(), obj, metav1.CreateOptions{})
	if retErr != nil {
		return nil, fmt.Errorf("create etcdbackup failed, err is %v", retErr)
	}
	return bak.encodeBackupObj(retObj)
}

// GetEtcdBackup gets backup
func (bak *Server) GetEtcdBackup(name, namespace string) (*backupapiv2.EtcdBackup, error) {
	obj, err := bak.cli.Resource(BackupSchema).Namespace(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return bak.encodeBackupObj(obj)
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
func (bak *Server) parseBackupConfig(cluster *kstoneapiv1.EtcdCluster) (*Config, string, error) {
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
func (bak *Server) initEtcdBackup(cluster *kstoneapiv1.EtcdCluster) (*backupapiv2.EtcdBackup, error) {
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
			EtcdEndpoints:   []string{cluster.Status.ServiceName},
			StorageType:     backupCfg.StorageType,
			ClientTLSSecret: secretName,
			//	InsecureSkipVerify: true,
			BackupPolicy: backupCfg.StoragePolicy,
			BackupSource: backupCfg.BackupSource,
		},
	}
	return backup, nil
}

// Equal checks whether the backup resource needs to be updated
func (bak *Server) Equal(cluster *kstoneapiv1.EtcdCluster) bool {
	namespace, name := cluster.Namespace, cluster.Name
	backup, err := bak.GetEtcdBackup(name, namespace)
	if err != nil {
		return k8serors.IsNotFound(err)
	}

	newBackup, err := bak.initEtcdBackup(cluster)
	if err != nil {
		klog.Errorf("init etcd backup failed, namespace is %s, name is %s, err is %v", namespace, name, err)
		return false
	}

	return !reflect.DeepEqual(backup, &newBackup)
}

// SyncEtcdBackup synchronizes the latest backup configuration.
func (bak *Server) SyncEtcdBackup(cluster *kstoneapiv1.EtcdCluster) error {
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
	if err != nil && !k8serors.IsNotFound(err) {
		klog.Errorf("failed to get etcd backup, namespace is %s, name is %s, err is %v", namespace, name, err)
		return err
	}
	if k8serors.IsNotFound(err) {
		_, err = bak.CreateEtcdBackupByYaml(newBackup)
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
