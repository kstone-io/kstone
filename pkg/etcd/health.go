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
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	klog "k8s.io/klog/v2"
)

type Health interface {

	// Init creates etcd healthcheck client
	Init(ca, cert, key, endpoint string) error

	// IsHealthy checks etcd health info
	IsHealthy() error

	// Version returns etcd version
	Version() (string, error)

	// Stats returns etcd status
	Stats() (*Stats, error)

	// Close closes etcd healthcheck client
	Close() error
}

type HealthCheckMethod string

const (
	HealthCheckHTTP    HealthCheckMethod = "http"
	HealthCheckEtcdctl HealthCheckMethod = "etcdctl"
)

// HealthCheckHTTPClient struct of etcd healthcheck client
type HealthCheckHTTPClient struct {
	method   HealthCheckMethod
	cli      *http.Client
	endpoint string
}

// NewEtcdHealthCheckBackend generates etcd healthcheck client
func NewEtcdHealthCheckBackend(method HealthCheckMethod) (Health, error) {
	if method == HealthCheckHTTP {
		return &HealthCheckHTTPClient{method: HealthCheckHTTP}, nil
	}
	return nil, errors.New("invalid health check method")
}

// Init creates etcd healthcheck client
func (c *HealthCheckHTTPClient) Init(ca, cert, key, endpoint string) error {
	c.endpoint = endpoint
	tr := &http.Transport{}
	tr.MaxIdleConns = 1
	tr.DisableKeepAlives = true
	if ca != "" && cert != "" && key != "" {
		caCert, err := ioutil.ReadFile(ca)
		if err != nil {
			return err
		}
		keyPair, err := tls.LoadX509KeyPair(cert, key)
		if err != nil {
			return err
		}
		caPool := x509.NewCertPool()
		caPool.AppendCertsFromPEM(caCert)

		tlsConfig := &tls.Config{
			Certificates: []tls.Certificate{keyPair},
			RootCAs:      caPool,
		}
		tlsConfig.BuildNameToCertificate()
		tlsConfig.InsecureSkipVerify = true
		tr.TLSClientConfig = tlsConfig
	}
	c.cli = &http.Client{Transport: tr, Timeout: time.Second * 3}
	return nil
}

// etcdHealth encodes data returned from etcd /healthz handler.
type etcdHealth struct {
	// Note this has to be public so the json library can modify it.
	Health string `json:"health"`
}

// etcdHealthCheck decodes data returned from etcd /healthz handler.
func (c *HealthCheckHTTPClient) etcdHealthCheck(data []byte) error {
	obj := etcdHealth{}
	if err := json.Unmarshal(data, &obj); err != nil {
		return err
	}
	if obj.Health != "true" {
		klog.Warningf("etcd is not healthy,endpoint is %s", c.endpoint)
		return fmt.Errorf("Unhealthy status: %s", obj.Health)
	}
	return nil
}

// IsHealthy returns etcd healthy info by access etcd endpoint
func (c *HealthCheckHTTPClient) IsHealthy() error {
	target := fmt.Sprintf("%s/health", c.endpoint)
	resp, err := c.cli.Get(target)
	if err != nil {
		klog.Errorf("failed to check etcd healthy,err is %v", err)
		return err
	}
	defer resp.Body.Close()
	body, _ := ioutil.ReadAll(resp.Body)
	return c.etcdHealthCheck(body)
}

func (c *HealthCheckHTTPClient) GetByAPI(path string) ([]byte, error) {
	target := fmt.Sprintf("%s/%s", c.endpoint, path)
	resp, err := c.cli.Get(target)
	if err != nil {
		klog.Errorf("failed to check etcd healthy,err is %v", err)
		return make([]byte, 0), err
	}
	defer resp.Body.Close()
	return ioutil.ReadAll(resp.Body)
}

type Version struct {
	EtcdServer string `json:"etcdserver"`
}

func (c *HealthCheckHTTPClient) Version() (string, error) {
	body, err := c.GetByAPI("version")
	if err != nil {
		return "", fmt.Errorf("send request failed:%s", err.Error())
	}
	var version Version
	err = json.Unmarshal(body, &version)
	if err != nil {
		return "", fmt.Errorf("version result json failed:%s, body:%s", err.Error(), string(body))
	}
	return version.EtcdServer, nil
}

type Stats struct {
	Name       string `json:"name"`
	ID         string `json:"id"`
	LeaderInfo struct {
		Leader string `json:"leader"`
	} `json:"leaderInfo"`
}

func (c *HealthCheckHTTPClient) Stats() (*Stats, error) {
	body, err := c.GetByAPI("v2/stats/self")
	if err != nil {
		return nil, fmt.Errorf("send request failed:%s", err.Error())
	}
	var stats Stats
	err = json.Unmarshal(body, &stats)
	if err != nil {
		return nil, fmt.Errorf("version result json failed:%s, body:%s", err.Error(), string(body))
	}
	return &stats, nil
}

// Close closes etcd healthcheck client
func (c *HealthCheckHTTPClient) Close() error {
	return nil
}
