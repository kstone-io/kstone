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

package clusterprovider

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"sigs.k8s.io/yaml"
	"sync"

	"k8s.io/klog/v2"

	kstoneapiv1 "tkestack.io/kstone/pkg/apis/kstone/v1alpha1"
)

type EtcdFactory func(cluster *kstoneapiv1.EtcdCluster) (Cluster, error)

type ConfigInitFunc func(interface{}) error

type Config map[kstoneapiv1.EtcdClusterType]interface{}

var (
	mutex     sync.Mutex
	providers = make(map[kstoneapiv1.EtcdClusterType]EtcdFactory)

	configs         Config
	configInitFuncs = make(map[kstoneapiv1.EtcdClusterType]ConfigInitFunc)
)

// RegisterEtcdClusterFactory registers the specified cluster provider
func RegisterEtcdClusterFactory(name kstoneapiv1.EtcdClusterType, config interface{}, initFunc ConfigInitFunc, factory EtcdFactory) {
	mutex.Lock()
	defer mutex.Unlock()

	if _, found := providers[name]; found {
		klog.V(2).Infof("etcdcluster provider %s was registered twice", name)
	}

	klog.V(2).Infof("register etcdCluster provider %s", name)
	providers[name] = factory
	configs[name] = config
	configInitFuncs[name] = initFunc
}

// GetEtcdClusterProvider gets the specified cluster provider
func GetEtcdClusterProvider(
	name kstoneapiv1.EtcdClusterType,
	cluster *kstoneapiv1.EtcdCluster,
) (Cluster, error) {
	mutex.Lock()
	defer mutex.Unlock()
	f, found := providers[name]

	klog.V(1).Infof("get provider name %s,status:%t", name, found)
	if !found {
		return nil, errors.New("fatal error,etcd cluster provider not found")
	}
	return f(cluster)
}

// ClusterProviderInit gets the specified cluster provider
func ClusterProviderInit(config io.Reader) error {
	mutex.Lock()
	defer mutex.Unlock()

	src, err := ioutil.ReadAll(config)
	if err != nil {
		return err
	}
	cfg := make(Config, 0)
	if err := yaml.Unmarshal(src, cfg); err != nil {
		return err
	}

	for k, fn := range configInitFuncs {
		if fn == nil {
			continue
		}
		if err := fn(cfg[k]); err != nil {
			return fmt.Errorf("failed to init %v cluster provider", k)
		}
	}
	return nil
}
