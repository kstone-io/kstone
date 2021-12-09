// Copyright 2017 The etcd-operator Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cosfactory

import (
	"fmt"
	"github.com/coreos/etcd-operator/pkg/util/tencentcloudutil/metadata/credential"
	"net/http"

	api "github.com/coreos/etcd-operator/pkg/apis/etcd/v1beta2"

	cos "github.com/tencentyun/cos-go-sdk-v5"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// COSClient is a wrapper for COS client that provides cleanup functionality.
type COSClient struct {
	COS *cos.Client
}

// NewClientFromSecret returns a COS client based on given k8s secret containing cos credentials.
func NewClientFromSecret(kubecli kubernetes.Interface, namespace, cosSecret string) (w *COSClient, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("new COS client failed: %v", err)
		}
	}()
	w = &COSClient{}
	se, err := kubecli.CoreV1().Secrets(namespace).Get(cosSecret, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("setup COS config failed: get k8s secret(%s) failed: %v", namespace, err)
	}
	secretId, exist := se.Data[api.COSSecretId]
	if !exist {
		return nil, fmt.Errorf("Get SecretId failed: %v", err)
	}
	secretKey, exist := se.Data[api.COSSecretKey]
	if !exist {
		return nil, fmt.Errorf("Get SecretKey failed: %v", err)
	}
	w.COS = cos.NewClient(nil, &http.Client{
		Transport: &cos.AuthorizationTransport{
			SecretID:  string(secretId),
			SecretKey: string(secretKey),
		},
	})
	return w, nil
}

func NewClientFromMetadata(role string) (w *COSClient, err error) {
	cred := credential.NewCredential(role)
	secretId, secretKey, token, err := cred.GetSecret()
	if err != nil {
		return nil, err
	}
	return &COSClient{
		COS: cos.NewClient(nil, &http.Client{
			Transport: &cos.AuthorizationTransport{
				SecretID:     secretId,
				SecretKey:    secretKey,
				SessionToken: token,
			},
		}),
	}, nil
}
