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

package authentication

import (
	"errors"
	"sync"

	"k8s.io/klog/v2"
)

var (
	mutex                  sync.Mutex
	TokenProviders         = make(map[string]TokenFactory)
	AuthenticatorProviders = make(map[string]AuthenticatorFactory)
)

type TokenFactory func(cfg *TokenContext) (Token, error)
type AuthenticatorFactory func(cfg *AuthenticatorContext) (Request, error)

// RegisterTokenFactory registers the specified token provider
func RegisterTokenFactory(name string, factory TokenFactory) {
	mutex.Lock()
	defer mutex.Unlock()

	if _, found := TokenProviders[name]; found {
		klog.V(2).Infof("token provider:%s was registered twice", name)
	}

	klog.V(2).Infof("token provider:%s", name)
	TokenProviders[name] = factory
}

// RegisterAuthenticatorFactory registers the specified authenticator provider
func RegisterAuthenticatorFactory(name string, factory AuthenticatorFactory) {
	mutex.Lock()
	defer mutex.Unlock()

	if _, found := AuthenticatorProviders[name]; found {
		klog.V(2).Infof("authenticator provider:%s was registered twice", name)
	}

	klog.V(2).Infof("authenticator provider:%s", name)
	AuthenticatorProviders[name] = factory
}

// GetTokenProvider gets the specified token provider
func GetTokenProvider(name string, ctx *TokenContext) (Token, error) {
	mutex.Lock()
	defer mutex.Unlock()
	f, found := TokenProviders[name]

	klog.V(1).Infof("get token name %s,status:%t", name, found)
	if !found {
		return nil, errors.New("fatal error,token provider not found")
	}
	return f(ctx)
}

// GetAuthenticatorProvider gets the specified authenticator provider
func GetAuthenticatorProvider(name string, ctx *AuthenticatorContext) (Request, error) {
	mutex.Lock()
	defer mutex.Unlock()
	f, found := AuthenticatorProviders[name]

	klog.V(1).Infof("get token name %s,status:%t", name, found)
	if !found {
		return nil, errors.New("fatal error,token provider not found")
	}
	return f(ctx)
}
