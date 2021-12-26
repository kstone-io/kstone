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

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"

	kstoneapiv1 "tkestack.io/kstone/pkg/apis/kstone/v1alpha1"
	"tkestack.io/kstone/pkg/backup"
	"tkestack.io/kstone/test/fixtures"
)

var _ = ginkgo.Describe("etcdcluster", func() {
	clusterName := "kstone-test"
	ginkgo.Describe("import an existed etcdcluster and enable monitor,backup,healthy,request,consistency,alarm features", func() {
		ginkgo.BeforeEach(func() {
			//TODO: kstone does not support headless service,just use pod ip to bypass
			podIP, err := getEtcdPodIP()
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			err = createEtcdCluster(clusterName, 3, kstoneapiv1.EtcdClusterImported, fixtures.DefaultFeatureGate, podIP+":2379")
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		})

		ginkgo.AfterEach(func() {
			err := deleteEtcdCluster(clusterName)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		})

		ginkgo.It("ensure cluster status to be running", func() {
			err := wait.PollImmediate(pollInterval, pollTimeout, func() (done bool, err error) {
				cluster, err := etcdClusterClient.KstoneV1alpha1().EtcdClusters(fixtures.DefaultKstoneNamespace).Get(context.TODO(), clusterName, metav1.GetOptions{})
				if err != nil {
					if errors.IsNotFound(err) {
						return false, nil
					}
					return false, err
				}
				if cluster.Status.Phase == kstoneapiv1.EtcdClusterRunning {
					return true, nil
				}
				return false, nil
			})
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		})

		ginkgo.It("kstone should generate etcdinspection/consistency resources", func() {
			err := wait.PollImmediate(pollInterval, pollTimeout, func() (done bool, err error) {
				_, err = etcdClusterClient.KstoneV1alpha1().EtcdInspections(fixtures.DefaultKstoneNamespace).Get(context.TODO(), clusterName+"-"+string(kstoneapiv1.KStoneFeatureConsistency), metav1.GetOptions{})
				if err != nil {
					if errors.IsNotFound(err) {
						return false, nil
					}
					return false, err
				}
				return true, nil
			})
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		})

		ginkgo.It("kstone should generate prometheus servicemonitor resources", func() {
			err := wait.PollImmediate(pollInterval, pollTimeout, func() (done bool, err error) {
				_, err = promCli.ServiceMonitors(fixtures.DefaultKstoneNamespace).Get(context.TODO(), clusterName, metav1.GetOptions{})
				if err != nil {
					if errors.IsNotFound(err) {
						return false, nil
					}
					return false, err
				}
				return true, nil
			})
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		})

		ginkgo.It("kstone should generate etcdinspection/healthy resources", func() {
			err := wait.PollImmediate(pollInterval, pollTimeout, func() (done bool, err error) {
				_, err = etcdClusterClient.KstoneV1alpha1().EtcdInspections(fixtures.DefaultKstoneNamespace).Get(context.TODO(), clusterName+"-"+string(kstoneapiv1.KStoneFeatureHealthy), metav1.GetOptions{})
				if err != nil {
					if errors.IsNotFound(err) {
						return false, nil
					}
					return false, err
				}
				return true, nil
			})
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		})

		ginkgo.It("kstone should generate etcdinspection/request resources", func() {
			err := wait.PollImmediate(pollInterval, pollTimeout, func() (done bool, err error) {
				_, err = etcdClusterClient.KstoneV1alpha1().EtcdInspections(fixtures.DefaultKstoneNamespace).Get(context.TODO(), clusterName+"-"+string(kstoneapiv1.KStoneFeatureRequest), metav1.GetOptions{})
				if err != nil {
					if errors.IsNotFound(err) {
						return false, nil
					}
					return false, err
				}
				return true, nil
			})
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		})

		ginkgo.It("kstone should generate etcdbackup resources", func() {
			err := wait.PollImmediate(pollInterval, pollTimeout, func() (done bool, err error) {
				_, err = dynamicCli.Resource(backup.BackupSchema).Namespace(fixtures.DefaultKstoneNamespace).Get(context.TODO(), clusterName, metav1.GetOptions{})
				if err != nil {
					if errors.IsNotFound(err) {
						return false, nil
					}
					return false, err
				}
				return true, nil
			})
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		})

		ginkgo.It("kstone should generate etcdinspection/alarm resources", func() {
			err := wait.PollImmediate(pollInterval, pollTimeout, func() (done bool, err error) {
				_, err = etcdClusterClient.KstoneV1alpha1().EtcdInspections(fixtures.DefaultKstoneNamespace).Get(context.TODO(), clusterName+"-"+string(kstoneapiv1.KStoneFeatureAlarm), metav1.GetOptions{})
				if err != nil {
					if errors.IsNotFound(err) {
						return false, nil
					}
					return false, err
				}
				return true, nil
			})
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		})
	})

	ginkgo.Describe("delete an existed etcdcluster", func() {
		ginkgo.It("kstone should delete servicemonitor resources", func() {
			err := wait.PollImmediate(pollInterval, pollTimeout, func() (done bool, err error) {
				_, err = promCli.ServiceMonitors(fixtures.DefaultKstoneNamespace).Get(context.TODO(), clusterName, metav1.GetOptions{})
				if errors.IsNotFound(err) {
					return true, nil
				}
				return false, nil
			})
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		})

		ginkgo.It("kstone should delete etcdinspection/healthy resources", func() {
			err := wait.PollImmediate(pollInterval, pollTimeout, func() (done bool, err error) {
				_, err = etcdClusterClient.KstoneV1alpha1().EtcdInspections(fixtures.DefaultKstoneNamespace).Get(context.TODO(), clusterName+"-"+string(kstoneapiv1.KStoneFeatureHealthy), metav1.GetOptions{})
				if errors.IsNotFound(err) {
					return true, nil
				}
				return false, nil
			})
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		})

		ginkgo.It("kstone should delete etcdinspection/request resources", func() {
			err := wait.PollImmediate(pollInterval, pollTimeout, func() (done bool, err error) {
				_, err = etcdClusterClient.KstoneV1alpha1().EtcdInspections(fixtures.DefaultKstoneNamespace).Get(context.TODO(), clusterName+"-"+string(kstoneapiv1.KStoneFeatureRequest), metav1.GetOptions{})
				if errors.IsNotFound(err) {
					return true, nil
				}
				return false, nil
			})
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		})

		ginkgo.It("kstone should delete etcdinspection/consistency resources", func() {
			err := wait.PollImmediate(pollInterval, pollTimeout, func() (done bool, err error) {
				_, err = etcdClusterClient.KstoneV1alpha1().EtcdInspections(fixtures.DefaultKstoneNamespace).Get(context.TODO(), clusterName+"-"+string(kstoneapiv1.KStoneFeatureConsistency), metav1.GetOptions{})
				if errors.IsNotFound(err) {
					return true, nil
				}
				return false, nil
			})
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		})

		ginkgo.It("kstone should delete etcdbackup resources", func() {
			err := wait.PollImmediate(pollInterval, pollTimeout, func() (done bool, err error) {
				_, err = dynamicCli.Resource(backup.BackupSchema).Namespace(fixtures.DefaultKstoneNamespace).Get(context.TODO(), clusterName, metav1.GetOptions{})
				if errors.IsNotFound(err) {
					return true, nil
				}
				return false, nil
			})
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		})

		ginkgo.It("kstone should delete etcdinspection/alarm resources", func() {
			err := wait.PollImmediate(pollInterval, pollTimeout, func() (done bool, err error) {
				_, err = etcdClusterClient.KstoneV1alpha1().EtcdInspections(fixtures.DefaultKstoneNamespace).Get(context.TODO(), clusterName+"-"+string(kstoneapiv1.KStoneFeatureAlarm), metav1.GetOptions{})
				if errors.IsNotFound(err) {
					return true, nil
				}
				return false, nil
			})
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		})
	})
})

func createEtcdCluster(name string, replicas uint, clusterType kstoneapiv1.EtcdClusterType, featureGate, clusterAddr string) error {
	etcdcluster := fixtures.NewEtcdCluster(name, replicas, clusterType, featureGate, clusterAddr)
	_, err := etcdClusterClient.KstoneV1alpha1().EtcdClusters(fixtures.DefaultKstoneNamespace).Create(context.TODO(), etcdcluster, metav1.CreateOptions{})
	return err
}

func deleteEtcdCluster(name string) error {
	err := etcdClusterClient.KstoneV1alpha1().EtcdClusters(fixtures.DefaultKstoneNamespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
	return err
}

func cleanAllEtcdCluster() error {
	clusters, err := etcdClusterClient.KstoneV1alpha1().EtcdClusters(fixtures.DefaultKstoneNamespace).List(context.TODO(), metav1.ListOptions{})
	gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
	for i := 0; i < len(clusters.Items); i++ {
		if clusters.Items[i].Annotations["autoTest"] == "true" {
			err = deleteEtcdCluster(clusters.Items[i].Name)
			gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
		}
	}
	return err
}

func getEtcdPodIP() (string, error) {
	var podIP string
	err := wait.PollImmediate(pollInterval, pollTimeout, func() (done bool, err error) {
		podIP, err = GetTmpTestEtcdClusterPodIP()
		if err != nil {
			return false, nil
		}
		return true, nil
	})
	return podIP, err
}
