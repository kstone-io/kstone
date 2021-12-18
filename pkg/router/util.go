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
	"go.etcd.io/etcd/client/pkg/v3/transport"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"

	kstoneapiv1 "tkestack.io/kstone/pkg/apis/kstone/v1alpha1"
	"tkestack.io/kstone/pkg/controllers/util"
	"tkestack.io/kstone/pkg/etcd"
	clientset "tkestack.io/kstone/pkg/generated/clientset/versioned"
)

func GetEtcdClusterInfo(ctx *gin.Context) (*kstoneapiv1.EtcdCluster, *transport.TLSInfo) {
	etcdName := ctx.Param("etcdName")
	// generate etcd client
	cfg, err := clientcmd.BuildConfigFromFlags("", "")
	if err != nil {
		klog.Errorf(err.Error())
		ctx.JSON(http.StatusInternalServerError, err)
		return nil, nil
	}

	// generate k8s client
	clusterClient, err := clientset.NewForConfig(cfg)
	if err != nil {
		klog.Errorf(err.Error())
		ctx.JSON(http.StatusInternalServerError, err)
		return nil, nil
	}

	// get cluster
	cluster, err := clusterClient.KstoneV1alpha1().EtcdClusters(Namespace).
		Get(context.TODO(), etcdName, metav1.GetOptions{})
	if err != nil {
		klog.Errorf(err.Error())
		ctx.JSON(http.StatusInternalServerError, err)
		return nil, nil
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
		return nil, nil
	}

	return cluster, tlsConfig
}
