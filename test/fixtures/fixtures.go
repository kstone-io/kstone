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

	kstonev1alpha1 "tkestack.io/kstone/pkg/apis/kstone/v1alpha1"
	testfiles2 "tkestack.io/kstone/test/testfiles"
)

const (
	DefaultFeatureGate                = "monitor=true,consistency=true,healthy=true,request=true,backup=true"
	DefaultTestClusterAddr            = "etcd-test-headless.default.svc.cluster.local:2379"
	DefaultTestClusterStatefulsetYaml = "etcd_statefulset.yaml"
	DefaultTestClusterSvcYaml         = "etcd_service.yaml"
	DefaultKstoneNamespace            = "kstone"
)

func NewEtcdCluster(
	name string,
	replicas uint,
	clusterType kstonev1alpha1.EtcdClusterType,
	featureGate,
	clusterAddr string) *kstonev1alpha1.EtcdCluster {
	return &kstonev1alpha1.EtcdCluster{
		TypeMeta: metav1.TypeMeta{APIVersion: kstonev1alpha1.SchemeGroupVersion.String()},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: DefaultKstoneNamespace,
			Labels:    map[string]string{},
			Annotations: map[string]string{
				"autoTest":     "true",
				"featureGates": featureGate,
				"importedAddr": clusterAddr,
				"extClientURL": fmt.Sprintf("127.0.0.1:2379->%s", clusterAddr),
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
		Spec: kstonev1alpha1.EtcdClusterSpec{
			ClusterType: clusterType,
			Size:        replicas,
			DiskSize:    50,
			DiskType:    "ssd",
			Repository:  "bitnami/etcd",
			Version:     "3.5.0",
			TotalCpu:    2,
			TotalMem:    8,
		},
	}
}

func NewEtcdInspection(name string, inspectionType kstonev1alpha1.KStoneFeature) *kstonev1alpha1.EtcdInspection {
	return &kstonev1alpha1.EtcdInspection{
		TypeMeta: metav1.TypeMeta{APIVersion: kstonev1alpha1.SchemeGroupVersion.String()},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: metav1.NamespaceDefault,
			Labels:    map[string]string{},
		},
		Spec: kstonev1alpha1.EtcdInspectionSpec{
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
