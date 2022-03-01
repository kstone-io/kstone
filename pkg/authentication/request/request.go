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

package request

import (
	"encoding/json"
	"errors"
	"io/ioutil"
	"strings"

	"github.com/gin-gonic/gin"
	"k8s.io/klog/v2"

	"tkestack.io/kstone/pkg/authentication"
	"tkestack.io/kstone/pkg/authentication/authenticator"
	"tkestack.io/kstone/pkg/authentication/token/jwt"
)

func LoginRequest(ctx *gin.Context) (*authentication.Response, bool, error) {
	username, password, err := getUser(ctx)
	if err != nil {
		klog.Errorf("get user error: %v", err)
		return authentication.UnauthenticatedResponse(), false, err
	}

	store := authentication.GetDefaultStoreInstance()
	user, err := store.UserGet(username)
	if err != nil {
		klog.Errorf("get store user error: %v", err)
		return authentication.UnauthenticatedResponse(), false, err
	}

	if err := authentication.CheckPassword(user.HashedPassword, password); err != nil {
		klog.Errorf("check password error: %v", err)
		return authentication.UnauthenticatedResponse(), false, errors.New(authentication.DataUnauthorized)
	}

	token, err := jwt.GenerateToken(username, user.HashedPassword)
	if err != nil {
		klog.Errorf("generate token error: %v", err)
		return nil, false, err
	}

	if authentication.IsDefaultUser(username, password) {
		return authentication.SuccessResetPasswordResponse(username, token), true, nil
	}

	return authentication.SuccessTokenResponse(username, token), true, nil
}

func MiddlewareRequest(ctx *gin.Context) (*authentication.Response, bool, error) {
	a, err := authentication.GetAuthenticatorProvider(authenticator.ProviderName, &authentication.AuthenticatorContext{})
	if err != nil {
		klog.Errorf("get authenticator error: %v", err)
		return authentication.InternalServerErrorResponse(authentication.UserUnknown, err.Error()), false, err
	}
	return a.AuthenticateRequest(ctx)
}

func UserListRequest(ctx *gin.Context) (*authentication.Response, error) {
	var usernames []string
	store := authentication.GetDefaultStoreInstance()
	users, err := store.UserList()
	if err != nil {
		klog.Errorf("list store users error: %v", err)
		return authentication.UnauthenticatedResponse(), err
	}
	for _, u := range users {
		usernames = append(usernames, u.Name)
	}

	return authentication.SuccessResponse("", strings.Join(usernames, ",")), nil
}

func UserUpdateRequest(ctx *gin.Context) (*authentication.Response, error) {
	username, password, err := getUser(ctx)
	if err != nil {
		klog.Errorf("get user error: %v", err)
		return authentication.InternalServerErrorResponse(username, err.Error()), err
	}
	store := authentication.GetDefaultStoreInstance()
	passwordHash, err := authentication.GeneratePasswordHash(password)
	if err != nil {
		klog.Errorf("generate password hash error: %v", err)
		return authentication.InternalServerErrorResponse(username, err.Error()), err
	}
	if err := store.UserChangePassword(username, passwordHash); err != nil {
		return authentication.InternalServerErrorResponse(username, err.Error()), err
	}
	return authentication.SuccessResponse(username, authentication.DataSuccess), nil
}

func UserAddRequest(ctx *gin.Context) (*authentication.Response, error) {
	username, password, err := getUser(ctx)
	if err != nil {
		klog.Errorf("get user error: %v", err)
		return authentication.InternalServerErrorResponse(username, err.Error()), err
	}
	store := authentication.GetDefaultStoreInstance()
	hashedPassword, err := authentication.GeneratePasswordHash(password)
	if err != nil {
		klog.Errorf("generate password hash error: %v", err)
		return authentication.InternalServerErrorResponse(username, err.Error()), err
	}
	if err := store.UserAdd(authentication.User{Name: username, HashedPassword: hashedPassword}); err != nil {
		return authentication.InternalServerErrorResponse(username, err.Error()), err
	}
	return authentication.SuccessResponse(username, authentication.DataSuccess), nil
}

func UserDeleteRequest(ctx *gin.Context) (*authentication.Response, error) {
	username, _, err := getUser(ctx)
	if err != nil {
		klog.Errorf("get user error: %v", err)
		return authentication.InternalServerErrorResponse(username, err.Error()), err
	}
	store := authentication.GetDefaultStoreInstance()
	if err := store.UserDelete(username); err != nil {
		return authentication.InternalServerErrorResponse(username, err.Error()), err
	}
	return authentication.SuccessResponse(username, authentication.DataSuccess), nil
}

func getUser(ctx *gin.Context) (username string, password string, err error) {
	body, err := ioutil.ReadAll(ctx.Request.Body)
	if err != nil {
		return "", "", err
	}

	requestMap := make(map[string]string)
	if err := json.Unmarshal(body, &requestMap); err != nil {
		return "", "", err
	}

	if requestMap["username"] == "" {
		return "", "", errors.New("username is empty")
	}

	return requestMap["username"], requestMap["password"], nil
}
