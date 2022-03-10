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

	"tkestack.io/kstone/pkg/authentication"
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
	authentication.RegisterTokenFactory(ProviderName,
		func(ctx *authentication.TokenContext) (authentication.Token, error) {
			return initTokenInstance(ctx)
		},
	)
}

func initTokenInstance(ctx *authentication.TokenContext) (*TokenAuthenticator, error) {
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

func (a *TokenAuthenticator) AuthenticateToken(ctx context.Context, token string) (*authentication.Response, bool, error) {
	info, err := a.info(token)
	if err != nil {
		return authentication.UnauthenticatedResponse(), false, err
	}

	store := authentication.GetDefaultStoreInstance()
	user, err := store.UserGet(info.username)
	if err != nil {
		klog.Errorf("get user error: %v", err)
		return authentication.UnauthenticatedResponse(), false, errors.New(authentication.DataUnauthorized)
	}
	if info.username != user.Name || info.password != user.HashedPassword {
		return authentication.UnauthenticatedResponse(), false, fmt.Errorf("incorrect username or password")
	}

	return authentication.SuccessResponse(info.username, authentication.DataSuccess), true, nil
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

func GenerateToken(username, hashedPassword string) (string, error) {
	key, err := authentication.GetPrivateKey()
	if err != nil {
		return "", err
	}

	t, err := NewTokenGenerator("", key, authentication.SignMethodRS256)
	if err != nil {
		klog.Errorf("new token generator error: %v", err)
		return "", err
	}

	return t.GenerateToken(context.TODO(), username, hashedPassword)
}
