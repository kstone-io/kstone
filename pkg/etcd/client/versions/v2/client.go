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

package v2

import (
	"context"
	"fmt"
	"net"
	"net/http"

	"github.com/coreos/etcd/pkg/transport"
	clientv2 "go.etcd.io/etcd/client/v2"
	klog "k8s.io/klog/v2"

	kstonev1alpha2 "tkestack.io/kstone/pkg/apis/kstone/v1alpha2"
	"tkestack.io/kstone/pkg/etcd"
	"tkestack.io/kstone/pkg/etcd/client"
)

type V2 struct {
	ctx *client.VersionContext
	cli *clientv2.Client
}

func (c *V2) MemberList() ([]client.Member, error) {
	API := clientv2.NewMembersAPI(*c.cli)
	rsp, err := API.List(context.Background())
	if err != nil {
		return nil, fmt.Errorf("load members err of endpoints:%s err:%s",
			c.ctx.Config.Endpoints, err.Error())
	}
	members := make([]client.Member, 0)
	for _, m := range rsp {
		members = append(members, client.Member{
			ID:         m.ID,
			Name:       m.Name,
			PeerURLs:   m.PeerURLs,
			ClientURLs: m.ClientURLs,
			IsLearner:  false,
		})
	}
	return members, nil
}

func (c *V2) Status(endpoint string) (*client.Member, error) {
	backend, err := etcd.NewEtcdHealthCheckBackend(etcd.HealthCheckHTTP)
	if err != nil {
		klog.Errorf("failed to get version backend,method %s,err is %v", etcd.HealthCheckHTTP, err)
		return nil, err
	}
	config := c.ctx.Config
	err = backend.Init(config.CaCert, config.Cert, config.Key, endpoint)
	if err != nil {
		klog.Errorf("failed to init version client,endpoint is %s,err is %v", endpoint, err)
		return nil, err
	}
	defer backend.Close()
	var version string
	version, err = backend.Version()
	if err != nil {
		klog.Errorf("failed to version,endpoint is %s,err is %v", endpoint, err)
		return nil, err
	}

	//get leader & memberID
	stats, err := backend.Stats()
	if err != nil {
		return nil, err
	}
	return &client.Member{
		ID:      stats.ID,
		Name:    stats.Name,
		Version: version,
		Leader:  stats.LeaderInfo.Leader,
	}, nil
}

func (c *V2) Close() {}

func init() {
	client.RegisterEtcdClientFactory(kstonev1alpha2.EtcdStorageV2,
		func(ctx *client.VersionContext) (client.VersionClient, error) {
			return initClient(ctx)
		})
}

func newClientCfg(ctx *client.VersionContext) (*clientv2.Config, error) {
	config := ctx.Config
	cfg := &clientv2.Config{
		Endpoints: config.Endpoints,
		Username:  config.Username,
		Password:  config.Password,
	}
	var cfgtls *transport.TLSInfo
	tlsinfo := transport.TLSInfo{}
	if ctx.Config.Cert != "" {
		tlsinfo.CertFile = config.Cert
		cfgtls = &tlsinfo
	}

	if config.Key != "" {
		tlsinfo.KeyFile = config.Key
		cfgtls = &tlsinfo
	}

	if config.CaCert != "" {
		tlsinfo.TrustedCAFile = config.CaCert
		cfgtls = &tlsinfo
	}

	if cfgtls != nil {
		clientTLS, err := cfgtls.ClientConfig()
		if err != nil {
			return nil, err
		}
		cfg.Transport = &http.Transport{
			Dial: (&net.Dialer{
				Timeout:   config.DialKeepAliveTimeout,
				KeepAlive: config.DialKeepAliveTime,
			}).Dial,
			TLSHandshakeTimeout: config.DialTimeout,
			TLSClientConfig:     clientTLS,
			MaxIdleConnsPerHost: 1,
			DisableKeepAlives:   true,
		}
	}
	return cfg, nil
}

func initClient(ctx *client.VersionContext) (client.VersionClient, error) {
	client := &V2{
		ctx: ctx,
		cli: nil,
	}
	cfg, err := newClientCfg(ctx)
	if err != nil {
		klog.Errorf("get new clientv2 cfg failed:%s", err)
		return nil, err
	}

	cli, err := clientv2.New(*cfg)
	if err != nil {
		klog.Errorf("create new clientv2 failed:%s", err)
		return nil, err
	}
	klog.V(2).Infof("init client ready of:%s", ctx.Config.Endpoints)
	client.cli = &cli
	return client, nil
}
