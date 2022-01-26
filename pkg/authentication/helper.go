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
	"io/ioutil"
)

const (
	JWTTokenKey = "kstone-api-jwt"

	SignMethodRS256 = "RS256"

	DataSuccess      = "success"
	DataUnauthorized = "failed to authenticate user"
	UserUnknown      = "unknown"

	DefaultConfigMapName   = "kstone-api-user"
	DefaultKeyPath         = "/app/certs/private.key"
	DefaultKstoneNamespace = "kstone"
	DefaultUsername        = "admin"
	DefaultPassword        = "adm1n@kstone.io"
)

// Response is the struct returned by authenticator interfaces
type Response struct {
	Username      string `json:"username"`
	ResetPassword bool   `json:"reset_password"`
	Token         string `json:"token"`
	Message       string `json:"message"`
}

func SuccessResponse(username string, message string) *Response {
	return &Response{
		Username: username,
		Message:  message,
	}
}

func SuccessTokenResponse(username string, token string) *Response {
	return &Response{
		Username: username,
		Token:    token,
	}
}

func SuccessResetPasswordResponse(username string, token string) *Response {
	return &Response{
		Username:      username,
		Token:         token,
		ResetPassword: true,
	}
}

func UnauthenticatedResponse() *Response {
	return &Response{
		Username: UserUnknown,
		Message:  DataUnauthorized,
	}
}

func InternalServerErrorResponse(username string, message string) *Response {
	return &Response{
		Username: username,
		Message:  message,
	}
}

func GetPrivateKey() (string, error) {
	key, err := ioutil.ReadFile(DefaultKeyPath)
	if err != nil {
		return "", err
	}
	return string(key), nil
}
