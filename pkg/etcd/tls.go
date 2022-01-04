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

	"go.etcd.io/etcd/client/pkg/v3/transport"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	listerscorev1 "k8s.io/client-go/listers/core/v1"
	"k8s.io/klog/v2"

	"tkestack.io/kstone/pkg/controllers/util"
)

type TLSGetter interface {
	Config(path string, sc string) (*transport.TLSInfo, error)
}

type TLSSecret struct {
	kubeCli      kubernetes.Interface
	secretLister listerscorev1.SecretLister
	secretGetter func(t *TLSSecret, namespace, secretName string) (*v1.Secret, error)
}

func NewTLSSecretGetter(clientBuilder util.ClientBuilder) *TLSSecret {
	return &TLSSecret{
		kubeCli:      clientBuilder.ClientOrDie(),
		secretGetter: Secret,
	}
}

func NewTLSSecretCacheGetter(secretLister listerscorev1.SecretLister) *TLSSecret {
	return &TLSSecret{
		secretLister: secretLister,
		secretGetter: SecretCache,
	}
}

func (t *TLSSecret) Config(path string, sc string) (*transport.TLSInfo, error) {
	if sc == "" {
		return nil, nil
	}
	items := strings.Split(sc, "/")
	namespace := "default"
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
	caFile, certFile, keyFile, err := GetTLSConfigPath(path, cert, key, ca)
	if err != nil {
		klog.Errorf("failed to get tls config path, name %s,err is %v", secretName, err)
		return nil, err
	}
	cfg := &transport.TLSInfo{
		TrustedCAFile: caFile,
		KeyFile:       keyFile,
		CertFile:      certFile,
	}

	return cfg, nil
}

func SecretCache(t *TLSSecret, namespace, secretName string) (*v1.Secret, error) {
	return t.secretLister.Secrets(namespace).Get(secretName)
}

func Secret(t *TLSSecret, namespace, secretName string) (*v1.Secret, error) {
	return t.kubeCli.CoreV1().Secrets(namespace).Get(context.TODO(), secretName, metav1.GetOptions{})
}
