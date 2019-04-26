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
	"flag"
	"fmt"
	"os"

	"github.com/golang/glog"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var kubeconfig string
var namespace string
var vpaLabels map[string]string

var (
	VERSION string
	COMMIT  string
)

func init() {
	// Init
	pflag.CommandLine.AddGoFlagSet(flag.CommandLine)
	flag.Set("logtostderr", "true")
	flag.Parse()

	// Flags
	rootCmd.PersistentFlags().StringVarP(&kubeconfig, "kubeconfig", "", "$HOME/.kube/config", "Kubeconfig location.")
	rootCmd.PersistentFlags().StringVarP(&namespace, "namespace", "n", "default", "Namespace to install the VPA objects in.")
	rootCmd.MarkFlagRequired("namespace")

	environmentVariables := map[string]string{
		"KUBECONFIG": "kubeconfig",
	}

	for env, flag := range environmentVariables {
		flag := rootCmd.PersistentFlags().Lookup(flag)
		flag.Usage = fmt.Sprintf("%v [%v]", flag.Usage, env)
		if value := os.Getenv(env); value != "" {
			flag.Value.Set(value)
		}
	}

	vpaLabels = map[string]string{
		"owner":  "ReactiveOps",
		"source": "vpa-analysis",
	}
}

var rootCmd = &cobra.Command{
	Use:   "vpa-analysis",
	Short: "vpa-analysis",
	Long:  `A tool for analysis of kubernetes deployment resource usage.`,
	Run: func(cmd *cobra.Command, args []string) {
		glog.Error("You must specify a sub-command.")
		cmd.Help()
		os.Exit(1)
	},
}

// Execute the stuff
func Execute(version string, commit string) {
	VERSION = version
	COMMIT = commit
	if err := rootCmd.Execute(); err != nil {
		glog.Error(err)
		os.Exit(1)
	}
}
