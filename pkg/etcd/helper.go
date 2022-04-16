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
	"fmt"
	"io/ioutil"
	"net"
	"net/http"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"go.etcd.io/etcd/client/pkg/v3/transport"
	clientv2 "go.etcd.io/etcd/client/v2"
	clientv3 "go.etcd.io/etcd/client/v3"
	klog "k8s.io/klog/v2"
)

const (
	DefaultDialTimeout      = 3 * time.Second
	DefaultCommandTimeOut   = 10 * time.Second
	DefaultKeepAliveTime    = 10 * time.Second
	DefaultKeepAliveTimeOut = 30 * time.Second

	CliCertFile = "client.pem"
	CliKeyFile  = "client-key.pem"
	CliCAFile   = "ca.pem"
	CliUsername = "username"
	CliPassword = "password"
)

// NewClientv3 generates etcd client v3
func NewClientv3(config *ClientConfig) (*clientv3.Client, error) {
	setDefaultConfig(config)
	cfg, err := newClientv3Config(config)
	if err != nil {
		klog.Errorf("get new clientv3 cfg failed:%s", err)
		return nil, err
	}

	client, err := clientv3.New(*cfg)
	if err != nil {
		klog.Errorf("create new clientv3 failed:%s", err)
		return nil, err
	}

	return client, nil
}

func setDefaultConfig(config *ClientConfig) {
	if config.DialTimeout == 0 {
		config.DialTimeout = DefaultDialTimeout
	}
	if config.DialKeepAliveTime == 0 {
		config.DialKeepAliveTime = DefaultKeepAliveTime
	}
	if config.DialKeepAliveTimeout == 0 {
		config.DialKeepAliveTimeout = DefaultKeepAliveTimeOut
	}
}

// newClientv3Config generates config of etcd client v3
func newClientv3Config(config *ClientConfig) (*clientv3.Config, error) {
	cfg := &clientv3.Config{
		Endpoints:            config.Endpoints,
		DialTimeout:          config.DialTimeout,
		DialKeepAliveTime:    config.DialKeepAliveTime,
		DialKeepAliveTimeout: config.DialKeepAliveTimeout,
		Username:             config.Username,
		Password:             config.Password,
	}
	// set tls if any one tls option set
	var cfgtls *transport.TLSInfo
	tlsinfo := transport.TLSInfo{}
	if config.Cert != "" {
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
		cfg.TLS = clientTLS
		cfg.TLS.InsecureSkipVerify = true

	}

	return cfg, nil
}

// MemberList gets etcd members
func MemberList(cli *clientv3.Client) (*clientv3.MemberListResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultDialTimeout)
	defer cancel()

	rsp, err := cli.MemberList(ctx)
	if err != nil {
		klog.Errorf("failed to get member list,err is %v", err)
		return nil, err
	}
	klog.V(6).Infof("get member list succ,resp info %v", rsp)
	return rsp, err
}

// Status returns new status
func Status(endpoint string, cli *clientv3.Client) (*clientv3.StatusResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultDialTimeout)
	defer cancel()

	return cli.Status(ctx, endpoint)
}

// writeFile writes []bytes to file
func writeFile(dir, file string, data []byte) (string, error) {
	p := filepath.Join(dir, file)
	return p, ioutil.WriteFile(p, data, 0600)
}

func GetTLSConfigPath(clusterName string, certData, keyData, caData []byte) (string, string, string, error) {
	// empty tlsFiles, return ""
	if len(certData) == 0 || len(keyData) == 0 || len(caData) == 0 {
		return "", "", "", nil
	}

	dir, err := ioutil.TempDir("", strings.ReplaceAll(clusterName, "/", "_"))
	if err != nil {
		return "", "", "", err
	}

	certFile, err := writeFile(dir, CliCertFile, certData)
	if err != nil {
		return "", "", "", err
	}
	keyFile, err := writeFile(dir, CliKeyFile, keyData)
	if err != nil {
		return "", "", "", err
	}
	caFile, err := writeFile(dir, CliCAFile, caData)
	if err != nil {
		return "", "", "", err
	}
	return caFile, certFile, keyFile, nil
}

func AddMemberWithCmd(isLearner bool, endpoints, peerURL, ca, cert, key string) error {
	args := make([]string, 0)
	if ca != "" && cert != "" && key != "" {
		args = append(args, fmt.Sprintf("--cacert=%s", ca))
		args = append(args, fmt.Sprintf("--cert=%s", cert))
		args = append(args, fmt.Sprintf("--key=%s", key))
	}

	endpointsStr := fmt.Sprintf("--endpoints=%s", endpoints)
	peerUlrStr := fmt.Sprintf("--peer-urls=%s", peerURL)
	isLearnerStr := fmt.Sprintf("--learner=%v", isLearner)

	name := strings.Split(strings.Split(peerURL, ":")[1], "/")[2]

	args = append(args, endpointsStr)
	args = append(args, "member")
	args = append(args, "add")
	args = append(args, name)
	args = append(args, peerUlrStr)
	args = append(args, isLearnerStr)

	cmd := exec.Command("etcdctl", args...)
	output, err := cmd.Output()
	if err != nil {
		fmt.Printf("Execute Shell:%s failed with error:%s", cmd, err.Error())
		return err
	}
	fmt.Printf("Execute Shell:%s finished with output:\n%s", cmd, string(output))
	return nil
}

// MemberHealthy checks healthy of member
func MemberHealthy(endpoint string, cli *ClientConfig) (bool, error) {
	backend, err := NewEtcdHealthCheckBackend(HealthCheckHTTP)
	if err != nil {
		klog.Errorf("failed to get healthcheck backend,method %s,err is %v", HealthCheckHTTP, err)
		return false, err
	}
	err = backend.Init(cli.CaCert, cli.Cert, cli.Key, endpoint)
	if err != nil {
		klog.Errorf("failed to init healthcheck client,endpoint is %s,err is %v", endpoint, err)
		return false, err
	}
	defer backend.Close()
	err = backend.IsHealthy()
	if err != nil {
		klog.Errorf("unhealthy,endpoint is %s,err is %v", endpoint, err)
		return false, nil
	}
	return true, nil
}

func NewShortConnectionClientv2(config *ClientConfig) (*clientv2.Client, error) {
	setDefaultConfig(config)
	cfg, err := newClientv2Config(config)
	if err != nil {
		klog.Errorf("get new clientv2 cfg failed:%s", err)
		return nil, err
	}

	client, err := clientv2.New(*cfg)
	if err != nil {
		klog.Errorf("create new clientv2 failed:%s", err)
		return nil, err
	}

	return &client, nil
}

// newClientv2Config generates config of etcd client v2
func newClientv2Config(config *ClientConfig) (*clientv2.Config, error) {
	tr, err := getTransport(config.DialTimeout, DefaultCommandTimeOut, config.SecureConfig, true)
	if err != nil {
		return nil, err
	}
	return &clientv2.Config{
		Transport:               tr,
		Endpoints:               config.Endpoints,
		HeaderTimeoutPerRequest: config.DialTimeout,
		Username:                config.Username,
		Password:                config.Password,
	}, nil
}

// getTransport gets *http.Transport
func getTransport(dialTimeout, totalTimeout time.Duration, scfg SecureConfig, short bool) (*http.Transport, error) {
	cafile := scfg.CaCert
	certfile := scfg.Cert
	keyfile := scfg.Key

	tls := transport.TLSInfo{
		CertFile:           certfile,
		KeyFile:            keyfile,
		TrustedCAFile:      cafile,
		InsecureSkipVerify: true,
	}

	if totalTimeout != 0 && totalTimeout < dialTimeout {
		dialTimeout = totalTimeout
	}
	if !short {
		return transport.NewTransport(tls, dialTimeout)
	}
	config, err := tls.ClientConfig()
	if err != nil {
		klog.Errorf("failed to get etcd server config,err is %v", err)
		return nil, err
	}
	return &http.Transport{
		Dial: (&net.Dialer{
			Timeout:   DefaultDialTimeout,
			KeepAlive: DefaultKeepAliveTime,
		}).Dial,
		TLSHandshakeTimeout: DefaultDialTimeout,
		TLSClientConfig:     config,
		MaxIdleConnsPerHost: 1,
		DisableKeepAlives:   true,
	}, nil
}

// AlarmList list etcd alarm
func AlarmList(cli *clientv3.Client) (*clientv3.AlarmResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), DefaultDialTimeout)
	defer cancel()

	rsp, err := cli.AlarmList(ctx)
	if err != nil {
		klog.Errorf("failed list etcd alarm,err is %v", err)
		return rsp, err
	}
	klog.V(6).Infof("list etcd alarm succ,resp info %v", rsp)
	return rsp, err
}
