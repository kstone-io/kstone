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
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/coreos/etcd-operator/pkg/apis/etcd/v1beta2"
	tencentCOS "github.com/tencentyun/cos-go-sdk-v5"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"

	"tkestack.io/kstone/pkg/apis/kstone/v1alpha2"
	"tkestack.io/kstone/pkg/backup"
	featureutil "tkestack.io/kstone/pkg/featureprovider/util"
)

const (
	ProviderName = string(v1beta2.BackupStorageTypeCOS)
)

type StorageCOS struct {
	kubeCli kubernetes.Interface
}

func init() {
	backup.RegisterBackupStorageFactory(ProviderName, func(config *backup.StorageConfig) (backup.Storage, error) {
		return NewCOSBackupProvider(config), nil
	})
}

func NewCOSBackupProvider(config *backup.StorageConfig) backup.Storage {
	return &StorageCOS{
		kubeCli: config.KubeCli,
	}
}

func (c *StorageCOS) List(cluster *v1alpha2.EtcdCluster) (interface{}, error) {
	// get backup config
	backupConfig, err := backup.GetBackupConfig(cluster)
	if err != nil {
		klog.Errorf("failed to get backup config,cluster %s,err is %v", cluster.Name, err)
		return nil, err
	}
	klog.V(3).Infof("backup config is %v", backupConfig)

	secret, err := c.kubeCli.CoreV1().Secrets(cluster.Namespace).Get(context.TODO(), backupConfig.COS.COSSecret, v1.GetOptions{})
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
	client := tencentCOS.NewClient(b, &http.Client{
		Transport: &tencentCOS.AuthorizationTransport{
			SecretID:  secretID,
			SecretKey: secretKey,
		},
	})

	result, _, err := client.Bucket.Get(context.Background(), &tencentCOS.BucketGetOptions{
		Prefix: strings.TrimLeft(b.BucketURL.Path, "/"),
	})
	if err != nil {
		klog.Errorf(err.Error())
		return nil, err
	}

	return result.Contents, nil
}

func (c *StorageCOS) Stat(objects interface{}) (int, error) {
	cosObjects, ok := objects.([]tencentCOS.Object)
	if !ok {
		return 0, errors.New("can not decode objects to COS objects")
	}
	backupFiles := 0
	for i := len(cosObjects) - 1; i >= 0; i-- {
		lastModifiedTime, err := time.Parse("2006-01-02T15:04:05Z", cosObjects[i].LastModified)
		if err != nil {
			return 0, errors.New("can not parse COS time")
		}
		timeElapse := int64(time.Since(lastModifiedTime).Seconds())
		if timeElapse > featureutil.OneDaySeconds {
			break
		}
		backupFiles++
	}
	return backupFiles, nil
}
