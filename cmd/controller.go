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
	"os"
	"os/signal"
	"syscall"

	"github.com/spf13/cobra"
	"k8s.io/klog/v2"

	"github.com/fairwindsops/goldilocks/pkg/controller"
	"github.com/fairwindsops/goldilocks/pkg/vpa"
)

var onByDefault bool
var includeNamespaces []string
var ignoreControllerKind []string
var excludeNamespaces []string
var dryRun bool

func init() {
	rootCmd.AddCommand(controllerCmd)
	controllerCmd.PersistentFlags().BoolVar(&onByDefault, "on-by-default", false, "Add goldilocks to every namespace that isn't explicitly excluded.")
	controllerCmd.PersistentFlags().BoolVar(&dryRun, "dry-run", false, "If true, don't mutate resources, just list what would have been created.")
	controllerCmd.PersistentFlags().StringSliceVar(&includeNamespaces, "include-namespaces", []string{}, "Comma delimited list of namespaces to include from recommendations.")
	controllerCmd.PersistentFlags().StringSliceVar(&excludeNamespaces, "exclude-namespaces", []string{}, "Comma delimited list of namespaces to exclude from recommendations.")
	controllerCmd.PersistentFlags().StringSliceVar(&ignoreControllerKind, "ignore-controller-kind", []string{}, "Comma delimited list of controller kinds to ignore from automatic VPA creation.")
}

var controllerCmd = &cobra.Command{
	Use:   "controller",
	Short: "Run goldilocks as a controller inside a kubernetes cluster.",
	Long:  `Run goldilocks as a controller.`,
	Run: func(cmd *cobra.Command, args []string) {
		vpaReconciler := vpa.GetInstance()
		vpaReconciler.OnByDefault = onByDefault
		vpaReconciler.IncludeNamespaces = includeNamespaces
		vpaReconciler.ExcludeNamespaces = excludeNamespaces

		klog.V(4).Infof("Starting controller with Reconciler: %+v", vpaReconciler)

		// create a channel for sending a stop to kube watcher threads
		stop := make(chan bool, 1)
		defer close(stop)
		go controller.NewController(stop)

		// create a channel to respond to signals
		signals := make(chan os.Signal, 1)
		defer close(signals)

		signal.Notify(signals, syscall.SIGTERM)
		signal.Notify(signals, syscall.SIGINT)
		s := <-signals
		stop <- true
		klog.Infof("Exiting, got signal: %v", s)
	},
}
