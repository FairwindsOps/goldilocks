// Copyright 2019 FairwindsOps Inc
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
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"k8s.io/klog"

	"github.com/fairwindsops/goldilocks/pkg/kube"
	"github.com/fairwindsops/goldilocks/pkg/vpa"
)

var dryrun bool

func init() {
	rootCmd.AddCommand(createCmd)
	createCmd.PersistentFlags().BoolVarP(&dryrun, "dry-run", "", false, "Don't actually create the VPAs, just list which ones would get created.")
	createCmd.PersistentFlags().StringVarP(&nsName, "namespace", "n", "default", "Namespace to install the VPA objects in.")
}

var createCmd = &cobra.Command{
	Use:   "create-vpas",
	Short: "Create VPAs",
	Long:  `Create a VPA for every deployment in the specified namespace.`,
	Run: func(cmd *cobra.Command, args []string) {
		klog.V(4).Infof("Starting to create the VPA objects in namespace: %s", nsName)
		kubeClient := kube.GetInstance()
		namespace, err := kube.GetNamespace(kubeClient, nsName)
		if err != nil {
			fmt.Println("Error getting namespace. Exiting.")
			os.Exit(1)
		}
		errReconcile := vpa.GetInstance().ReconcileNamespace(namespace, dryrun)
		if errReconcile != nil {
			fmt.Println("Errors encountered during reconciliation.")
			os.Exit(1)
		}
	},
}
