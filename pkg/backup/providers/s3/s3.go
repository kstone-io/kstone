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

package s3

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	awsS3 "github.com/aws/aws-sdk-go/service/s3"
	"github.com/coreos/etcd-operator/pkg/apis/etcd/v1beta2"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"

	"tkestack.io/kstone/pkg/apis/kstone/v1alpha2"
	"tkestack.io/kstone/pkg/backup"
	featureutil "tkestack.io/kstone/pkg/featureprovider/util"
)

const (
	ProviderName = string(v1beta2.BackupStorageTypeS3)
)

type StorageS3 struct {
	kubeCli kubernetes.Interface
}

func init() {
	backup.RegisterBackupStorageFactory(ProviderName, func(config *backup.StorageConfig) (backup.Storage, error) {
		return NewS3BackupProvider(config), nil
	})
}

func NewS3BackupProvider(config *backup.StorageConfig) backup.Storage {
	return &StorageS3{
		kubeCli: config.KubeCli,
	}
}

func (c *StorageS3) List(cluster *v1alpha2.EtcdCluster) (interface{}, error) {
	// get backup config
	backupConfig, err := backup.GetBackupConfig(cluster)
	if err != nil {
		klog.Errorf("failed to get backup config,cluster %s,err is %v", cluster.Name, err)
		return nil, err
	}
	klog.V(3).Infof("backup config is %v", backupConfig)

	secret, err := c.kubeCli.CoreV1().Secrets("kstone").Get(context.TODO(), backupConfig.S3.AWSSecret, v1.GetOptions{})
	if err != nil {
		klog.Errorf(err.Error())
		return nil, err
	}

	bucket := strings.Split(backupConfig.S3.Path, "/")
	if len(bucket) < 1 {
		err = fmt.Errorf("s3.path=%s, not valid, bucket not set", backupConfig.S3.Path)
		klog.Errorf(err.Error())
		return nil, err
	}

	cli, err := NewClientFromSecret(c.kubeCli, "kstone", backupConfig.S3.Endpoint, backupConfig.S3.AWSSecret, backupConfig.S3.ForcePathStyle)
	if err != nil {
		klog.Errorf(err.Error())
		return nil, err
	}
	defer cli.Close()

	klog.Info(fmt.Sprintf("secret name: %s, data: %v", secret.Name, secret.Data))

	i := 0
	var resultObjs []*awsS3.Object
	err = cli.S3.ListObjectsPages(&awsS3.ListObjectsInput{
		Bucket: &bucket[0],
	}, func(p *awsS3.ListObjectsOutput, last bool) (shouldContinue bool) {
		i++
		resultObjs = append(resultObjs, p.Contents...)
		return true
	})
	if err != nil {
		klog.Errorf("failed to list objects, error: %s", err.Error())
		return nil, err
	}
	klog.Infof("Total Page %d", i)

	return resultObjs, nil
}

func (c *StorageS3) Stat(objects interface{}) (int, error) {
	s3Objects, ok := objects.([]*awsS3.Object)
	if !ok {
		return 0, errors.New("can not decode objects to S3 objects")
	}
	backupFiles := 0
	for i := len(s3Objects) - 1; i >= 0; i-- {
		lastModifiedTime := *s3Objects[i].LastModified
		timeElapse := int64(time.Since(lastModifiedTime).Seconds())
		if timeElapse > featureutil.OneDaySeconds {
			break
		}
		backupFiles++
	}
	return backupFiles, nil
}
