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

package monitor

import (
	"context"
	"os"
	"reflect"
	"sort"
	"strconv"
	"strings"

	promapiv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"
	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/client/versioned/typed/monitoring/v1"
	corev1 "k8s.io/api/core/v1"
	k8sV1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"k8s.io/client-go/kubernetes"
	klog "k8s.io/klog/v2"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	kstonev1alpha2 "tkestack.io/kstone/pkg/apis/kstone/v1alpha2"
	"tkestack.io/kstone/pkg/controllers/util"
	"tkestack.io/kstone/pkg/etcd"
	featureutil "tkestack.io/kstone/pkg/featureprovider/util"
	platformscheme "tkestack.io/kstone/pkg/generated/clientset/versioned/scheme"
)

const (
	DefaultEtcdV3SecretName  = "etcd-v3-certs"
	EtcdTLSPrefix            = "/etc/prometheus/secrets/"
	DefaultEtcdPromNamespace = "kstone"
)

var EtcdPromNamespace = os.Getenv("PROM_NAMESPACE")

type PrometheusMonitor struct {
	kubeCli kubernetes.Interface
	promCli *monitoringv1.MonitoringV1Client
}

// NewPrometheusMonitor generates prometheus provider
func NewPrometheusMonitor(clientBuilder util.ClientBuilder) (*PrometheusMonitor, error) {
	// init prom cli
	promCli, err := monitoringv1.NewForConfig(clientBuilder.ConfigOrDie())
	if err != nil {
		klog.Errorf("failed to init prom client, err is %v", err)
		return nil, err
	}
	return &PrometheusMonitor{
		kubeCli: clientBuilder.ClientOrDie(),
		promCli: promCli,
	}, nil
}

// GetEtcdService gets service
func (prom *PrometheusMonitor) GetEtcdService(namespace, name string) (*corev1.Service, error) {
	svr, err := prom.kubeCli.CoreV1().Services(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		klog.Errorf("get etcd service ,namespaces is %s,name is %s,error is %v", namespace, name, err)
		return nil, err
	}
	return svr, err
}

// CreateEtcdService creates service
func (prom *PrometheusMonitor) CreateEtcdService(service *corev1.Service) (*corev1.Service, error) {
	svr, err := prom.kubeCli.CoreV1().Services(service.Namespace).Create(context.TODO(), service, metav1.CreateOptions{})
	if err != nil {
		klog.Errorf("create etcd service ,namespaces is %s,name is %s,error is %v", service.Namespace, service.Name, err)
		return nil, err
	}
	return svr, err
}

// UpdateEtcdService updates service
func (prom *PrometheusMonitor) UpdateEtcdService(service *corev1.Service) (*corev1.Service, error) {
	svr, err := prom.kubeCli.CoreV1().Services(service.Namespace).Update(context.TODO(), service, metav1.UpdateOptions{})
	if err != nil {
		klog.Errorf("update etcd service ,namespaces is %s,name is %s,error is %v", service.Namespace, service.Name, err)
		return nil, err
	}
	return svr, err
}

// DeleteEtcdService deletes service
func (prom *PrometheusMonitor) DeleteEtcdService(namespace, name string) error {
	err := prom.kubeCli.CoreV1().Services(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
	if err != nil {
		klog.Errorf("delete etcd service, namespace is %s, name is %s,error is %v", namespace, name, err)
	}
	return err
}

// GetEtcdEndpoint gets endpoints
func (prom *PrometheusMonitor) GetEtcdEndpoint(namespace, name string) (*corev1.Endpoints, error) {
	ep, err := prom.kubeCli.CoreV1().Endpoints(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		klog.Errorf("get etcd ep ,namespaces is %s,name is %s,error is %v", namespace, name, err)
		return nil, err
	}
	return ep, err
}

// UpdateEtcdEndpoint updates endpoints
func (prom *PrometheusMonitor) UpdateEtcdEndpoint(ep *corev1.Endpoints) (*corev1.Endpoints, error) {
	ep, err := prom.kubeCli.CoreV1().Endpoints(ep.Namespace).Update(context.TODO(), ep, metav1.UpdateOptions{})
	if err != nil {
		klog.Errorf("failed to update etcd ep ,namespaces is %s,name is %s,error is %v", ep.Namespace, ep.Name, err)
		return nil, err
	}
	return ep, err
}

// CreateEtcdEndpoint creates endpoints
func (prom *PrometheusMonitor) CreateEtcdEndpoint(ep *corev1.Endpoints) (*corev1.Endpoints, error) {
	newEp, err := prom.kubeCli.CoreV1().Endpoints(ep.Namespace).Create(context.TODO(), ep, metav1.CreateOptions{})
	if err != nil {
		klog.Errorf("create etcd endpoint ,namespaces is %s,name is %s,error is %v", ep.Namespace, ep.Name, err)
		return nil, err
	}
	return newEp, err
}

// GetServiceMonitorTask gets service monitor by namespace and name
func (prom *PrometheusMonitor) GetServiceMonitorTask(namespace, name string) (*promapiv1.ServiceMonitor, error) {
	task, err := prom.promCli.ServiceMonitors(namespace).Get(context.TODO(), name, metav1.GetOptions{})
	if err != nil {
		klog.Errorf("get service monitor,namespaces is %s,name is %s,error is %v", namespace, name, err)
		return nil, err
	}
	return task, err
}

// UpdateServiceMonitorSpec updates spec of service monitor
func (prom *PrometheusMonitor) UpdateServiceMonitorSpec(task *promapiv1.ServiceMonitor) (
	*promapiv1.ServiceMonitor,
	error) {
	newTask, err := prom.promCli.ServiceMonitors(task.Namespace).Update(context.TODO(), task, metav1.UpdateOptions{})
	if err != nil {
		klog.Errorf("update service monitor,namespaces is %s,name is %s,error is %v", task.Namespace, task.Name, err)
		return nil, err
	}
	klog.V(6).Infof("update task %v spec succ", newTask)
	return newTask, err
}

// CreateServiceMonitor creates service monitor
func (prom *PrometheusMonitor) CreateServiceMonitor(task *promapiv1.ServiceMonitor) (*promapiv1.ServiceMonitor, error) {
	newTask, err := prom.promCli.ServiceMonitors(task.Namespace).Create(context.TODO(), task, metav1.CreateOptions{})
	if err != nil {
		klog.Errorf("create service monitor,namespaces is %s,name is %s,error is %v", task.Namespace, task.Name, err)
		return nil, err
	}
	klog.V(6).Infof("update task %v spec succ", newTask)
	return newTask, err
}

// UpdateServiceMonitor updates service monitor
func (prom *PrometheusMonitor) UpdateServiceMonitor(task *promapiv1.ServiceMonitor) (*promapiv1.ServiceMonitor, error) {
	newTask, err := prom.promCli.ServiceMonitors(task.Namespace).Update(context.TODO(), task, metav1.UpdateOptions{})
	if err != nil {
		klog.Errorf("update service monitor,namespaces is %s,name is %s,error is %v", task.Namespace, task.Name, err)
		return nil, err
	}
	klog.V(6).Infof("update task %v spec succ", newTask)
	return newTask, err
}

// DeleteServiceMonitor deletes service monitor
func (prom *PrometheusMonitor) DeleteServiceMonitor(namespace, name string) error {
	err := prom.promCli.ServiceMonitors(namespace).Delete(context.TODO(), name, metav1.DeleteOptions{})
	if err != nil {
		klog.Errorf("delete service monitor,namespaces is %s,name is %s,error is %v", namespace, name, err)
		return err
	}
	klog.V(6).Infof("delete service monitor %s succ", name)
	return err
}

// ServiceMonitorIsEquivalent compares old service monitor with new service monitor
func (prom *PrometheusMonitor) ServiceMonitorIsEquivalent(old, new *promapiv1.ServiceMonitor) bool {
	if !reflect.DeepEqual(old.Labels, new.Labels) {
		return false
	}
	if !reflect.DeepEqual(old.Spec, new.Spec) {
		return false
	}
	return true
}

// EndpointIsEquivalent compares old endpoints with new endpoints
func (prom *PrometheusMonitor) EndpointIsEquivalent(old, new *corev1.Endpoints) bool {
	if !reflect.DeepEqual(old.Labels, new.Labels) {
		return false
	}

	if !reflect.DeepEqual(old.Subsets, new.Subsets) {
		return false
	}
	return true
}

// ServiceIsEquivalent compares old service with old service
func (prom *PrometheusMonitor) ServiceIsEquivalent(old, new *corev1.Service) bool {
	if !reflect.DeepEqual(old.Labels, new.Labels) {
		return false
	}
	if !reflect.DeepEqual(old.Spec, new.Spec) {
		return false
	}
	return true
}

// UnpackEndPointSubsets unpacks endpoint subnets
func (prom *PrometheusMonitor) UnpackEndPointSubsets(endpoint *corev1.Endpoints) ([]string, error) {
	var addrs []string
	for _, subset := range endpoint.Subsets {
		for _, ipAddr := range subset.Addresses {
			for _, port := range subset.Ports {
				s := ipAddr.IP + ":" + strconv.Itoa(int(port.Port))
				addrs = append(addrs, s)
			}
		}
	}
	sort.Strings(addrs)
	return addrs, nil
}

func (prom *PrometheusMonitor) IsMonitorEnabled(cluster *kstonev1alpha2.EtcdCluster) bool {
	return featureutil.IsFeatureGateEnabled(cluster.ObjectMeta.Annotations, kstonev1alpha2.KStoneFeatureMonitor)
}

// Equal checks to Update ServiceMonitor & svc & ep, when label & memberIp change, update
func (prom *PrometheusMonitor) Equal(cluster *kstonev1alpha2.EtcdCluster) bool {
	var epAddrs []string
	epLabels := make(map[string]string)

	var clusterEndpoints []string
	for _, m := range cluster.Status.Members {
		items := strings.Split(m.ExtensionClientUrl, ":")
		endPoint := strings.TrimPrefix(items[1], "//")
		clusterEndpoints = append(clusterEndpoints, endPoint+":"+items[2])
	}

	endpoints, err := prom.GetEtcdEndpoint(DefaultEtcdPromNamespace, cluster.Name)
	if err != nil {
		if !apierrors.IsNotFound(err) {
			klog.Errorf("get endpoint failed, err is %v", err)
			return false
		}
	} else {
		epAddrs, err = prom.UnpackEndPointSubsets(endpoints)
		if err != nil {
			klog.Errorf("unpack endpoint failed, err is %v", err)
			return false
		}
	}

	if reflect.DeepEqual(epLabels, cluster.ObjectMeta.Labels) &&
		reflect.DeepEqual(epAddrs, clusterEndpoints) {
		return false
	}
	return true
}

// initEtcdServiceMonitor inits cluster serviceMonitor
func (prom *PrometheusMonitor) initEtcdServiceMonitor(cluster *kstonev1alpha2.EtcdCluster) (
	*promapiv1.ServiceMonitor,
	error) {
	certName := cluster.ObjectMeta.Annotations[util.ClusterTLSSecretName]
	endpointList := make([]promapiv1.Endpoint, 0)

	if certName == "" {
		certName = DefaultEtcdV3SecretName
	}
	relabel := &promapiv1.RelabelConfig{
		Action: "labelmap",
		Regex:  "__meta_kubernetes_service_label_(.+)",
	}
	replaceRelabel := &promapiv1.RelabelConfig{
		Action:      "replace",
		Regex:       "(.*)-(.*)-(.*)-(.*)",
		Replacement: "$1.$2.$3.$4",
		SourceLabels: []string{
			"endpoint",
		},
		TargetLabel: "endpoint",
	}

	scheme := "http"
	if strings.HasPrefix(cluster.Status.ServiceName, "https") {
		scheme = "https"
	}

	secretName := certName
	if strings.Contains(secretName, "/") {
		secretName = strings.Split(secretName, "/")[1]
	}
	for _, memberStatus := range cluster.Status.Members {
		endpoint := promapiv1.Endpoint{
			Port:     strings.ReplaceAll(memberStatus.Endpoint, ".", "-"),
			Scheme:   scheme,
			Interval: "30s",
			RelabelConfigs: []*promapiv1.RelabelConfig{
				relabel,
				replaceRelabel,
			},
			ProxyURL: nil,
		}
		if scheme == "https" {
			endpoint.TLSConfig = &promapiv1.TLSConfig{
				SafeTLSConfig: promapiv1.SafeTLSConfig{
					CA: promapiv1.SecretOrConfigMap{
						Secret: &k8sV1.SecretKeySelector{
							LocalObjectReference: k8sV1.LocalObjectReference{
								Name: secretName,
							},
							Key: etcd.CliCAFile,
						},
					},
					Cert: promapiv1.SecretOrConfigMap{
						Secret: &k8sV1.SecretKeySelector{
							LocalObjectReference: k8sV1.LocalObjectReference{
								Name: secretName,
							},
							Key: etcd.CliCertFile,
						},
					},
					KeySecret: &k8sV1.SecretKeySelector{
						LocalObjectReference: k8sV1.LocalObjectReference{
							Name: secretName,
						},
						Key: etcd.CliKeyFile,
					},
					InsecureSkipVerify: true,
				},
			}
		}
		endpointList = append(endpointList, endpoint)
	}

	labels := cluster.ObjectMeta.Labels
	labels["etcdName"] = cluster.Name
	servicemonitor := &promapiv1.ServiceMonitor{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.Name,
			Namespace: DefaultEtcdPromNamespace,
			Labels:    labels,
		},
		Spec: promapiv1.ServiceMonitorSpec{
			Endpoints:         endpointList,
			NamespaceSelector: promapiv1.NamespaceSelector{MatchNames: []string{DefaultEtcdPromNamespace}},
			Selector: metav1.LabelSelector{MatchLabels: map[string]string{
				"etcdName": cluster.Name,
			}},
		},
	}

	err := controllerutil.SetOwnerReference(cluster, servicemonitor, platformscheme.Scheme)
	if err != nil {
		return nil, err
	}

	return servicemonitor, nil
}

// initEtcdSvc inits cluster svc
func (prom *PrometheusMonitor) initEtcdSvc(cluster *kstonev1alpha2.EtcdCluster) (*corev1.Service, error) {
	portList := make([]corev1.ServicePort, 0)
	count := 0
	for _, m := range cluster.Status.Members {
		addr := strings.Split(m.ExtensionClientUrl, ":")
		port, _ := strconv.Atoi(addr[2])
		portName := strings.ReplaceAll(m.Endpoint, ".", "-")

		if len(portList) > 0 {
			if portName == portList[len(portList)-1].Name {
				continue
			}
		}

		servicePort := corev1.ServicePort{
			Name:       portName,
			Protocol:   corev1.ProtocolTCP,
			Port:       int32(2379 + count),
			TargetPort: intstr.FromInt(port),
		}
		portList = append(portList, servicePort)
		count++
	}

	labelMap := cluster.ObjectMeta.Labels
	labelMap["etcdName"] = cluster.Name

	svr := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.Name,
			Namespace: DefaultEtcdPromNamespace,
			Labels:    labelMap,
		},
		Spec: corev1.ServiceSpec{
			Ports: portList,
			Type:  corev1.ServiceTypeClusterIP,
		},
	}

	if cluster.Spec.ClusterType == kstonev1alpha2.EtcdClusterKstone {
		svr.Spec.Selector = map[string]string{
			"etcdcluster.etcd.tkestack.io/cluster-name": cluster.Name,
		}
	}

	err := controllerutil.SetOwnerReference(cluster, svr, platformscheme.Scheme)
	if err != nil {
		klog.Errorf("set reference failed, err is %s, service is %s", svr.Name)
		return nil, err
	}

	return svr, nil
}

// initEtcdEndpoint inits cluster ep
func (prom *PrometheusMonitor) initEtcdEndpoint(cluster *kstonev1alpha2.EtcdCluster) (*corev1.Endpoints, error) {
	subsets := make([]corev1.EndpointSubset, 0)
	if cluster.Spec.ClusterType == kstonev1alpha2.EtcdClusterImported {
		for _, m := range cluster.Status.Members {
			addr := strings.Split(m.ExtensionClientUrl, ":")
			port, err := strconv.Atoi(addr[2])
			if err != nil {
				klog.Errorf("failed to convert port string to int: %v", err)
				return nil, err
			}
			s := corev1.EndpointSubset{
				Addresses: []corev1.EndpointAddress{
					{
						IP: strings.TrimPrefix(addr[1], "//"),
					},
				},
				Ports: []corev1.EndpointPort{
					{
						Name:     strings.ReplaceAll(m.Endpoint, ".", "-"),
						Protocol: corev1.ProtocolTCP,
						Port:     int32(port),
					},
				},
			}
			subsets = append(subsets, s)
		}
	}
	ep := &corev1.Endpoints{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cluster.Name,
			Namespace: DefaultEtcdPromNamespace,
			Labels:    cluster.ObjectMeta.Labels,
		},
		Subsets: subsets,
	}
	return ep, nil
}

// CheckEqualIfDisabled Checks whether the monitoring resource has been deleted if monitor feature is disabled.
func (prom *PrometheusMonitor) CheckEqualIfDisabled(cluster *kstonev1alpha2.EtcdCluster) bool {
	_, err := prom.GetServiceMonitorTask(cluster.Namespace, cluster.Name)
	if err != nil && apierrors.IsNotFound(err) {
		_, err = prom.GetEtcdService(cluster.Namespace, cluster.Name)
		if err != nil && apierrors.IsNotFound(err) {
			return true
		}
	}
	return false
}

// CheckEqualIfEnabled check whether the desired monitoring resources are consistent with the actual resources,
// if monitor feature is enabled.
func (prom *PrometheusMonitor) CheckEqualIfEnabled(cluster *kstonev1alpha2.EtcdCluster) bool {
	var epAddrs []string
	epLabels := make(map[string]string)

	var clusterEndpoints []string
	for _, m := range cluster.Status.Members {
		items := strings.Split(m.ExtensionClientUrl, ":")
		endPoint := strings.TrimPrefix(items[1], "//")
		clusterEndpoints = append(clusterEndpoints, endPoint+":"+items[2])
	}

	endpoints, err := prom.GetEtcdEndpoint(DefaultEtcdPromNamespace, cluster.Name)
	if err != nil {
		klog.Errorf("get endpoint failed, cluster name %s,err is %v", cluster.Name, err)
		return false
	}
	if epAddrs, err = prom.UnpackEndPointSubsets(endpoints); err != nil {
		klog.Errorf("unpack endpoint failed, err is %v", err)
		return false
	}

	if reflect.DeepEqual(epLabels, cluster.ObjectMeta.Labels) &&
		reflect.DeepEqual(epAddrs, clusterEndpoints) {
		return true
	}
	return false
}

// CleanPrometheusMonitor cleans prometheus monitor for etcdcluster if it is disabled
func (prom *PrometheusMonitor) CleanPrometheusMonitor(cluster *kstonev1alpha2.EtcdCluster) error {
	if err := prom.DeleteServiceMonitor(cluster.Namespace, cluster.Name); err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	if err := prom.DeleteEtcdService(cluster.Namespace, cluster.Name); err != nil && !apierrors.IsNotFound(err) {
		return err
	}
	return nil
}

// SyncPrometheusMonitor syncs prometheus monitor for etcdcluster if it is enabled
func (prom *PrometheusMonitor) SyncPrometheusMonitor(cluster *kstonev1alpha2.EtcdCluster) error {
	taskName := cluster.Name

	// 1 init service
	newSvc, nErr := prom.initEtcdSvc(cluster)
	if nErr != nil {
		klog.Errorf("init etcdSvc failed, err is %v, cluster is %s", nErr, taskName)
		return nErr
	}
	curSvc, err := prom.GetEtcdService(DefaultEtcdPromNamespace, taskName)
	if apierrors.IsNotFound(err) {
		_, err = prom.CreateEtcdService(newSvc)
		if err != nil {
			klog.Errorf("create etcd %s svr failed:%v", taskName, err)
			return err
		}
	} else if err != nil {
		klog.Errorf("get etcd %s svr failed:%v", taskName, err)
		return err
	} else if !prom.ServiceIsEquivalent(curSvc, newSvc) {
		newSvc.Spec.ClusterIP = curSvc.Spec.ClusterIP
		newSvc.ResourceVersion = curSvc.ResourceVersion
		_, err = prom.UpdateEtcdService(newSvc)
		if err != nil {
			klog.Errorf("failed to update etcd %s svr failed:%v", taskName, err)
			return err
		}
	}

	// 2 init ep
	if cluster.Spec.ClusterType == kstonev1alpha2.EtcdClusterImported {
		newEp, err := prom.initEtcdEndpoint(cluster)
		if err != nil {
			return err
		}
		curEp, err := prom.GetEtcdEndpoint(DefaultEtcdPromNamespace, taskName)
		if apierrors.IsNotFound(err) {
			_, err = prom.CreateEtcdEndpoint(newEp)
			if err != nil {
				klog.Errorf("create etcd %s ep failed:%v", taskName, err)
				return err
			}
		} else if err != nil {
			klog.Errorf("get etcd %s ep failed:%v", taskName, err)
			return err
		} else if !prom.EndpointIsEquivalent(curEp, newEp) {
			newEp.ResourceVersion = curEp.ResourceVersion
			_, err = prom.UpdateEtcdEndpoint(newEp)
			if err != nil {
				klog.Errorf("failed to update etcd %s ep,err is %v", taskName, err)
				return err
			}
		}
	}

	// 3 init servicemonitor
	newServiceMonitor, err := prom.initEtcdServiceMonitor(cluster)
	if err != nil {
		return err
	}
	curServiceMonitor, err := prom.GetServiceMonitorTask(DefaultEtcdPromNamespace, taskName)
	if apierrors.IsNotFound(err) {
		_, err = prom.CreateServiceMonitor(newServiceMonitor)
		if err != nil {
			klog.Errorf("create etcd %s service monitor failed:%v", taskName, err)
			return err
		}
	} else if err != nil {
		klog.Errorf("get etcd %s service monitor failed:%v", taskName, err)
		return err
	} else if !prom.ServiceMonitorIsEquivalent(curServiceMonitor, newServiceMonitor) {
		newServiceMonitor.ResourceVersion = curServiceMonitor.ResourceVersion
		_, err = prom.UpdateServiceMonitor(newServiceMonitor)
		if err != nil {
			klog.Errorf("failed to update etcd %s service monitor,err is %v", taskName, err)
			return err
		}
	}

	klog.V(2).Infof("add etcd task %s succ", taskName)
	return err
}
