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

package fixtures

import (
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	utilyaml "k8s.io/apimachinery/pkg/util/yaml"
	scheme "k8s.io/client-go/kubernetes/scheme"

	kstonev1alpha2 "tkestack.io/kstone/pkg/apis/kstone/v1alpha2"
	testfiles2 "tkestack.io/kstone/test/testfiles"
)

const (
	DefaultFeatureGate                   = "monitor=true,consistency=true,healthy=true,request=true,backup=true,alarm=true,backupcheck=true"
	DefaultTestClusterAddr               = "etcd-test-headless.default.svc.cluster.local:2379"
	DefaultTestClusterStatefulsetYaml    = "etcd_statefulset.yaml"
	DefaultTestClusterSvcYaml            = "etcd_service.yaml"
	DefaultKstoneNamespace               = "kstone"
	DefaultImportedClusterName           = "kstone-test"
	DefaultImportedPodName               = "etcd-test-0"
	DefaultNamespace                     = "default"
	DefaultKstoneEtcdOperatorClusterName = "kstone-etcd-operator-test"
	DefaultKstoneEtcdOperatorPodName     = "kstone-etcd-operator-test-etcd-0"
)

func NewEtcdCluster(
	name string,
	replicas uint,
	clusterType kstonev1alpha2.EtcdClusterType,
	featureGate,
	clusterAddr,
	scheme string) *kstonev1alpha2.EtcdCluster {
	cluster := &kstonev1alpha2.EtcdCluster{
		TypeMeta: metav1.TypeMeta{APIVersion: kstonev1alpha2.SchemeGroupVersion.String()},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: DefaultKstoneNamespace,
			Labels:    map[string]string{},
			Annotations: map[string]string{
				"autoTest":     "true",
				"featureGates": featureGate,
				"backup": `
{
	"backupPolicy": {
		"backupIntervalInSecond": 600,
		"maxBackups": 20,
		"timeoutInSecond": 10000
	},
	"cos": {
		"cosSecret": "kstone-test",
		"path": "kstone-test.cos.ap-nanjing.myqcloud.com/kstone-test"
	},
	"storageType": "COS"
}
`,
			},
		},
		Spec: kstonev1alpha2.EtcdClusterSpec{
			ClusterType: clusterType,
			Size:        replicas,
			DiskSize:    1,
			DiskType:    "ssd",
			Version:     "3.4.13",
		},
	}
	switch clusterType {
	case kstonev1alpha2.EtcdClusterImported:
		cluster.ObjectMeta.Annotations["importedAddr"] = clusterAddr
		cluster.ObjectMeta.Annotations["extClientURL"] = fmt.Sprintf("127.0.0.1:2379->%s", clusterAddr)
	case kstonev1alpha2.EtcdClusterKstone:
		cluster.ObjectMeta.Annotations["scheme"] = scheme
	}
	return cluster
}

func NewEtcdInspection(name string, inspectionType kstonev1alpha2.KStoneFeature) *kstonev1alpha2.EtcdInspection {
	return &kstonev1alpha2.EtcdInspection{
		TypeMeta: metav1.TypeMeta{APIVersion: kstonev1alpha2.SchemeGroupVersion.String()},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: metav1.NamespaceDefault,
			Labels:    map[string]string{},
		},
		Spec: kstonev1alpha2.EtcdInspectionSpec{
			InspectionType: string(inspectionType),
		},
	}
}

// SvcFromManifest reads a .json/yaml file and returns the service in it.
func SvcFromManifest(fileName string) (*v1.Service, error) {
	var svc v1.Service
	data, err := testfiles2.Read(fileName)
	if err != nil {
		return nil, err
	}

	json, err := utilyaml.ToJSON(data)
	if err != nil {
		return nil, err
	}
	if err := runtime.DecodeInto(scheme.Codecs.UniversalDecoder(), json, &svc); err != nil {
		return nil, err
	}
	return &svc, nil
}

// StatefulSetFromManifest returns a StatefulSet from a manifest stored in fileName in the Namespace indicated by ns.
func StatefulSetFromManifest(fileName, ns string) (*appsv1.StatefulSet, error) {
	var ss appsv1.StatefulSet
	data, err := testfiles2.Read(fileName)
	if err != nil {
		return nil, err
	}

	json, err := utilyaml.ToJSON(data)
	if err != nil {
		return nil, err
	}
	if err := runtime.DecodeInto(scheme.Codecs.UniversalDecoder(), json, &ss); err != nil {
		return nil, err
	}
	ss.Namespace = ns
	if ss.Spec.Selector == nil {
		ss.Spec.Selector = &metav1.LabelSelector{
			MatchLabels: ss.Spec.Template.Labels,
		}
	}
	return &ss, nil
}
