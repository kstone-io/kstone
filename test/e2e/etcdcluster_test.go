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
	"errors"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"

	kstonev1alpha2 "tkestack.io/kstone/pkg/apis/kstone/v1alpha2"
	"tkestack.io/kstone/pkg/backup"
	"tkestack.io/kstone/test/fixtures"
)

var _ = ginkgo.Describe("kstone-etcd-operator etcdcluster", getGinkgoFunc(fixtures.DefaultKstoneEtcdOperatorClusterName, fixtures.DefaultKstoneEtcdOperatorPodName))
var _ = ginkgo.Describe("imported etcdcluster", getGinkgoFunc(fixtures.DefaultImportedClusterName, fixtures.DefaultImportedPodName))

func getGinkgoFunc(clusterName, podName string) func() {
	return func() {
		ginkgo.Describe("create an etcdcluster and enable monitor,backup,healthy,request,consistency,alarm,backupcheck features", func() {

			ginkgo.It("ensure cluster status to be running", func() {
				err := waitClusterStatusToRunning(clusterName)
				if err != nil {
					PrintTraceInfo(clusterName, podName)
				}
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			})

			ginkgo.It("kstone should generate etcdinspection/consistency resources", func() {
				err := CheckInspectionEnabled(clusterName, kstonev1alpha2.KStoneFeatureConsistency)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			})

			ginkgo.It("kstone should generate etcdinspection/healthy resources", func() {
				err := CheckInspectionEnabled(clusterName, kstonev1alpha2.KStoneFeatureHealthy)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			})

			ginkgo.It("kstone should generate etcdinspection/request resources", func() {
				err := CheckInspectionEnabled(clusterName, kstonev1alpha2.KStoneFeatureRequest)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			})

			ginkgo.It("kstone should generate etcdinspection/alarm resources", func() {
				err := CheckInspectionEnabled(clusterName, kstonev1alpha2.KStoneFeatureAlarm)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			})

			ginkgo.It("kstone should generate etcdinspection/backupcheck resources", func() {
				err := CheckInspectionEnabled(clusterName, kstonev1alpha2.KStoneFeatureBackupCheck)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			})

			ginkgo.It("kstone should generate prometheus servicemonitor resources", func() {
				err := CheckServiceMonitorEnabled(clusterName)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			})

			ginkgo.It("kstone should generate etcdbackup resources", func() {
				err := CheckBackupEnabled(clusterName)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			})

			ginkgo.It("kstone should be able to disable etcdinspection/consistency feature", func() {
				err := EnsureInspectionDisabled(clusterName, kstonev1alpha2.KStoneFeatureConsistency)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			})

			ginkgo.It("kstone should be able to disable etcdinspection/healthy feature", func() {
				err := EnsureInspectionDisabled(clusterName, kstonev1alpha2.KStoneFeatureHealthy)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			})

			ginkgo.It("kstone should be able to disable etcdinspection/request feature", func() {
				err := EnsureInspectionDisabled(clusterName, kstonev1alpha2.KStoneFeatureRequest)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			})

			ginkgo.It("kstone should be able to disable etcdinspection/alarm feature", func() {
				err := EnsureInspectionDisabled(clusterName, kstonev1alpha2.KStoneFeatureAlarm)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			})

			ginkgo.It("kstone should be able to disable etcdinspection/backupcheck feature", func() {
				err := EnsureInspectionDisabled(clusterName, kstonev1alpha2.KStoneFeatureBackupCheck)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			})

			ginkgo.It("kstone should be able to disable prometheus servicemonitor feature", func() {
				err := EnsureServiceMonitorDisabled(clusterName)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			})

			ginkgo.It("kstone should be able to disable etcdbackup feature", func() {
				err := EnsureBackupDisabled(clusterName)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			})
		})

		ginkgo.Describe("delete an existed etcdcluster", func() {
			ginkgo.It("kstone should delete etcd cluster", func() {
				err := deleteEtcdCluster(clusterName)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			})

			ginkgo.It("kstone should delete servicemonitor resources", func() {
				err := CheckServiceMonitorDisabled(clusterName)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			})

			ginkgo.It("kstone should delete etcdinspection/healthy resources", func() {
				err := CheckInspectionDisabled(clusterName, kstonev1alpha2.KStoneFeatureHealthy)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			})

			ginkgo.It("kstone should delete etcdinspection/request resources", func() {
				err := CheckInspectionDisabled(clusterName, kstonev1alpha2.KStoneFeatureRequest)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			})

			ginkgo.It("kstone should delete etcdinspection/consistency resources", func() {
				err := CheckInspectionDisabled(clusterName, kstonev1alpha2.KStoneFeatureConsistency)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			})

			ginkgo.It("kstone should delete etcdinspection/alarm resources", func() {
				err := CheckInspectionDisabled(clusterName, kstonev1alpha2.KStoneFeatureAlarm)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			})

			ginkgo.It("kstone should delete etcdinspection/backupcheck resources", func() {
				err := CheckInspectionDisabled(clusterName, kstonev1alpha2.KStoneFeatureBackupCheck)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			})

			ginkgo.It("kstone should delete etcdbackup resources", func() {
				err := CheckBackupDisabled(clusterName)
				gomega.Expect(err).ShouldNot(gomega.HaveOccurred())
			})

		})
	}
}

func createEtcdCluster(name string, replicas uint, clusterType kstonev1alpha2.EtcdClusterType, featureGate, clusterAddr, scheme string) error {
	etcdcluster := fixtures.NewEtcdCluster(name, replicas, clusterType, featureGate, clusterAddr, scheme)
	_, err := etcdClusterClient.KstoneV1alpha2().EtcdClusters(fixtures.DefaultKstoneNamespace).Create(context.TODO(), etcdcluster, metav1.CreateOptions{})
	return err
}

func deleteEtcdCluster(name string) error {
	err := etcdClusterClient.KstoneV1alpha2().EtcdClusters(fixtures.DefaultKstoneNamespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
	return err
}

func cleanAllEtcdCluster() error {
	clusters, err := etcdClusterClient.KstoneV1alpha2().EtcdClusters(fixtures.DefaultKstoneNamespace).List(context.TODO(), metav1.ListOptions{})
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

func waitClusterStatusToRunning(clusterName string) error {
	return wait.PollImmediate(pollInterval, pollTimeout, func() (done bool, err error) {
		cluster, err := etcdClusterClient.KstoneV1alpha2().EtcdClusters(fixtures.DefaultKstoneNamespace).Get(context.TODO(), clusterName, metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}
		if cluster.Status.Phase == kstonev1alpha2.EtcdClusterRunning {
			return true, nil
		}
		return false, nil
	})
}

func EnsureBackupDisabled(clusterName string) error {
	return wait.PollImmediate(pollInterval, pollTimeout, func() (done bool, err error) {
		err = DisableFeature(clusterName, kstonev1alpha2.KStoneFeatureBackup)
		if err != nil {
			return false, err
		}
		err = CheckBackupDisabled(clusterName)
		if err != nil {
			return false, err
		}
		return true, nil
	})
}

func CheckBackupEnabled(clusterName string) error {
	return wait.PollImmediate(pollInterval, pollTimeout, func() (done bool, err error) {
		_, err = dynamicCli.Resource(backup.BackupSchema).Namespace(fixtures.DefaultKstoneNamespace).Get(context.TODO(), clusterName, metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}
		return true, nil
	})
}

func CheckBackupDisabled(clusterName string) error {
	return wait.PollImmediate(pollInterval, pollTimeout, func() (done bool, err error) {
		_, err = dynamicCli.Resource(backup.BackupSchema).Namespace(fixtures.DefaultKstoneNamespace).Get(context.TODO(), clusterName, metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				return true, nil
			}
			return false, err
		}
		return false, nil
	})
}

func EnsureServiceMonitorDisabled(clusterName string) error {
	return wait.PollImmediate(pollInterval, pollTimeout, func() (done bool, err error) {
		err = DisableFeature(clusterName, kstonev1alpha2.KStoneFeatureMonitor)
		if err != nil {
			return false, err
		}
		err = CheckServiceMonitorDisabled(clusterName)
		if err != nil {
			return false, err
		}
		return true, nil
	})
}

func CheckServiceMonitorEnabled(clusterName string) error {
	return wait.PollImmediate(pollInterval, pollTimeout, func() (done bool, err error) {
		_, err = promCli.ServiceMonitors(fixtures.DefaultKstoneNamespace).Get(context.TODO(), clusterName, metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}
		return true, nil
	})
}

func CheckServiceMonitorDisabled(clusterName string) error {
	return wait.PollImmediate(pollInterval, pollTimeout, func() (done bool, err error) {
		_, err = promCli.ServiceMonitors(fixtures.DefaultKstoneNamespace).Get(context.TODO(), clusterName, metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				return true, nil
			}
			return false, err
		}
		return false, nil
	})
}

func EnsureInspectionDisabled(clusterName string, feature kstonev1alpha2.KStoneFeature) error {
	return wait.PollImmediate(pollInterval, pollTimeout, func() (done bool, err error) {
		err = DisableFeature(clusterName, feature)
		if err != nil {
			return false, err
		}
		err = CheckInspectionDisabled(clusterName, feature)
		if err != nil {
			return false, err
		}
		return true, nil
	})
}

func CheckInspectionEnabled(clusterName string, feature kstonev1alpha2.KStoneFeature) error {
	return wait.PollImmediate(pollInterval, pollTimeout, func() (done bool, err error) {
		_, err = etcdClusterClient.KstoneV1alpha2().EtcdInspections(fixtures.DefaultKstoneNamespace).Get(context.TODO(), clusterName+"-"+string(feature), metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}
		return true, nil
	})
}

func CheckInspectionDisabled(clusterName string, feature kstonev1alpha2.KStoneFeature) error {
	return wait.PollImmediate(pollInterval, pollTimeout, func() (done bool, err error) {
		_, err = etcdClusterClient.KstoneV1alpha2().EtcdInspections(fixtures.DefaultKstoneNamespace).Get(context.TODO(), clusterName+"-"+string(feature), metav1.GetOptions{})
		if err != nil {
			if apierrors.IsNotFound(err) {
				return true, nil
			}
			return false, err
		}
		return false, nil
	})
}

func DisableFeature(clusterName string, feature kstonev1alpha2.KStoneFeature) error {
	return wait.PollImmediate(pollInterval, pollTimeout, func() (done bool, err error) {
		cluster, err := etcdClusterClient.KstoneV1alpha2().EtcdClusters(fixtures.DefaultKstoneNamespace).Get(context.TODO(), clusterName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		annotations := cluster.ObjectMeta.Annotations
		annotations = UpdateAnnotationFeature(annotations, feature, false)
		if annotations == nil {
			return false, errors.New("can't change annotation")
		}
		_, err = etcdClusterClient.KstoneV1alpha2().EtcdClusters(fixtures.DefaultKstoneNamespace).Update(context.TODO(), cluster, metav1.UpdateOptions{})
		if err != nil {
			if apierrors.IsConflict(err) {
				return false, nil
			}
			return false, err
		}
		return true, nil
	})
}

func UpdateAnnotationFeature(annotations map[string]string, name kstonev1alpha2.KStoneFeature, open bool) map[string]string {
	if gates, found := annotations[kstonev1alpha2.KStoneFeatureAnno]; found && gates != "" {
		featurelist := strings.Split(gates, ",")
		feature := string(name)
		newItem := feature + "=" + strconv.FormatBool(open)
		for _, item := range featurelist {
			if strings.Contains(item, feature) {
				annotations[kstonev1alpha2.KStoneFeatureAnno] = strings.Replace(gates, item, newItem, 1)
				return annotations
			}
		}
	}
	return nil
}

func PrintTraceInfo(clusterName, podName string) {
	cmd1 := "kubectl describe po " + podName + " -n " + fixtures.DefaultKstoneNamespace
	cmd2 := "kubectl describe cluster " + clusterName + " -n " + fixtures.DefaultKstoneNamespace
	cmd3 := "kubectl describe etcd " + clusterName + " -n " + fixtures.DefaultKstoneNamespace
	cmd4 := "kubectl logs po " + podName + " -n " + fixtures.DefaultKstoneNamespace
	cmds := []string{cmd1, cmd2, cmd3, cmd4}
	nodeName, err := os.Hostname()
	if err != nil {
		klog.Errorf("unable to get hostname")
	} else {
		cmd5 := "kubectl describe no " + nodeName
		cmds = append(cmds, cmd5)
	}
	for _, cmd := range cmds {
		cmnd, err := exec.Command("sh", "-c", cmd).Output()
		if err != nil {
			klog.Error("failed to exec command, err is %v", err)
		} else {
			klog.Info(string(cmnd))
		}
	}
}
