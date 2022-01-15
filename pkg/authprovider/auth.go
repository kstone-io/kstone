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

package authprovider

import (
	"context"
	"os"
	"sync"

	"github.com/gin-gonic/gin"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"

	"tkestack.io/kstone/pkg/controllers/util"
)

var (
	tokenGeneratorProvider = os.Getenv("TOKEN_GENERATOR_PROVIDER")
	once                   sync.Once
	instance               *AuthContext
)

type Authenticator struct {
	Username      string
	Password      string
	ResetPassword bool
}

type AuthenticatorGetter interface {
	Config(namespace, userConfigMap, username string, resetPassword bool) (*Authenticator, error)
}

type AuthContext struct {
	kubeCli kubernetes.Interface
}

func InitAuthContextGetter(clientBuilder util.ClientBuilder) *AuthContext {
	once.Do(func() {
		instance = &AuthContext{
			kubeCli: clientBuilder.ClientOrDie(),
		}
	})
	return instance
}

func (t *AuthContext) Config(namespace, userConfigMap, username string, resetPassword bool) (*Authenticator, error) {

	cm, err := t.kubeCli.CoreV1().ConfigMaps(namespace).Get(context.TODO(), userConfigMap, metav1.GetOptions{})
	if err != nil {
		klog.Errorf("failed to get user configMap, namespace is %s, configMap name is %s", namespace, userConfigMap)
		return nil, err
	}

	return &Authenticator{
		Username:      username,
		Password:      cm.Data[username],
		ResetPassword: resetPassword,
	}, nil
}

func (a *Authenticator) AuthenticateRequest(ctx *gin.Context) (*Response, bool, error) {
	key, err := GetPrivateKey()
	if err != nil {
		return UnauthenticatedResponse(), false, err
	}
	c := &TokenGeneratorContext{
		SignMethod: SignMethodRS256,
		PrivateKey: key,
	}
	t, err := GetTokenGeneratorProvider(tokenGeneratorProvider, c)
	if err != nil {
		klog.Errorf("login error: %v", err)
		return UnauthenticatedResponse(), false, err
	}

	token, err := t.GenerateToken(context.TODO(), a.Username, a.Password)
	if err != nil {
		klog.Errorf("failed to generate token: %v", err)
		return UnauthenticatedResponse(), false, err
	}

	if a.ResetPassword {
		return SuccessResetPasswordResponse(a.Username, token), true, nil
	}

	return SuccessTokenResponse(a.Username, token), true, nil
}
