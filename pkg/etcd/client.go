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
	"strings"
	"time"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	listerscorev1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/klog/v2"

	"tkestack.io/kstone/pkg/controllers/util"
)

type ClientConfigGetter interface {
	New(path string, sc string) (*ClientConfig, error)
}

type ClientConfigSecret struct {
	kubeCli      kubernetes.Interface
	secretLister listerscorev1.SecretLister
	secretGetter func(t *ClientConfigSecret, namespace, secretName string) (*v1.Secret, error)
}

type ClientConfig struct {
	// Endpoints is a list of URLs.
	Endpoints []string

	// DialTimeout is the timeout for failing to establish a connection.
	DialTimeout time.Duration

	// DialKeepAliveTime is the time after which client pings the server to see if
	// transport is alive.
	DialKeepAliveTime time.Duration

	// DialKeepAliveTimeout is the time that the client waits for a response for the
	// keep-alive probe. If the response is not received in this time, the connection is closed.
	DialKeepAliveTimeout time.Duration

	// SecureConfig is secure config for authentication
	SecureConfig
}

type SecureConfig struct {
	// Cert is a cert for authentication.
	Cert string

	// Key is a key for authentication.
	Key string

	// CaCert is a CA cert for authentication.
	CaCert string

	// Username is a user name for authentication.
	Username string

	// Password is a password for authentication.
	Password string
}

func NewClientConfigSecretGetter(clientBuilder util.ClientBuilder) *ClientConfigSecret {
	return &ClientConfigSecret{
		kubeCli:      clientBuilder.ClientOrDie(),
		secretGetter: Secret,
	}
}

func NewClientConfigSecretCacheGetter(secretLister listerscorev1.SecretLister) *ClientConfigSecret {
	return &ClientConfigSecret{
		secretLister: secretLister,
		secretGetter: SecretCache,
	}
}

func (t *ClientConfigSecret) New(path string, sc string) (*ClientConfig, error) {
	if sc == "" {
		return &ClientConfig{}, nil
	}
	items := strings.Split(sc, "/")
	paths := strings.Split(path, "/")
	namespace := "default"
	if len(paths) == 2 {
		namespace = paths[0]
	}
	secretName := sc
	if len(items) > 2 {
		return nil, errors.New("invalid secretname")
	} else if len(items) == 2 {
		namespace = items[0]
		secretName = items[1]
	}
	var secret *v1.Secret
	var err error
	secret, err = t.secretGetter(t, namespace, secretName)
	if err != nil {
		klog.Errorf("failed to get secret, namespace is %s, secret name is %s", namespace, secretName)
		return nil, err
	}

	cert := secret.Data[CliCertFile]
	key := secret.Data[CliKeyFile]
	ca := secret.Data[CliCAFile]
	username := secret.Data[CliUsername]
	password := secret.Data[CliPassword]
	caFile, certFile, keyFile, err := GetTLSConfigPath(path, cert, key, ca)
	if err != nil {
		klog.Errorf("failed to get tls config path, name %s,err is %v", secretName, err)
		return nil, err
	}
	return &ClientConfig{
		SecureConfig: SecureConfig{
			CaCert:   caFile,
			Cert:     certFile,
			Key:      keyFile,
			Username: string(username),
			Password: string(password),
		},
	}, nil
}

func SecretCache(t *ClientConfigSecret, namespace, secretName string) (*v1.Secret, error) {
	return t.secretLister.Secrets(namespace).Get(secretName)
}

func Secret(t *ClientConfigSecret, namespace, secretName string) (*v1.Secret, error) {
	return t.kubeCli.CoreV1().Secrets(namespace).Get(context.TODO(), secretName, metav1.GetOptions{})
}
