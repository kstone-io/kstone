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

package cos

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/coreos/etcd-operator/pkg/apis/etcd/v1beta2"
	tencentCOS "github.com/tencentyun/cos-go-sdk-v5"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	klog "k8s.io/klog/v2"

	"tkestack.io/kstone/pkg/apis/kstone/v1alpha1"
	"tkestack.io/kstone/pkg/backup"
)

const (
	ProviderName = string(v1beta2.BackupStorageTypeCOS)
)

type BackupProvider struct {
	kubeconfig string
}

func init() {
	backup.RegisterBackupFactory(ProviderName, func(config *backup.ProviderConfig) (backup.Provider, error) {
		return NewCOSBackupProvider(config), nil
	})
}

func NewCOSBackupProvider(config *backup.ProviderConfig) backup.Provider {
	return &BackupProvider{
		kubeconfig: config.Kubeconfig,
	}
}

func (p *BackupProvider) List(cluster *v1alpha1.EtcdCluster) (interface{}, error) {
	var err error
	strCfg, found := cluster.Annotations[backup.AnnoBackupConfig]
	if !found {
		err = fmt.Errorf(
			"backup config not found, annotation key %s not exists, namespace is %s, name is %s",
			backup.AnnoBackupConfig,
			cluster.Namespace,
			cluster.Name,
		)
		klog.Errorf(err.Error())
		return nil, err
	}

	backupConfig := &backup.Config{}
	err = json.Unmarshal([]byte(strCfg), backupConfig)
	if err != nil {
		klog.Errorf(err.Error())
		return nil, err
	}
	klog.Info(backupConfig)

	cfg, err := clientcmd.BuildConfigFromFlags("", p.kubeconfig)
	if err != nil {
		klog.Errorf(err.Error())
		return nil, err
	}

	kubeClient, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		klog.Errorf(err.Error())
		return nil, err
	}

	secret, err := kubeClient.CoreV1().Secrets("kstone").Get(context.TODO(), backupConfig.COS.COSSecret, v1.GetOptions{})
	if err != nil {
		klog.Errorf(err.Error())
		return nil, err
	}

	secretID := string(secret.Data["secret-id"])
	secretKey := string(secret.Data["secret-key"])

	klog.Info(secretID)
	klog.Info(secretKey)

	cosPath := backupConfig.COS.Path
	if !strings.Contains(cosPath, "https://") {
		cosPath = fmt.Sprintf("https://%s", cosPath)
	}

	u, _ := url.Parse(cosPath)
	b := &tencentCOS.BaseURL{BucketURL: u}
	c := tencentCOS.NewClient(b, &http.Client{
		Transport: &tencentCOS.AuthorizationTransport{
			SecretID:  secretID,
			SecretKey: secretKey,
		},
	})

	result, _, err := c.Bucket.Get(context.Background(), &tencentCOS.BucketGetOptions{
		Prefix: strings.TrimLeft(b.BucketURL.Path, "/"),
	})
	if err != nil {
		klog.Errorf(err.Error())
		return nil, err
	}

	return result.Contents, nil
}
