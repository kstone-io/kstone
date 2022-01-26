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

package authenticator

import (
	"sync"

	"github.com/gin-gonic/gin"

	"tkestack.io/kstone/cmd/kstone-api/config"
	"tkestack.io/kstone/pkg/authentication"
)

const (
	ProviderName = "generic"
)

var (
	once     sync.Once
	instance *Authenticator
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
	once.Do(func() {
		instance = &Authenticator{}
	})
	return instance, nil
}

// AuthenticateRequest authenticates the request using custom authenticator.Request objects.
func (a *Authenticator) AuthenticateRequest(ctx *gin.Context) (*authentication.Response, bool, error) {
	authenticator, err := authentication.GetAuthenticatorProvider(config.Cfg.Authenticator, &authentication.AuthenticatorContext{})
	if err != nil {
		return authentication.UnauthenticatedResponse(), false, err
	}
	return authenticator.AuthenticateRequest(ctx)
}
