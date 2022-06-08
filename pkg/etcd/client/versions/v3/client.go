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

package v3

import (
	"strconv"

	clientv3 "go.etcd.io/etcd/client/v3"
	"k8s.io/klog/v2"

	kstonev1alpha2 "tkestack.io/kstone/pkg/apis/kstone/v1alpha2"
	"tkestack.io/kstone/pkg/etcd"
	"tkestack.io/kstone/pkg/etcd/client"
)

type V3 struct {
	ctx *client.VersionContext
	cli *clientv3.Client
}

func (c *V3) MemberList() ([]client.Member, error) {
	members := make([]client.Member, 0)
	memberRsp, err := etcd.MemberList(c.cli)
	if err != nil {
		klog.Errorf("failed to get member list, endpoints is %s,err is %v", c.ctx.Config.Endpoints, err)
		return members, err
	}
	for _, m := range memberRsp.Members {
		members = append(members, client.Member{
			ID:         strconv.FormatUint(m.ID, 10),
			Name:       m.Name,
			PeerURLs:   m.PeerURLs,
			ClientURLs: m.ClientURLs,
			IsLearner:  m.IsLearner,
		})
	}
	return members, nil
}

func (c *V3) Status(endpoint string) (*client.Member, error) {
	statusRsp, err := etcd.Status(c.ctx.Config.Endpoints[0], c.cli)
	if err != nil {
		return nil, err
	}
	return &client.Member{
		Version:   statusRsp.Version,
		IsLearner: statusRsp.IsLearner,
		Leader:    strconv.FormatUint(statusRsp.Leader, 10),
	}, nil
}

func (c *V3) Close() {
	c.cli.Close()
}

func init() {
	client.RegisterEtcdClientFactory(kstonev1alpha2.EtcdStorageV3,
		func(ctx *client.VersionContext) (client.VersionClient, error) {
			return initClient(ctx)
		})
}

func initClient(ctx *client.VersionContext) (client.VersionClient, error) {
	client := &V3{
		ctx: ctx,
		cli: nil,
	}
	var err error
	client.cli, err = etcd.NewClientv3(ctx.Config)
	if err != nil {
		klog.Errorf("failed to get new etcd clientv3,err is %v ", err)
		return nil, err
	}
	return client, nil
}
