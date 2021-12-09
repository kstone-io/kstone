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

package etcdinspection

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-martini/martini"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/kubernetes/scheme"
	typedcorev1 "k8s.io/client-go/kubernetes/typed/core/v1"
	"k8s.io/client-go/tools/cache"
	"k8s.io/client-go/tools/record"
	"k8s.io/client-go/util/workqueue"
	klog "k8s.io/klog/v2"

	kstonev1alpha1 "tkestack.io/kstone/pkg/apis/kstone/v1alpha1"
	// register etcd cluster providers
	_ "tkestack.io/kstone/pkg/clusterprovider/providers"
	"tkestack.io/kstone/pkg/controllers/util"
	"tkestack.io/kstone/pkg/featureprovider"
	// register etcd feature providers
	_ "tkestack.io/kstone/pkg/featureprovider/providers"
	clientset "tkestack.io/kstone/pkg/generated/clientset/versioned"
	platformscheme "tkestack.io/kstone/pkg/generated/clientset/versioned/scheme"
	informers "tkestack.io/kstone/pkg/generated/informers/externalversions/kstone/v1alpha1"
	listers "tkestack.io/kstone/pkg/generated/listers/kstone/v1alpha1"
)

// InspectionController is the controller implementation for etcdinspection resources
type InspectionController struct {
	// kubeclientset is a standard kubernetes clientset
	kubeclientset kubernetes.Interface
	// platformclientset is a clientset for our own API group
	platformclientset clientset.Interface

	etcdinspectionLister listers.EtcdInspectionLister
	etcdinspectionSynced cache.InformerSynced
	// To allow injection of doClusterInspection for testing.
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
}

func NewInspectionControllerMetric() http.Handler {
	m := martini.New()
	r := martini.NewRouter()
	r.Get("/health", func() (int, string) {
		return 200, "ok"
	})
	r.Get("/metrics", promhttp.Handler())
	m.MapTo(r, (*martini.Routes)(nil))
	m.Action(r.Handle)
	return m
}

// NewEtcdInspectionController returns a new etcdinspection controller
func NewEtcdInspectionController(
	clientbuilder util.ClientBuilder,
	kubeclientset kubernetes.Interface,
	platformclientset clientset.Interface,
	etcdinspectionInformer informers.EtcdInspectionInformer) *InspectionController {

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
		corev1.EventSource{Component: util.ComponentEtcdInspectionController},
	)

	controller := &InspectionController{
		clientbuilder:        clientbuilder,
		kubeclientset:        kubeclientset,
		platformclientset:    platformclientset,
		etcdinspectionLister: etcdinspectionInformer.Lister(),
		etcdinspectionSynced: etcdinspectionInformer.Informer().HasSynced,
		workqueue: workqueue.NewNamedRateLimitingQueue(
			workqueue.DefaultControllerRateLimiter(),
			"etcdinspections",
		),
		recorder: recorder,
	}
	controller.syncHandler = controller.doClusterInspection

	klog.Info("Setting up event handlers")
	// Set up an event handler for when etcdinspection resources change
	etcdinspectionInformer.Informer().AddEventHandler(cache.ResourceEventHandlerFuncs{
		AddFunc: controller.enqueueEtcdInspection,
		UpdateFunc: func(old, new interface{}) {
			controller.enqueueEtcdInspection(new)
		},
	})

	return controller
}

// Run will set up the event handlers for types we are interested in, as well
// as syncing informer caches and starting workers. It will block until stopCh
// is closed, at which point it will shutdown the workqueue and wait for
// workers to finish processing their current work items.
func (c *InspectionController) Run(threadiness int, stopCh <-chan struct{}) error {
	defer utilruntime.HandleCrash()
	defer c.workqueue.ShutDown()

	// Start the informer factories to begin populating the informer caches
	klog.Info("Starting etcdinspection controller")

	// Wait for the caches to be synced before starting workers
	klog.Info("Waiting for informer caches to sync")
	if ok := cache.WaitForCacheSync(stopCh, c.etcdinspectionSynced); !ok {
		return fmt.Errorf("failed to wait for caches to sync")
	}

	klog.Info("Starting workers")
	// Launch two workers to process etcdinspection resources
	for i := 0; i < threadiness; i++ {
		go wait.Until(c.runWorker, time.Second, stopCh)
	}

	go func() {
		err := http.ListenAndServe(":9090", NewInspectionControllerMetric())
		if err != nil {
			klog.Errorf("listenAndServer error is %v", err)
		}
	}()

	klog.Info("Started workers")
	<-stopCh
	klog.Info("Shutting down workers")

	return nil
}

// runWorker is a long-running function that will continually call the
// processNextWorkItem function in order to read and process a message on the
// workqueue.
func (c *InspectionController) runWorker() {
	for c.processNextWorkItem() {
	}
}

// processNextWorkItem will read a single work item off the workqueue and
// attempt to process it, by calling the syncHandler.
func (c *InspectionController) processNextWorkItem() bool {
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

func (c *InspectionController) doClusterInspection(key string) error {
	// Convert the namespace/name string into a distinct namespace and name
	namespace, name, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		utilruntime.HandleError(fmt.Errorf("invalid resource key: %s", key))
		return nil
	}

	// Get the etcdinspection resource with this namespace/name
	etcdinspection, err := c.etcdinspectionLister.EtcdInspections(namespace).Get(name)
	if err != nil {
		// The etcdinspection resource may no longer exist, in which case we stop
		// processing.
		if errors.IsNotFound(err) {
			utilruntime.HandleError(fmt.Errorf("etcdinspection '%s' in work queue no longer exists", key))
			return nil
		}
		return err
	}

	return c.doInspectionTask(etcdinspection)
}

// enqueueEtcdInspection takes a etcdinspection resource and converts it into a namespace/name
// string which is then put onto the work queue. This method should *not* be
// passed resources of any type other than etcdinspection.
func (c *InspectionController) enqueueEtcdInspection(obj interface{}) {
	var key string
	var err error
	if key, err = cache.MetaNamespaceKeyFunc(obj); err != nil {
		utilruntime.HandleError(err)
		return
	}
	c.workqueue.Add(key)
}

func (c *InspectionController) GetInspectionFeatureProvider(name string) (featureprovider.Feature, error) {
	ctx := &featureprovider.FeatureContext{Clientbuilder: c.clientbuilder}
	feature, err := featureprovider.GetFeatureProvider(name, ctx)
	if err != nil {
		return nil, err
	}
	return feature, nil
}

func (c *InspectionController) doInspectionTask(etcdinspection *kstonev1alpha1.EtcdInspection) error {
	inspectionType := etcdinspection.Spec.InspectionType
	feature, err := c.GetInspectionFeatureProvider(inspectionType)
	if err != nil {
		return err
	}
	if err = feature.Init(); err != nil {
		klog.Errorf("failed to init feature %s provider, err is %v", inspectionType, err)
		return err
	}
	return feature.Do(etcdinspection)
}
