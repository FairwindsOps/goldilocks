// Copyright 2019 ReactiveOps
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"github.com/golang/glog"
	"github.com/spf13/cobra"

	"github.com/reactiveops/vpa-analysis/pkg/vpa"
)

var runonce bool
var dryrun bool

func init() {
	rootCmd.AddCommand(createCmd)
	createCmd.PersistentFlags().BoolVarP(&runonce, "run-once", "", true, "Only run once and do not loop.")
	createCmd.PersistentFlags().BoolVarP(&dryrun, "dry-run", "", false, "Don't actually create the VPAs, just list which ones would get created.")
}

var createCmd = &cobra.Command{
	Use:   "create-vpas",
	Short: "Create VPAs",
	Long:  `Create a VPA for every deployment in the specified namespace.`,
	Run: func(cmd *cobra.Command, args []string) {
		glog.V(4).Infof("Starting to create the VPA objects in namespace: %s", namespace)
		vpa.Create(namespace, &kubeconfig, vpaLabels, runonce, dryrun)
	},
}
