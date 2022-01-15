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
	"errors"
	"os"

	"github.com/gin-gonic/gin"
	"k8s.io/klog/v2"

	"tkestack.io/kstone/pkg/controllers/util"
)

var tokenProvider = os.Getenv("TOKEN_PROVIDER")

func LoginRequest(ctx *gin.Context) (*Response, bool, error) {
	username, password, err := GetUser(ctx)
	if err != nil {
		klog.Errorf("get user error: %v", err)
		return UnauthenticatedResponse(), false, err
	}

	authGetter := InitAuthContextGetter(util.NewSimpleClientBuilder(""))
	a, err := authGetter.Config(DefaultKstoneNamespace, DefaultConfigMapName, username, IsDefaultUser(username, password))
	if err != nil {
		klog.Errorf("get authenticator error: %v", err)
		return InternalServerErrorResponse(username, err.Error()), false, err
	}

	if err := CheckPassword(a.Password, password); err != nil {
		klog.Errorf("check password error: %v", err)
		return UnauthenticatedResponse(), false, errors.New(DataUnauthorized)
	}

	if username != a.Username {
		return UnauthenticatedResponse(), false, errors.New(DataUnauthorized)
	}

	return a.AuthenticateRequest(ctx)
}

func LogoutRequest(ctx *gin.Context) (*Response, error) {
	return SuccessResponse("", DataSuccess), nil
}

func MiddlewareRequest(ctx *gin.Context) (*Response, bool, error) {
	key, err := GetPrivateKey()
	if err != nil {
		return InternalServerErrorResponse(UserUnknown, err.Error()), false, err
	}
	c := &TokenContext{
		SignMethod: SignMethodRS256,
		PrivateKey: key,
	}
	t, err := GetTokenProvider(tokenProvider, c)
	if err != nil {
		klog.Errorf("login error: %v", err)
		return InternalServerErrorResponse(UserUnknown, err.Error()), false, err
	}
	return t.AuthenticateToken(context.TODO(), ctx.GetHeader(JWTTokenKey))
}

func UserUpdateRequest(ctx *gin.Context) (*Response, error) {
	username, password, err := GetUser(ctx)
	if err != nil {
		klog.Errorf("get user error: %v", err)
		return InternalServerErrorResponse(username, err.Error()), err
	}
	store := NewDefaultStore()
	passwordHash, err := GeneratePasswordHash(password)
	if err != nil {
		klog.Errorf("generate password hash error: %v", err)
		return InternalServerErrorResponse(username, err.Error()), err
	}
	if err := store.UserChangePassword(username, passwordHash); err != nil {
		return InternalServerErrorResponse(username, err.Error()), err
	}
	return SuccessResponse(username, DataSuccess), nil
}

func UserAddRequest(ctx *gin.Context) (*Response, error) {
	username, password, err := GetUser(ctx)
	if err != nil {
		klog.Errorf("get user error: %v", err)
		return InternalServerErrorResponse(username, err.Error()), err
	}
	store := NewDefaultStore()
	passwordHash, err := GeneratePasswordHash(password)
	if err != nil {
		klog.Errorf("generate password hash error: %v", err)
		return InternalServerErrorResponse(username, err.Error()), err
	}
	if err := store.UserAdd(User{Name: username, Password: passwordHash}); err != nil {
		return InternalServerErrorResponse(username, err.Error()), err
	}
	return SuccessResponse(username, DataSuccess), nil
}

func UserDeleteRequest(ctx *gin.Context) (*Response, error) {
	username, _, err := GetUser(ctx)
	if err != nil {
		klog.Errorf("get user error: %v", err)
		return InternalServerErrorResponse(username, err.Error()), err
	}
	store := NewDefaultStore()
	if err := store.UserDelete(username); err != nil {
		return InternalServerErrorResponse(username, err.Error()), err
	}
	return SuccessResponse(username, DataSuccess), nil
}
