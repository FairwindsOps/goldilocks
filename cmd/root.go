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
	"flag"
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"k8s.io/klog/v2"
)

var kubeconfig string
var nsName string
var exitCode int

var (
	version string
	commit  string
)

func init() {
	// Flags
	rootCmd.PersistentFlags().StringVarP(&kubeconfig, "kubeconfig", "", "$HOME/.kube/config", "Kubeconfig location.")

	klog.InitFlags(nil)
	pflag.CommandLine.AddGoFlag(flag.CommandLine.Lookup("v"))

	environmentVariables := map[string]string{
		"KUBECONFIG": "kubeconfig",
	}

	for env, flag := range environmentVariables {
		flag := rootCmd.PersistentFlags().Lookup(flag)
		flag.Usage = fmt.Sprintf("%v [%v]", flag.Usage, env)
		if value := os.Getenv(env); value != "" {
			err := flag.Value.Set(value)
			if err != nil {
				klog.Errorf("Error setting flag %v to %s from environment variable %s", flag, value, env)
			}
		}
	}

}

var rootCmd = &cobra.Command{
	Use:   "goldilocks",
	Short: "goldilocks",
	Long:  `A tool for analysis of kubernetes workload resource usage.`,
	Run: func(cmd *cobra.Command, args []string) {
		klog.Error("You must specify a sub-command.")
		err := cmd.Help()
		if err != nil {
			klog.Error(err)
		}
		os.Exit(1)
	},
	PersistentPostRun: func(cmd *cobra.Command, args []string) {
		if os.Stderr != nil {
			os.Stderr.WriteString("\n\nWant more? Automate Goldilocks for free with Fairwinds Insights!\n🚀 https://fairwinds.com/insights-signup/goldilocks 🚀 \n")
		} else {
			fmt.Println("\n\nWant more? Automate Goldilocks for free with Fairwinds Insights!\n🚀 https://fairwinds.com/insights-signup/goldilocks 🚀")
		}
		os.Exit(exitCode)

	},
}

// Execute the stuff
func Execute(VERSION string, COMMIT string) {
	version = VERSION
	commit = COMMIT
	if err := rootCmd.Execute(); err != nil {
		klog.Error(err)
		os.Exit(1)
	}
}
