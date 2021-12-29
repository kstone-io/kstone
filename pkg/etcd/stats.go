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

package etcd

import (
	"context"
	"errors"

	clientv2 "go.etcd.io/etcd/client/v2"
	clientv3 "go.etcd.io/etcd/client/v3"
	"k8s.io/klog/v2"

	featureutil "tkestack.io/kstone/pkg/featureprovider/util"
)

const (
	EtcdV2Backend = "v2"
	EtcdV3Backend = "v3"
)

type Stat interface {

	// Init create etcd client
	Init(ca, cert, key, endpoint string) error

	// GetTotalKeyNum counts the number of total keys
	GetTotalKeyNum(ctx context.Context, keyPrefix string) (uint64, error)

	// GetIndex gets the etcd metadata index
	GetIndex(ctx context.Context, endpoint string) (map[featureutil.ConsistencyType]uint64, error)

	// Close close etcd client
	Close() error
}

type StatV3 struct {
	backendName string
	cli         *clientv3.Client
}

type StatV2 struct {
	backendName string
	cli         *clientv2.Client
}

func NewEtcdStatBackend(storageBackend string) (Stat, error) {
	if storageBackend == EtcdV2Backend {
		return &StatV2{backendName: storageBackend}, nil
	} else if storageBackend == EtcdV3Backend {
		return &StatV3{backendName: storageBackend}, nil
	} else {
		return nil, errors.New("invalid storageBackend")
	}
}

func (c *StatV3) Init(ca, cert, key, endpoint string) error {
	var err error
	c.cli, err = NewClientv3(ca, cert, key, []string{endpoint})
	if err != nil {
		return err
	}
	return nil
}

func (c *StatV3) GetTotalKeyNum(ctx context.Context, keyPrefix string) (uint64, error) {
	rsp, err := c.cli.Get(ctx, keyPrefix, clientv3.WithPrefix(), clientv3.WithCountOnly())
	if err != nil {
		klog.Errorf("failed to get etcdcluster total key num,err is %v", err)
		return 0, err
	}
	totalKeyNum := rsp.Count
	klog.V(2).Infof("finished to get etcdcluster total key num %d", totalKeyNum)
	return uint64(totalKeyNum), nil
}

func (c *StatV3) GetIndex(ctx context.Context, endpoint string) (map[featureutil.ConsistencyType]uint64, error) {
	metadata := make(map[featureutil.ConsistencyType]uint64, 4)
	statusRsp, err := c.cli.Status(ctx, endpoint)
	if err != nil {
		klog.Errorf("failed to get etcdcluster index,err is %v", err)
		return nil, err
	}
	metadata[featureutil.ConsistencyRevision] = uint64(statusRsp.Header.Revision)
	metadata[featureutil.ConsistencyRaftRaftAppliedIndex] = statusRsp.RaftAppliedIndex
	metadata[featureutil.ConsistencyRaftIndex] = statusRsp.RaftIndex
	return metadata, nil
}

func (c *StatV3) Close() error {
	return c.cli.Close()
}

func (c *StatV2) Init(ca, cert, key, endpoint string) error {
	var err error
	c.cli, err = NewShortConnectionClientv2(ca, cert, key, []string{endpoint})
	if err != nil {
		return err
	}
	return nil
}

func (c *StatV2) GetTotalKeyNum(ctx context.Context, keyPrefix string) (uint64, error) {
	api := clientv2.NewKeysAPI(*c.cli)
	// Note that in order to avoid affecting performance, only some keys are counted.
	rsp, err := api.Get(ctx, keyPrefix, &clientv2.GetOptions{Recursive: false, Sort: true, Quorum: true})
	if err != nil {
		klog.Errorf("failed to get etcdcluster total key num,err is %v", err)
		return 0, err
	}
	totalKeyNum := uint64(len(rsp.Node.Nodes))
	klog.V(2).Infof("finished to get etcdcluster total key num %d,index is %d", totalKeyNum, rsp.Index)
	return totalKeyNum, nil
}

func (c *StatV2) GetIndex(ctx context.Context, endpoint string) (map[featureutil.ConsistencyType]uint64, error) {
	metadata := make(map[featureutil.ConsistencyType]uint64, 4)
	api := clientv2.NewKeysAPI(*c.cli)
	rsp, err := api.Get(ctx, "", &clientv2.GetOptions{Recursive: false, Sort: true, Quorum: true})
	if err != nil {
		klog.Errorf("failed to get etcdcluster index,err is %v", err)
		return nil, err
	}
	klog.V(2).Infof("finish to get etcdcluster index,%d", rsp.Index)
	metadata[featureutil.ConsistencyIndex] = rsp.Index
	metadata[featureutil.ConsistencyRaftRaftAppliedIndex] = 0
	metadata[featureutil.ConsistencyRaftIndex] = 0
	return metadata, nil
}

func (c *StatV2) Close() error {
	return nil
}
