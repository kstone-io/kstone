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

package etcdcluster

import (
	"context"
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	"k8s.io/klog/v2"

	kstonev1alpha1 "tkestack.io/kstone/pkg/apis/kstone/v1alpha1"
	"tkestack.io/kstone/pkg/clusterprovider"
	// register cluster provider
	_ "tkestack.io/kstone/pkg/clusterprovider/providers"
	"tkestack.io/kstone/pkg/controllers/util"
	"tkestack.io/kstone/pkg/etcd"
	"tkestack.io/kstone/pkg/featureprovider"
	// register feature provider
	_ "tkestack.io/kstone/pkg/featureprovider/providers"
	clientset "tkestack.io/kstone/pkg/generated/clientset/versioned"
	platformscheme "tkestack.io/kstone/pkg/generated/clientset/versioned/scheme"
	informers "tkestack.io/kstone/pkg/generated/informers/externalversions/kstone/v1alpha1"
	listers "tkestack.io/kstone/pkg/generated/listers/kstone/v1alpha1"
)

// ClusterController is the controller implementation for EtcdCluster resources
type ClusterController struct {
	// kubeclientset is a standard kubernetes clientset
	kubeclientset kubernetes.Interface
	// platformclientset is a clientset for our own API group
	platformclientset clientset.Interface

	etcdclusterLister listers.EtcdClusterLister
	etcdclusterSynced cache.InformerSynced

	// To allow injection of syncEtcdCluster for testing.
	syncHandler func(eKey string) error

	// workqueue is a rate limited work queue. This is used to queue work to be
	// processed instead of performing it as soon as a change happens. This
	// means we can ensure we only process a fixed amount of resources at a
	// time, and makes it easy to ensure we are never processing the same item
	// simultaneously in two different workers.
	workqueue workqueue.RateLimitingInterface
	// recorder is an event recorder for recording Event resources to the
	// Kubernetes API.
	recorder record.EventRecorder

	clientbuilder util.ClientBuilder
	tlsGetter     etcd.TLSGetter
}

// NewEtcdclusterController returns a new etcdcluster controller
func NewEtcdclusterController(
	clientbuilder util.ClientBuilder,
	kubeclientset kubernetes.Interface,
	platformclientset clientset.Interface,
	etcdclusterInformer informers.EtcdClusterInformer) *ClusterController {

	// Create event broadcaster
	// Add kstone types to the default Kubernetes Scheme so Events can be
	// logged for kstone types.
	utilruntime.Must(platformscheme.AddToScheme(scheme.Scheme))
	klog.V(4).Info("Creating event broadcaster")
	eventBroadcaster := record.NewBroadcaster()
	eventBroadcaster.StartStructuredLogging(0)
	eventBroadcaster.StartRecordingToSink(&typedcorev1.EventSinkImpl{Interface: kubeclientset.CoreV1().Events("")})
	recorder := eventBroadcaster.NewRecorder(
		scheme.Scheme,
		corev1.EventSource{Component: util.ComponentEtcdClusterController},
	)

	controller := &ClusterController{
		clientbuilder:     clientbuilder,
		kubeclientset:     kubeclientset,
		platformclientset: platformclientset,
		etcdclusterLister: etcdclusterInformer.Lister(),
		etcdclusterSynced: etcdclusterInformer.Informer().HasSynced,
		workqueue:         workqueue.NewNamedRateLimitingQueue(workqueue.DefaultControllerRateLimiter(), "EtcdClusters"),
		recorder:          recorder,
	}

	controller.syncHandler = controller.syncEtcdCluster
	controller.tlsGetter = etcd.NewTLSSecretGetter(clientbuilder)

	klog.Info("Setting up event handlers")
	// Set up an event handler for when EtcdCluster resources change
	etcdclusterInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueEtcdcluster,
		UpdateFunc: func(old, new interface{}) {
			controller.enqueueEtcdcluster(new)
		},
	})

	return controller
}

// Run will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (c *ClusterController) Run(threadiness int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()

	// Start the informer factories to begin populating the informer caches
	klog.Info("Starting EtcdCluster controller")

	// Wait for the caches to be synced before starting workers
	klog.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, c.etcdclusterSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	klog.Info("Starting workers")
	// Launch two workers to process EtcdCluster resources
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	klog.Info("Started workers")
	<-stopCh
	klog.Info("Shutting down workers")

	return nil
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (c *ClusterController) runWorker() {
	for c.processNextWorkItem() {
	}
}

// processNextWorkItem will read a single work item off the workqueue and
// attempt to process it, by calling the syncHandler.
func (c *ClusterController) processNextWorkItem() bool {
	obj, shutdown := c.workqueue.Get()

	if shutdown {
		return false
	}

	// We wrap this block in a func so we can defer c.workqueue.Done.
	err := util.ProcessWorkQueue(c.workqueue, c.syncHandler, obj)
	if err != nil {
		utilruntime.HandleError(err)
		return true
	}

	return true
}

// syncEtcdCluster compares the actual state with the desired, and attempts to
// converge the two. It then updates the Status block of the EtcdCluster resource
// with the current status of the resource.
func (c *ClusterController) syncEtcdCluster(key string) error {
	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	// Get the EtcdCluster resource with this namespace/name
	etcdcluster, err := c.etcdclusterLister.EtcdClusters(namespace).Get(name)
	if err != nil {
		// The EtcdCluster resource may no longer exist, in which case we stop
		// processing.
		if errors.IsNotFound(err) {
			utilruntime.HandleError(fmt.Errorf("EtcdCluster '%s' in work queue no longer exists", key))
			return nil
		}
		return err
	}

	return c.reconcileEtcdCluster(etcdcluster.DeepCopy())
}

// updateEtcdClusterStatus
func (c *ClusterController) updateEtcdClusterStatus(cluster *kstonev1alpha1.EtcdCluster) (
	*kstonev1alpha1.EtcdCluster,
	error,
) {
	// NEVER modify objects from the store. It's a read-only, local cache.
	// You can use DeepCopy() to make a deep copy of original object and modify this copy
	// Or create a copy manually for better performance
	// clusterCopy := cluster.DeepCopy()
	// If the CustomResourceSubresources feature gate is not enabled,
	// we must use Update instead of UpdateStatus to update the Status block of the EtcdCluster resource.
	// UpdateStatus will not allow changes to the Spec of the resource,
	// which is ideal for ensuring nothing other than resource status has been updated.
	etcdcluster, err := c.platformclientset.KstoneV1alpha1().EtcdClusters(cluster.Namespace).
		Update(context.TODO(), cluster, metav1.UpdateOptions{})
	if err != nil {
		return cluster, err
	}
	return etcdcluster, nil
}

// enqueueEtcdcluster takes a EtcdCluster resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than EtcdCluster.
func (c *ClusterController) enqueueEtcdcluster(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.workqueue.Add(key)
}

func (c *ClusterController) handleClusterManagement(cluster *kstonev1alpha1.EtcdCluster) (
	*kstonev1alpha1.EtcdCluster,
	error,
) {
	// Get cluster provider
	provider, err := c.GetEtcdClusterProvider(cluster.Spec.ClusterType)
	if err != nil {
		klog.Errorf("failed to get cluster provider %s, err is %v, cluster is %s",
			cluster.Spec.ClusterType, err, cluster.Name)
		return cluster, err
	}
	nextAction, err := c.getDesiredAction(cluster, provider)
	if err != nil {
		return cluster, err
	}
	switch nextAction {
	case kstonev1alpha1.EtcdCluterCreating:
		cluster, err = c.handleClusterCreate(cluster, provider)
	case kstonev1alpha1.EtcdClusterUpdating:
		cluster, err = c.handleClusterUpdate(cluster, provider)
	default:
		cluster, err = c.handleClusterStatus(cluster, provider)
	}
	_, _ = c.updateEtcdClusterStatus(cluster)
	if err != nil {
		c.recorder.Eventf(
			cluster,
			corev1.EventTypeWarning,
			string(nextAction),
			"failed to set cluster %s, err is %v, cluster is %s",
			string(nextAction),
			err,
			cluster.ClusterName,
		)
		return cluster, err
	}
	return cluster, nil
}

func (c *ClusterController) enabledFeatureGate(annotations map[string]string, feature string) bool {
	if gates, found := annotations[kstonev1alpha1.KStoneFeatureAnno]; found && gates != "" {
		features := strings.Split(gates, ",")
		for _, f := range features {
			ff := strings.Split(f, "=")
			if len(ff) != 2 {
				continue
			}

			g, _ := strconv.ParseBool(ff[1])
			if ff[0] == feature && g {
				return true
			}
		}
	}
	return false
}

// handleClusterLabels updates EtcdCluster Resource labels.
func (c *ClusterController) handleClusterLabels(
	cluster *kstonev1alpha1.EtcdCluster) (*kstonev1alpha1.EtcdCluster, error) {
	annotations := cluster.ObjectMeta.Annotations
	if annotations == nil {
		annotations = make(map[string]string)
	}

	labels := cluster.ObjectMeta.Labels
	if labels == nil {
		labels = make(map[string]string)
	}
	for opsName := range featureprovider.EtcdFeatureProviders {
		labels[opsName] = strconv.FormatBool(c.enabledFeatureGate(annotations, opsName))
	}

	labels["clusterType"] = string(cluster.Spec.ClusterType)
	labels["version"] = cluster.Spec.Version
	if !reflect.DeepEqual(cluster.ObjectMeta.Labels, labels) {
		cluster.ObjectMeta.Labels = labels
		return c.updateEtcdClusterStatus(cluster)
	}
	return cluster, nil
}

func (c *ClusterController) handleClusterFeature(cluster *kstonev1alpha1.EtcdCluster) (
	*kstonev1alpha1.EtcdCluster,
	error) {
	// Check cluster status, ensure cluster is running
	if cluster.Status.Phase != kstonev1alpha1.EtcdClusterRunning || len(cluster.Status.Members) == 0 {
		klog.V(3).Infof("cluster status is not running, waiting")
		return cluster, nil
	}

	annotations := cluster.ObjectMeta.Annotations
	if annotations == nil {
		annotations = make(map[string]string)
	}

	if cluster.Status.FeatureGatesStatus == nil {
		cluster.Status.FeatureGatesStatus = make(map[kstonev1alpha1.KStoneFeature]string)
	}

	for name := range featureprovider.EtcdFeatureProviders {
		if !c.enabledFeatureGate(annotations, name) {
			klog.V(4).Infof("feature %s is disabled,skip it,cluster is %s", name, cluster.Name)
			continue
		}

		feature, err := c.GetFeatureProvider(name)
		if err != nil {
			klog.Errorf("failed to get feature %s provider, err is %v", name, err)
			continue
		}

		if !feature.Equal(cluster) {
			klog.V(4).Infof("skip feature %s,no changed, cluster is %s", name, cluster.Name)
			continue
		}
		featureName := kstonev1alpha1.KStoneFeature(name)

		err = feature.Sync(cluster)
		if err != nil {
			klog.Errorf("failed to enable %s feature, err is %v, cluster is %s", name, err, cluster.Name)
			cluster.Status.FeatureGatesStatus[featureName] = fmt.Sprintf("failed to enable %s feature, err is %v", name, err)
			continue
		}

		cluster.Status.FeatureGatesStatus[featureName] = "done"
	}

	return c.updateEtcdClusterStatus(cluster)
}

func (c *ClusterController) reconcileEtcdCluster(cluster *kstonev1alpha1.EtcdCluster) error {
	// Handle cluster Creation,Update operations
	cluster, err := c.handleClusterManagement(cluster)
	if err != nil {
		klog.Errorf("failed to handle cluster management operations, err is %v, cluster is %s", err, cluster.Name)
		return err
	}

	// If cluster is not running, do not proceed to the next step
	if cluster.Status.Phase != kstonev1alpha1.EtcdClusterRunning {
		klog.Warningf("cluster %s is not ready", cluster.Name)
		return nil
	}

	// Handle cluster labels
	cluster, err = c.handleClusterLabels(cluster)
	if err != nil {
		klog.Errorf("failed to handle cluster labels, err is %v, cluster is %s", err, cluster.Name)
		return err
	}

	// Handle cluster feature
	cluster, err = c.handleClusterFeature(cluster)
	if err != nil {
		klog.Errorf("failed to handle cluster feature, err is %v, cluster is %s", err, cluster.Name)
		return err
	}

	return nil
}

func (c *ClusterController) GetFeatureProvider(name string) (featureprovider.Feature, error) {
	ctx := &featureprovider.FeatureContext{Clientbuilder: c.clientbuilder}
	feature, err := featureprovider.GetFeatureProvider(name, ctx)
	if err != nil {
		return nil, err
	}
	return feature, nil
}

func (c *ClusterController) GetEtcdClusterProvider(name kstonev1alpha1.EtcdClusterType) (clusterprovider.Cluster, error) {
	ctx := &clusterprovider.ClusterContext{
		Clientbuilder: c.clientbuilder,
	}
	cluster, err := clusterprovider.GetEtcdClusterProvider(name, ctx)
	if err != nil {
		return nil, err
	}
	return cluster, nil
}

func (c *ClusterController) getDesiredAction(
	cluster *kstonev1alpha1.EtcdCluster,
	provider clusterprovider.Cluster,
) (kstonev1alpha1.EtcdClusterPhase, error) {
	if len(cluster.Status.Conditions) == 0 {
		return kstonev1alpha1.EtcdCluterCreating, nil
	}

	conditionIndex := len(cluster.Status.Conditions) - 1
	lastCondition := cluster.Status.Conditions[conditionIndex]

	switch lastCondition.Type {
	case kstonev1alpha1.EtcdClusterConditionCreate:
		if lastCondition.Status == corev1.ConditionFalse {
			return kstonev1alpha1.EtcdCluterCreating, nil
		}
	case kstonev1alpha1.EtcdClusterConditionUpdate:
		if lastCondition.Status == corev1.ConditionFalse {
			return kstonev1alpha1.EtcdClusterUpdating, nil
		}
	}

	equal, err := provider.Equal(cluster)
	if err != nil {
		klog.Errorf("failed to check if the cluster is equal, err is %v,cluster is %s", err, cluster.Name)
		return kstonev1alpha1.EtcdClusterUnknown, err
	}
	if !equal {
		klog.Infof("spec is different, need to update etcd, cluster is %s", cluster.Name)
		return kstonev1alpha1.EtcdClusterUpdating, nil
	}

	return kstonev1alpha1.EtcdClusterRunning, nil
}

func (c *ClusterController) generateConditions(
	conditions []kstonev1alpha1.EtcdClusterCondition,
	phase kstonev1alpha1.EtcdClusterPhase,
	nextConditionType kstonev1alpha1.EtcdClusterConditionType,
) []kstonev1alpha1.EtcdClusterCondition {
	conditionIndex := len(conditions) - 1
	if conditionIndex >= 0 {
		lastCondition := conditions[conditionIndex]
		if lastCondition.Status != corev1.ConditionTrue {
			return conditions
		}

		if lastCondition.Type == nextConditionType {
			conditions = conditions[:conditionIndex]
		}
	}

	conditions = append(conditions, kstonev1alpha1.EtcdClusterCondition{
		Type:      nextConditionType,
		Status:    corev1.ConditionFalse,
		StartTime: metav1.Now(),
	})

	return conditions
}

func (c *ClusterController) handleClusterCreate(
	cluster *kstonev1alpha1.EtcdCluster,
	provider clusterprovider.Cluster,
) (*kstonev1alpha1.EtcdCluster, error) {
	cluster.Status.Conditions = c.generateConditions(
		cluster.Status.Conditions,
		cluster.Status.Phase,
		kstonev1alpha1.EtcdClusterConditionCreate,
	)
	cluster.Status.Phase = kstonev1alpha1.EtcdCluterCreating

	conditionIndex := len(cluster.Status.Conditions) - 1

	err := provider.BeforeCreate(cluster)
	if err != nil {
		klog.Errorf("failed to do something before create, err is %v, cluster is %s", err, cluster.Name)
		cluster.Status.Conditions[conditionIndex].Reason = err.Error()
		return cluster, err
	}

	err = provider.Create(cluster)
	if err != nil {
		klog.Errorf("failed to create, err is %v, cluster is %s", err, cluster.Name)
		cluster.Status.Conditions[conditionIndex].Reason = err.Error()
		return cluster, err
	}

	err = provider.AfterCreate(cluster)
	if err != nil {
		klog.Errorf("failed to do something after create, err is %v, cluster is %s", err, cluster.Name)
		cluster.Status.Conditions[conditionIndex].Reason = err.Error()
		return cluster, err
	}

	cluster.Status.Conditions[conditionIndex].Reason = ""
	cluster.Status.Conditions[conditionIndex].EndTime = metav1.Now()
	cluster.Status.Conditions[conditionIndex].Status = corev1.ConditionTrue
	return cluster, nil
}

func (c *ClusterController) handleClusterUpdate(
	cluster *kstonev1alpha1.EtcdCluster,
	provider clusterprovider.Cluster,
) (*kstonev1alpha1.EtcdCluster, error) {
	cluster.Status.Conditions = c.generateConditions(
		cluster.Status.Conditions,
		cluster.Status.Phase,
		kstonev1alpha1.EtcdClusterConditionUpdate,
	)
	cluster.Status.Phase = kstonev1alpha1.EtcdClusterUpdating
	conditionIndex := len(cluster.Status.Conditions) - 1

	err := provider.BeforeUpdate(cluster)
	if err != nil {
		klog.Errorf("failed to do something before update, err is %v, cluster is %s", err, cluster.Name)
		cluster.Status.Conditions[conditionIndex].Reason = err.Error()
		return cluster, err
	}

	err = provider.Update(cluster)
	if err != nil {
		klog.Errorf("failed to update, err is %v, cluster is %s", err, cluster.Name)
		cluster.Status.Conditions[conditionIndex].Reason = err.Error()
		return cluster, err
	}

	err = provider.AfterUpdate(cluster)
	if err != nil {
		klog.Errorf("failed to do something after update, err is %v, cluster is %s", err, cluster.Name)
		cluster.Status.Conditions[conditionIndex].Reason = err.Error()
		return cluster, err
	}

	cluster.Status.Conditions[conditionIndex].Reason = ""
	cluster.Status.Conditions[conditionIndex].EndTime = metav1.Now()
	cluster.Status.Conditions[conditionIndex].Status = corev1.ConditionTrue
	return cluster, nil
}

// handleClusterStatus checks the status, if equal, updates status
// if not equal, updates etcdclusters.etcd.tkestack.io
func (c *ClusterController) handleClusterStatus(
	cluster *kstonev1alpha1.EtcdCluster,
	provider clusterprovider.Cluster,
) (*kstonev1alpha1.EtcdCluster, error) {
	// Check and update Cluster Status
	annotations := cluster.ObjectMeta.Annotations
	secretName := ""
	if annotations != nil {
		if _, found := annotations[util.ClusterTLSSecretName]; found {
			secretName = annotations[util.ClusterTLSSecretName]
		}
	}
	tlsConfig, err := c.tlsGetter.Config(cluster.Name, secretName)
	if err != nil {
		return cluster, err
	}

	status, err := provider.Status(tlsConfig, cluster)
	if err != nil {
		c.recorder.Eventf(
			cluster,
			corev1.EventTypeWarning,
			string(util.EtcdClusterUpdateStatus),
			"failed to get cluster status %v",
			err,
		)
	}
	cluster.Status = status

	return cluster, nil
}
