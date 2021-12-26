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

package router

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/http/httputil"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	// import backup provider
	_ "tkestack.io/kstone/pkg/backup/providers"
)

var (
	KubeScheme = "https"
	KubeTarget = os.Getenv("KUBE_TARGET")
	KubeToken  = os.Getenv("KUBE_TOKEN")
)

const (
	GroupName   = "kstone.tkestack.io"
	VersionName = "v1alpha1"
	Namespace   = "kstone"
)

// NewRouter generates router
func NewRouter() *gin.Engine {
	r := gin.Default()

	r.GET("/apis/:resource", ReverseProxy())
	r.POST("/apis/:resource", ReverseProxy())
	r.GET("/apis/:resource/:name", ReverseProxy())
	r.PUT("/apis/:resource/:name", ReverseProxy())
	r.PATCH("/apis/:resource/:name", ReverseProxy())
	r.DELETE("/apis/:resource/:name", ReverseProxy())

	r.GET("/apis/etcd/:etcdName", EtcdKeyList)
	r.GET("/apis/backup/:etcdName", BackupList)
	r.GET("/apis/alarm/:etcdName", AlarmList)
	r.POST("/apis/alarm/:etcdName", AlarmDisarm)
	return r
}

// ReverseProxy reverses proxy to kubernetes api
func ReverseProxy() gin.HandlerFunc {
	target := KubeTarget

	return func(c *gin.Context) {
		resource := c.Param("resource")
		name := c.Param("name")

		director := func(req *http.Request) {
			req.URL.Scheme = KubeScheme
			req.URL.Host = target
			req.Host = target
			// set Authorization to add k8s token
			req.Header = map[string][]string{
				"Authorization": {
					fmt.Sprintf("Bearer %s", KubeToken),
				},
			}

			var path string
			// handle different resource according to the resource type
			switch resource {
			case "etcdclusters":
				if name == "" {
					if req.Method == http.MethodGet {
						path = fmt.Sprintf("/apis/%s/%s/%s", GroupName, VersionName, resource)
					} else if req.Method == http.MethodPost || req.Method == http.MethodPut {
						path = fmt.Sprintf("/apis/%s/%s/namespaces/%s/%s", GroupName, VersionName, Namespace, resource)
					}
				} else {
					path = fmt.Sprintf("/apis/%s/%s/namespaces/%s/%s/%s", GroupName, VersionName, Namespace, resource, name)
				}
			case "secrets":
				if name == "" {
					if req.Method == http.MethodPost || req.Method == http.MethodGet {
						path = fmt.Sprintf("/api/v1/namespaces/%s/%s", Namespace, resource)
					}
				} else {
					path = fmt.Sprintf("/api/v1/namespaces/%s/%s/%s", Namespace, resource, name)
				}
			case "configmaps":
				if name == "" {
					if req.Method == http.MethodPost || req.Method == http.MethodGet {
						path = fmt.Sprintf("/api/v1/namespaces/%s/%s", Namespace, resource)
					}
				} else {
					path = fmt.Sprintf("/api/v1/namespaces/%s/%s/%s", Namespace, resource, name)
				}
			}

			req.URL.Path = path
			req.RequestURI = path
		}
		proxy := &httputil.ReverseProxy{Director: director}
		proxy.Transport = &http.Transport{
			Proxy:               http.ProxyFromEnvironment,
			TLSHandshakeTimeout: 10 * time.Second,
			TLSClientConfig:     &tls.Config{InsecureSkipVerify: true},
		}
		proxy.ServeHTTP(c.Writer, c.Request)
	}
}
