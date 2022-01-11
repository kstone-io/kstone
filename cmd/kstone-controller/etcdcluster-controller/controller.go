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
	"io"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/client-go/tools/clientcmd"
	klog "k8s.io/klog/v2"

	"tkestack.io/kstone/pkg/controllers/etcdcluster"
	"tkestack.io/kstone/pkg/controllers/util"
	"tkestack.io/kstone/pkg/k8s"
	"tkestack.io/kstone/pkg/signals"
)

type EtcdClusterCommand struct {
	out           io.Writer
	kubeconfig    string
	masterURL     string
	labelSelector string
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

	if err = controller.Run(2, stopCh); err != nil {
		klog.Fatalf("Error running etcd controller: %s", err.Error())
		return err
	}
	return nil
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
}
