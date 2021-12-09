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
	"k8s.io/klog/v2"

	"tkestack.io/kstone/pkg/middlewares"
	kstoneRouter "tkestack.io/kstone/pkg/router"
)

// NewAPIServerCommand creates a *cobra.Command object with default parameters
func NewAPIServerCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "kstone-apiserver",
		Long: `The Kstone API server validates and configures data
for the api objects which include etcdinspections, etcdclusters, and others.
The API Server services REST operations and provides the frontend to the 
other components interact, such as kstone-controller, kstone-dashboard.`,

		// stop printing usage when the command errors
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := Run(); err != nil {
				return err
			}
			return nil
		},
	}

	return cmd
}

func Run() error {
	klog.Info("start kstone-api")
	router := kstoneRouter.NewRouter()
	router.Use(middlewares.Cors())
	err := router.Run()
	if err != nil {
		return err
	}
	return nil
}
