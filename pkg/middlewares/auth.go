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

package middlewares

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"tkestack.io/kstone/pkg/authentication/request"
)

// Auth authenticates requests
func Auth() gin.HandlerFunc {
	return func(c *gin.Context) {
		rsp, ok, err := request.MiddlewareRequest(c)
		if !ok || err != nil {
			c.JSON(http.StatusUnauthorized, *rsp)
			c.Abort()
		}
		c.Next()
	}
}
