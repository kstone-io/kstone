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

package e2e

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/client/versioned/typed/monitoring/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	kstonev1alpha2 "tkestack.io/kstone/pkg/apis/kstone/v1alpha2"
	clientset "tkestack.io/kstone/pkg/generated/clientset/versioned"
	"tkestack.io/kstone/test/fixtures"
	"tkestack.io/kstone/test/testfiles"
)

const (
	// TestSuiteSetupTimeOut defines the time after which the suite setup times out.
	TestSuiteSetupTimeOut = 300 * time.Second
	// TestSuiteTeardownTimeOut defines the time after which the suite tear down times out.
	TestSuiteTeardownTimeOut = 300 * time.Second
	// pollInterval defines the interval time for a poll operation.
	pollInterval = 1 * time.Second
	// pollTimeout defines the time after which the poll operation times out.
	pollTimeout = 180 * time.Second
)

var (
	kubeconfig        string
	fixturesDir       string
	restConfig        *rest.Config
	kubeClient        kubernetes.Interface
	etcdClusterClient clientset.Interface
	dynamicCli        dynamic.Interface
	promCli           *monitoringv1.MonitoringV1Client
)

func TestE2E(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "e2e suite")
}

var _ = ginkgo.BeforeSuite(func() {
	// KUBECONFIG=/Users/etcd/.kube/config
	workspace := os.Getenv("GITHUB_WORKSPACE")
	gomega.Expect(workspace).ShouldNot(gomega.BeEmpty())

	kubeconfig = workspace + "/" + os.Getenv("E2E_KUBECONFIG_PATH")
	gomega.Expect(kubeconfig).ShouldNot(gomega.BeEmpty())

	// FIXTURESDIR=/Users/etcd/go/src/tkestack.io/kstone/test/fixtures/manifests
	fixturesDir = workspace + "/" + os.Getenv("FIXTURES_DIR")
	gomega.Expect(fixturesDir).ShouldNot(gomega.BeEmpty())

	var err error
	restConfig, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

	kubeClient, err = kubernetes.NewForConfig(restConfig)
	gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

	dynamicCli, err = dynamic.NewForConfig(restConfig)
	gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

	etcdClusterClient, err = clientset.NewForConfig(restConfig)
	gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

	promCli, err = monitoringv1.NewForConfig(restConfig)
	gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

	testfiles.AddFileSource(testfiles.RootFileSource{Root: fixturesDir})

	err = CreateTmpTestEtcdCluster()
	gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

	err = createEtcdCluster(fixtures.DefaultHTTPKstoneEtcdOperatorClusterName, 1, kstonev1alpha2.EtcdClusterKstone, fixtures.DefaultFeatureGate, "", "http")
	gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

	err = createEtcdCluster(fixtures.DefaultHTTPSKstoneEtcdOperatorClusterName, 1, kstonev1alpha2.EtcdClusterKstone, fixtures.DefaultFeatureGate, "", "https")
	gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

	podIP, err := getEtcdPodIP()
	gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
	err = createEtcdCluster(fixtures.DefaultImportedClusterName, 3, kstonev1alpha2.EtcdClusterImported, fixtures.DefaultFeatureGate, podIP+":2379", "")
	gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

}, TestSuiteSetupTimeOut.Seconds())

var _ = ginkgo.AfterSuite(func() {
	err := DeleteTmpTestEtcdCluster()
	gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

	err = cleanAllEtcdCluster()
	gomega.Expect(err).ShouldNot(gomega.HaveOccurred())

}, TestSuiteTeardownTimeOut.Seconds())

func CreateTmpTestEtcdCluster() error {
	sts, err := fixtures.StatefulSetFromManifest(fixtures.DefaultTestClusterStatefulsetYaml, fixtures.DefaultKstoneNamespace)
	if err != nil {
		return err
	}
	_, err = kubeClient.AppsV1().StatefulSets(fixtures.DefaultKstoneNamespace).Create(context.TODO(), sts, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	svc, err := fixtures.SvcFromManifest(fixtures.DefaultTestClusterSvcYaml)
	if err != nil {
		return err
	}
	_, err = kubeClient.CoreV1().Services(fixtures.DefaultKstoneNamespace).Create(context.TODO(), svc, metav1.CreateOptions{})
	if err != nil {
		return err
	}
	return nil
}

func GetTmpTestEtcdClusterPodIP() (string, error) {
	var podIP string
	podList, err := kubeClient.CoreV1().Pods(fixtures.DefaultKstoneNamespace).List(context.TODO(), metav1.ListOptions{LabelSelector: "app=etcd-test"})
	if err != nil {
		return "", err
	}
	for i := 0; i < len(podList.Items); i++ {
		if podList.Items[i].Status.Phase == v1.PodRunning &&
			len(podList.Items[i].Status.Conditions) > 0 &&
			podList.Items[i].Status.Conditions[0].Status == v1.ConditionTrue &&
			podList.Items[i].Status.PodIP != "" {
			podIP = podList.Items[i].Status.PodIP
			return podIP, nil
		}
	}
	return podIP, fmt.Errorf("etcd pod ip is invalid")
}

func DeleteTmpTestEtcdCluster() error {
	sts, err := fixtures.StatefulSetFromManifest(fixtures.DefaultTestClusterStatefulsetYaml, fixtures.DefaultKstoneNamespace)
	if err != nil {
		return err
	}
	err = kubeClient.AppsV1().StatefulSets(fixtures.DefaultKstoneNamespace).Delete(context.TODO(), sts.Name, metav1.DeleteOptions{})
	if err != nil {
		return err
	}
	svc, err := fixtures.SvcFromManifest(fixtures.DefaultTestClusterSvcYaml)
	if err != nil {
		return err
	}
	err = kubeClient.CoreV1().Services(fixtures.DefaultKstoneNamespace).Delete(context.TODO(), svc.Name, metav1.DeleteOptions{})
	if err != nil {
		return err
	}
	return nil
}
