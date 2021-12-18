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
	"net/http"

	"github.com/gin-gonic/gin"
	clientv3 "go.etcd.io/etcd/client/v3"
	"k8s.io/klog/v2"

	"tkestack.io/kstone/pkg/etcd"
)

// EtcdKeyList returns etcd key list
func EtcdKeyList(ctx *gin.Context) {
	etcdKey := ctx.DefaultQuery("key", "")

	cluster, tlsConfig := GetEtcdClusterInfo(ctx)
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
