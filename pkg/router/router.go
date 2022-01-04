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
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
	"net/http/httputil"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	clientv3 "go.etcd.io/etcd/client/v3"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	klog "k8s.io/klog/v2"

	"tkestack.io/kstone/pkg/backup"
	// import backup provider
	_ "tkestack.io/kstone/pkg/backup/providers"
	"tkestack.io/kstone/pkg/controllers/util"
	"tkestack.io/kstone/pkg/etcd"
	"tkestack.io/kstone/pkg/featureprovider"
	// import feature provider
	_ "tkestack.io/kstone/pkg/featureprovider/providers"
	featureutil "tkestack.io/kstone/pkg/featureprovider/util"
	clientset "tkestack.io/kstone/pkg/generated/clientset/versioned"
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
	r.GET("/apis/features", FeatureList)
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

// EtcdKeyList returns etcd key list
func EtcdKeyList(ctx *gin.Context) {
	etcdName := ctx.Param("etcdName")
	etcdKey := ctx.DefaultQuery("key", "")

	// generate etcd client
	cfg, err := clientcmd.BuildConfigFromFlags("", "")
	if err != nil {
		klog.Errorf(err.Error())
		ctx.JSON(http.StatusInternalServerError, err)
		return
	}

	clusterClient, err := clientset.NewForConfig(cfg)
	if err != nil {
		klog.Errorf(err.Error())
		ctx.JSON(http.StatusInternalServerError, err)
		return
	}

	cluster, err := clusterClient.KstoneV1alpha1().EtcdClusters("kstone").
		Get(context.TODO(), etcdName, metav1.GetOptions{})
	if err != nil {
		klog.Errorf(err.Error())
		ctx.JSON(http.StatusInternalServerError, err)
		return
	}

	annotations := cluster.Annotations
	secretName := ""
	if annotations != nil {
		if _, found := annotations["certName"]; found {
			secretName = annotations["certName"]
		}
	}
	tlsGetter := etcd.NewTLSSecretGetter(util.NewSimpleClientBuilder(""))
	klog.Infof("secretName: %s", secretName)
	tlsConfig, err := tlsGetter.Config(cluster.Name, secretName)
	if err != nil {
		klog.Errorf(err.Error())
		ctx.JSON(http.StatusInternalServerError, err)
		return
	}

	ca, cert, key := "", "", ""
	if tlsConfig != nil {
		ca, cert, key = tlsConfig.TrustedCAFile, tlsConfig.CertFile, tlsConfig.KeyFile
	}
	klog.Infof("endpoint: %s, ca: %s, cert: %s, key: %s", cluster.Status.ServiceName, ca, cert, key)
	client, err := etcd.NewClientv3(ca, cert, key, []string{cluster.Status.ServiceName})
	if err != nil {
		klog.Errorf(err.Error())
		ctx.JSON(http.StatusInternalServerError, err)
		return
	}
	defer client.Close()

	if etcdKey == "" {
		resp, err := client.Get(context.TODO(), "", clientv3.WithPrefix(), clientv3.WithKeysOnly())
		if err != nil {
			klog.Errorf(err.Error())
			ctx.JSON(http.StatusInternalServerError, err)
			return
		}

		data := make([]string, 0)
		for _, value := range resp.Kvs {
			data = append(data, string(value.Key))
		}

		ctx.JSON(http.StatusOK, map[string]interface{}{
			"code": 0,
			"data": data,
		})
		return
	}
	klog.Infof("get value by key: %s", etcdKey)
	resp, err := client.Get(context.TODO(), etcdKey, clientv3.WithPrefix())
	if err != nil {
		klog.Errorf(err.Error())
		ctx.JSON(http.StatusInternalServerError, err)
		return
	}
	if resp.Count == 0 {
		ctx.JSON(http.StatusNotFound, map[string]interface{}{
			"code": 1,
			"data": "",
		})
	} else {
		result := map[string]interface{}{
			"code": 0,
			"err":  "",
		}
		if cluster.Annotations["kubernetes"] == "true" && etcdKey != "compact_rev_key" {
			jsonValue := etcd.ConvertToJSON(resp.Kvs[0])
			inMediaType, in, err := etcd.DetectAndExtract(resp.Kvs[0].Value)
			if err != nil {
				klog.Errorf(err.Error())
				ctx.JSON(http.StatusInternalServerError, err)
				return
			}
			respData, err := etcd.ConvertToData(inMediaType, in)
			if err != nil {
				klog.Errorf(err.Error())
				if respData == nil {
					respData = make(map[string]string)
				}
				result["err"] = err.Error()
			}
			respData["json"] = jsonValue
			respDataList := make([]map[string]string, 0)
			for dataType, value := range respData {
				respDataList = append(respDataList, map[string]string{
					"type": dataType,
					"data": value,
				})
			}
			result["data"] = respDataList
		} else {
			result["data"] = []map[string]string{
				{
					"type": "javascript",
					"data": string(resp.Kvs[0].Value),
				},
			}
		}
		ctx.JSON(http.StatusOK, result)
	}
}

// BackupList returns backup list
func BackupList(ctx *gin.Context) {
	etcdName := ctx.Param("etcdName")

	clientBuilder := util.NewSimpleClientBuilder("")

	// generate k8s client
	clusterClient, err := clientset.NewForConfig(clientBuilder.ConfigOrDie())
	if err != nil {
		klog.Errorf(err.Error())
		ctx.JSON(http.StatusInternalServerError, err)
		return
	}

	// get cluster
	cluster, err := clusterClient.KstoneV1alpha1().EtcdClusters(Namespace).
		Get(context.TODO(), etcdName, metav1.GetOptions{})
	if err != nil {
		klog.Errorf(err.Error())
		ctx.JSON(http.StatusInternalServerError, err)
		return
	}

	// get backup config
	backupConfig, err := featureutil.GetBackupConfig(cluster)
	if err != nil {
		klog.Errorf("failed to get backup config,cluster %s,err is %v", cluster.Name, err)
		ctx.JSON(http.StatusInternalServerError, err)
	}

	// get specified backup storage provider
	storage, err := backup.GetBackupStorageProvider(string(backupConfig.StorageType), &backup.StorageConfig{
		KubeCli: clientBuilder.ClientOrDie(),
	})
	if err != nil {
		klog.Errorf(err.Error())
		ctx.JSON(http.StatusInternalServerError, err)
		return
	}
	resp, err := storage.List(cluster)
	if err != nil {
		klog.Errorf(err.Error())
		ctx.JSON(http.StatusInternalServerError, err)
		return
	}
	ctx.JSON(http.StatusOK, resp)
}

// FeatureList returns all features
func FeatureList(ctx *gin.Context) {
	features := featureprovider.ListFeatureProvider()
	ctx.JSON(http.StatusOK, features)
}
