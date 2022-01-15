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

package jwt

import (
	"context"
	"crypto/rsa"
	"sync"
	"time"

	"github.com/golang-jwt/jwt"
	"k8s.io/klog/v2"

	"tkestack.io/kstone/pkg/authprovider"
)

const (
	ProviderName = "jwt"
)

var (
	// defaultTTL will be used when a 'ttl' is not specified
	defaultTTL = 24 * time.Hour
	once       sync.Once
	instance   *TokenGenerator
)

type TokenGenerator struct {
	signMethod jwt.SigningMethod
	key        interface{}
	ttl        time.Duration
}

func init() {
	authprovider.RegisterTokenGeneratorFactory(ProviderName,
		func(ctx *authprovider.TokenGeneratorContext) (authprovider.TokenGenerator, error) {
			return initTokenGeneratorInstance(ctx)
		},
	)
}

func initTokenGeneratorInstance(ctx *authprovider.TokenGeneratorContext) (*TokenGenerator, error) {
	var err error
	once.Do(func() {
		var duration time.Duration
		var key *rsa.PrivateKey
		if ctx.TTL == "" {
			duration = defaultTTL
		} else {
			duration, err = time.ParseDuration(ctx.TTL)
		}

		key, err = jwt.ParseRSAPrivateKeyFromPEM([]byte(ctx.PrivateKey))
		instance = &TokenGenerator{
			signMethod: jwt.GetSigningMethod(ctx.SignMethod),
			ttl:        duration,
			key:        key,
		}
	})
	if err != nil {
		return nil, err
	}

	return instance, nil
}

func (a *TokenGenerator) GenerateToken(ctx context.Context, username string, password string) (string, error) {
	tk := jwt.NewWithClaims(a.signMethod,
		jwt.MapClaims{
			"username": username,
			"password": password,
			"exp":      time.Now().Add(a.ttl).Unix(),
		})

	token, err := tk.SignedString(a.key)
	if err != nil {
		klog.Infof("failed to sign a JWT token for user %s, error is: %v", username, err)
		return "", err
	}
	return token, err
}
