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

package bearertoken

import (
	"context"
	"sync"

	"github.com/gin-gonic/gin"
	"k8s.io/klog/v2"

	"tkestack.io/kstone/cmd/kstone-api/config"
	"tkestack.io/kstone/pkg/authentication"
)

const (
	ProviderName = "bearertoken"
)

var (
	authContextOnce sync.Once
	authenticator   *Authenticator
)

type Authenticator struct{}

func init() {
	authentication.RegisterAuthenticatorFactory(ProviderName,
		func(ctx *authentication.AuthenticatorContext) (authentication.Request, error) {
			return initAuthenticatorInstance(ctx)
		},
	)
}

func initAuthenticatorInstance(ctx *authentication.AuthenticatorContext) (*Authenticator, error) {
	authContextOnce.Do(func() {
		authenticator = &Authenticator{}
	})

	return authenticator, nil
}

func (a *Authenticator) AuthenticateRequest(ctx *gin.Context) (*authentication.Response, bool, error) {
	key, err := authentication.GetPrivateKey()
	if err != nil {
		return authentication.InternalServerErrorResponse(authentication.UserUnknown, err.Error()), false, err
	}
	c := &authentication.TokenContext{
		SignMethod: authentication.SignMethodRS256,
		PrivateKey: key,
	}
	t, err := authentication.GetTokenProvider(config.Cfg.Token, c)
	if err != nil {
		klog.Errorf("login error: %v", err)
		return authentication.InternalServerErrorResponse(authentication.UserUnknown, err.Error()), false, err
	}
	return t.AuthenticateToken(context.TODO(), ctx.GetHeader(authentication.JWTTokenKey))
}
