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

package etcdinspectioncontroller

import (
	"io"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/client-go/tools/clientcmd"
	klog "k8s.io/klog/v2"

	"tkestack.io/kstone/pkg/controllers/etcdinspection"
	"tkestack.io/kstone/pkg/controllers/util"
	"tkestack.io/kstone/pkg/k8s"
	"tkestack.io/kstone/pkg/signals"
)

type EtcdInspectionCommand struct {
	out           io.Writer
	kubeconfig    string
	masterURL     string
	labelSelector string
}

// NewEtcdInspectionControllerCommand creates a *cobra.Command object with default parameters
func NewEtcdInspectionControllerCommand(out io.Writer) *cobra.Command {
	cc := &EtcdInspectionCommand{out: out}
	cmd := &cobra.Command{
		Use:   "inspection",
		Short: "run inspection controller",
		Long: `The inspection controller is a daemon, it will watches the changes of etcdinspection resources 
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

// Run start etcdinspection controller
func (c *EtcdInspectionCommand) Run() error {
	stopCh := signals.SetupSignalHandler()
	config, err := clientcmd.BuildConfigFromFlags(c.masterURL, c.kubeconfig)
	if err != nil {
		klog.Fatalf("Error building kubeconfig: %s", err.Error())
		return err
	}

	kubeClient, clustetClient, kubeInformerFactory, informerFactory, err := k8s.GenerateInformer(config, c.labelSelector)
	if err != nil {
		klog.Fatalf("Error generate informer: %v", err)
		return err
	}

	controller := etcdinspection.NewEtcdInspectionController(
		util.NewSimpleClientBuilder(c.kubeconfig),
		kubeClient,
		clustetClient,
		informerFactory.Kstone().V1alpha1().EtcdInspections(),
	)
	// notice that there is no need to run Start methods in a separate goroutine.
	// (i.e. go kubeInformerFactory.Start(stopCh)
	// Start method is non-blocking and runs all registered informers in a dedicated goroutine.
	kubeInformerFactory.Start(stopCh)
	informerFactory.Start(stopCh)

	if err = controller.Run(2, stopCh); err != nil {
		klog.Fatalf("Error running monitor controller: %s", err.Error())
		return err
	}

	return nil
}

func (c *EtcdInspectionCommand) AddFlags(fs *pflag.FlagSet) {
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
