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

package etcdclustercontroller

import (
	"context"
	"io"
	"net/http"
	"os"
	"time"

	// import http pprof
	_ "net/http/pprof"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/uuid"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
	klog "k8s.io/klog/v2"

	"tkestack.io/kstone/pkg/controllers/etcdcluster"
	"tkestack.io/kstone/pkg/controllers/util"
	"tkestack.io/kstone/pkg/k8s"
	"tkestack.io/kstone/pkg/signals"
)

type EtcdClusterCommand struct {
	out                io.Writer
	kubeconfig         string
	masterURL          string
	labelSelector      string
	leaseLockName      string
	leaseLockNamespace string
	enableProfiling    bool
}

// NewEtcdClusterControllerCommand creates a *cobra.Command object with default parameters
func NewEtcdClusterControllerCommand(out io.Writer) *cobra.Command {
	cc := &EtcdClusterCommand{out: out}
	cmd := &cobra.Command{
		Use:   "etcdcluster",
		Short: "run etcdcluster controller",
		Long: `The etcdcluster controller is a daemon, it will watches the changes of EtcdCluster resources 
through the apiserver and makes changes attempting to move the current state towards the desired state.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.Flags().VisitAll(func(flag *pflag.Flag) {
				klog.V(1).Infof("FLAG: --%s=%q", flag.Name, flag.Value)
			})
			if err := cc.Run(); err != nil {
				return err
			}
			return nil
		},
	}

	fs := cmd.PersistentFlags()
	cc.AddFlags(fs)
	return cmd
}

// Run start etcdcluster controller
func (c *EtcdClusterCommand) Run() error {
	stopCh := signals.SetupSignalHandler()

	config, err := clientcmd.BuildConfigFromFlags(c.masterURL, c.kubeconfig)
	if err != nil {
		klog.Fatalf("Error to build kube config: %v", err)
		return err
	}

	kubeClient, clusterClient, kubeInformerFactory, informerFactory, err := k8s.GenerateInformer(config, c.labelSelector)
	if err != nil {
		klog.Fatalf("Error to generate informer: %v", err)
		return err
	}

	controller := etcdcluster.NewEtcdclusterController(
		util.NewSimpleClientBuilder(c.kubeconfig),
		kubeClient,
		clusterClient,
		kubeInformerFactory.Core().V1().Secrets(),
		informerFactory.Kstone().V1alpha2().EtcdClusters(),
	)
	// notice that there is no need to run Start methods in a separate goroutine.
	// (i.e. go kubeInformerFactory.Start(stopCh)
	// Start method is non-blocking and runs all registered informers in a dedicated goroutine.
	kubeInformerFactory.Start(stopCh)
	informerFactory.Start(stopCh)

	leaderElectionConfig, err := c.makeLeaderElectionConfig(kubeClient, controller, stopCh)
	if err != nil {
		klog.Fatalf("Error to generate leader election config: %v", err)
		return err
	}

	// use a Go context so we can tell the leaderelection code when we
	// want to step down
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if c.enableProfiling {
		go func() {
			addr := "0.0.0.0:6060"
			klog.Infof("Listen on %s for profiling", addr)
			klog.Error(http.ListenAndServe(addr, nil))
		}()
	}

	go func() {
		<-stopCh
		klog.Info("Received termination, signaling shutdown")
		cancel()
	}()

	// start the leader election code loop
	leaderelection.RunOrDie(ctx, *leaderElectionConfig)
	return err
}

func (c *EtcdClusterCommand) AddFlags(fs *pflag.FlagSet) {
	fs.StringVarP(
		&c.kubeconfig,
		"kubeconfig",
		"k",
		"",
		"force to specify the kubeconfig",
	)
	fs.StringVar(
		&c.masterURL,
		"master",
		"",
		"The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.",
	)
	fs.StringVar(
		&c.labelSelector,
		"labelSelector",
		"",
		"The address of the Kubernetes API server. Overrides any value in kubeconfig. Only required if out-of-cluster.",
	)
	fs.StringVar(
		&c.leaseLockName,
		"lease-lock-name",
		"kstone-etcdcluster-controller",
		"the lease lock resource name",
	)
	fs.StringVar(&c.leaseLockNamespace,
		"lease-lock-namespace",
		"kstone",
		"the lease lock resource namespace",
	)
	fs.BoolVar(&c.enableProfiling,
		"profiling",
		true,
		"enable profiling via web interface host:port/debug/pprof/.")
}

func (c *EtcdClusterCommand) makeLeaderElectionConfig(kubeClient *kubernetes.Clientset, controller *etcdcluster.ClusterController, stopCh <-chan struct{}) (*leaderelection.LeaderElectionConfig, error) {
	if c.leaseLockName == "" {
		klog.Fatal("unable to get lease lock resource name (missing lease-lock-name flag).")
	}
	if c.leaseLockNamespace == "" {
		klog.Fatal("unable to get lease lock resource namespace (missing lease-lock-namespace flag).")
	}

	hostname, err := os.Hostname()
	if err != nil {
		klog.Errorf("unable to get hostname: %v", err)
		return nil, err
	}
	// add a uniquifier so that two processes on the same host don't accidentally both become active
	id := hostname + "_" + string(uuid.NewUUID())

	// we use the Lease lock type since edits to Leases are less common
	// and fewer objects in the cluster watch "all Leases".
	lock := &resourcelock.LeaseLock{
		LeaseMeta: metav1.ObjectMeta{
			Name:      c.leaseLockName,
			Namespace: c.leaseLockNamespace,
		},
		Client: kubeClient.CoordinationV1(),
		LockConfig: resourcelock.ResourceLockConfig{
			Identity: id,
		},
	}

	// start the leader election code loop
	return &leaderelection.LeaderElectionConfig{
		Lock: lock,
		// IMPORTANT: you MUST ensure that any code you have that
		// is protected by the lease must terminate **before**
		// you call cancel. Otherwise, you could have a background
		// loop still running and another process could
		// get elected before your background loop finished, violating
		// the stated goal of the lease.
		ReleaseOnCancel: true,
		LeaseDuration:   60 * time.Second,
		RenewDeadline:   15 * time.Second,
		RetryPeriod:     5 * time.Second,
		Callbacks: leaderelection.LeaderCallbacks{
			OnStartedLeading: func(ctx context.Context) {
				// we're notified when we start - this is where you would
				// usually put your code
				if err := controller.Run(2, stopCh); err != nil {
					klog.Fatalf("Error running etcd controller: %s", err.Error())
				}
			},
			OnStoppedLeading: func() {
				// we can do cleanup here
				klog.Infof("leader lost: %s", id)
				os.Exit(0)
			},
			OnNewLeader: func(identity string) {
				// we're notified when new leader elected
				if identity == id {
					// I just got the lock
					return
				}
				klog.Infof("new leader elected: %s", identity)
			},
		},
	}, nil

}
