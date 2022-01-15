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
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/golang-jwt/jwt"
	"k8s.io/klog/v2"

	"tkestack.io/kstone/pkg/authprovider"
	"tkestack.io/kstone/pkg/controllers/util"
)

const (
	ProviderName = "jwt"
)

var (
	// defaultTTL will be used when a 'ttl' is not specified
	defaultTTL = 24 * time.Hour
	once       sync.Once
	instance   *TokenAuthenticator
)

type TokenAuthenticator struct {
	signMethod jwt.SigningMethod
	key        interface{}
	ttl        time.Duration
}

type authInfo struct {
	username string
	password string
}

func init() {
	authprovider.RegisterTokenFactory(ProviderName,
		func(ctx *authprovider.TokenContext) (authprovider.Token, error) {
			return initTokenInstance(ctx)
		},
	)
}

func initTokenInstance(ctx *authprovider.TokenContext) (*TokenAuthenticator, error) {
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
		instance = &TokenAuthenticator{
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

func (a *TokenAuthenticator) AuthenticateToken(ctx context.Context, token string) (*authprovider.Response, bool, error) {
	info, err := a.info(token)
	if err != nil {
		return authprovider.UnauthenticatedResponse(), false, err
	}

	authGetter := authprovider.InitAuthContextGetter(util.NewSimpleClientBuilder(""))
	auth, err := authGetter.Config(authprovider.DefaultKstoneNamespace, authprovider.DefaultConfigMapName, info.username, false)
	if err != nil {
		klog.Errorf("get authenticator error: %v", err)
		return authprovider.UnauthenticatedResponse(), false, errors.New(authprovider.DataUnauthorized)
	}

	if info.username != auth.Username || info.password != auth.Password {
		return authprovider.UnauthenticatedResponse(), false, fmt.Errorf("incorrect username or password")
	}

	return authprovider.SuccessResponse(info.username, authprovider.DataSuccess), true, nil
}

func (a *TokenAuthenticator) info(token string) (*authInfo, error) {
	parsed, err := jwt.Parse(token, func(token *jwt.Token) (interface{}, error) {
		if token.Method.Alg() != a.signMethod.Alg() {
			return nil, errors.New("invalid signing method")
		}
		switch k := a.key.(type) {
		case *rsa.PrivateKey:
			return &k.PublicKey, nil
		default:
			return nil, errors.New("invalid private key")
		}
	})

	if err != nil {
		klog.Errorf("failed to parse a JWT token: %s, error: %v", token, err)
		return nil, err
	}

	claims, ok := parsed.Claims.(jwt.MapClaims)
	if !parsed.Valid || !ok || claims["username"] == nil || claims["password"] == nil {
		klog.Errorf("invalid JWT token: %s", token)
		return nil, fmt.Errorf("invalid JWT token: %s", token)
	}
	return &authInfo{
		username: claims["username"].(string),
		password: claims["password"].(string),
	}, err
}
