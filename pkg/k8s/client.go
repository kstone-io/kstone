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

package k8s

import (
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	kubeinformers "k8s.io/client-go/informers"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/klog/v2"

	clientset "tkestack.io/kstone/pkg/generated/clientset/versioned"
	informers "tkestack.io/kstone/pkg/generated/informers/externalversions"
)

// GetClientConfig gets *rest.Config with the kube config
func GetClientConfig(kubeconfig string) (*rest.Config, error) {
	var cfg *rest.Config
	var err error
	if kubeconfig != "" {
		cfg, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, err
		}
	} else {
		cfg, err = rest.InClusterConfig()
		if err != nil {
			return nil, err
		}
	}
	return cfg, nil
}

// GenerateInformer generates informer and client for controller
func GenerateInformer(config *rest.Config, labelSelector string) (
	*kubernetes.Clientset,
	*clientset.Clientset,
	kubeinformers.SharedInformerFactory,
	informers.SharedInformerFactory,
	error,
) {
	kubeClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		klog.Fatalf("Error building kubernetes clientset: %s", err.Error())
		return nil, nil, nil, nil, err
	}

	clustetClient, err := clientset.NewForConfig(config)
	if err != nil {
		klog.Fatalf("Error building example clientset: %s", err.Error())
		return nil, nil, nil, nil, err
	}

	informerFactory := informers.NewSharedInformerFactory(clustetClient, time.Second*30)
	if labelSelector != "" {
		optionsFunc := func(options *metav1.ListOptions) {
			options.LabelSelector = labelSelector
		}
		informerFactory = informers.NewSharedInformerFactoryWithOptions(
			clustetClient,
			time.Second*30,
			informers.WithTweakListOptions(optionsFunc),
		)
	}
	kubeInformerFactory := kubeinformers.NewSharedInformerFactory(kubeClient, time.Second*30)

	return kubeClient, clustetClient, kubeInformerFactory, informerFactory, nil
}
