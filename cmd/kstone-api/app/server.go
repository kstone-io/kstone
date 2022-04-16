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

package app

import (
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/klog/v2"

	"tkestack.io/kstone/cmd/kstone-api/config"
	"tkestack.io/kstone/pkg/authentication"
	"tkestack.io/kstone/pkg/middlewares"
	kstoneRouter "tkestack.io/kstone/pkg/router"
)

type APIServerCommand struct {
	token         string
	authenticator string
	namespace     string
	authCfg       string
}

// NewAPIServerCommand creates a *cobra.Command object with default parameters
func NewAPIServerCommand() *cobra.Command {
	ac := &APIServerCommand{}
	cmd := &cobra.Command{
		Use: "kstone-apiserver",
		Long: `The Kstone API server validates and configures data
for the api objects which include etcdinspections, etcdclusters, and others.
The API Server services REST operations and provides the frontend to the 
other components interact, such as kstone-controller, kstone-dashboard.`,

		// stop printing usage when the command errors
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			cmd.Flags().VisitAll(func(flag *pflag.Flag) {
				klog.V(1).Infof("FLAG: --%s=%q", flag.Name, flag.Value)
			})
			if err := ac.Run(); err != nil {
				return err
			}
			return nil
		},
	}

	fs := cmd.PersistentFlags()
	ac.AddFlags(fs)
	return cmd
}

func (c *APIServerCommand) Run() error {
	klog.Info("start kstone-api")
	config.CreateConfigFromFlags(c.token, c.authenticator)
	kstoneRouter.SetWorkNamespace(c.namespace)
	authentication.SetAuthConfigMapName(c.authCfg)

	router := kstoneRouter.NewRouter()
	router.Use(middlewares.Cors())
	err := router.Run()
	if err != nil {
		return err
	}
	return nil
}

func (c *APIServerCommand) AddFlags(fs *pflag.FlagSet) {
	fs.StringVar(
		&c.token,
		"token",
		"jwt",
		"specify the authentication token type.",
	)
	fs.StringVar(
		&c.authenticator,
		"authenticator",
		"bearertoken",
		"specify the authenticator type.",
	)
	fs.StringVar(&c.namespace,
		"namespace",
		"kstone",
		"specify the work namespace of kstone.")
	fs.StringVar(&c.authCfg,
		"auth",
		"kstone-api-user",
		"specify the auth configmap of kstone.")
}
