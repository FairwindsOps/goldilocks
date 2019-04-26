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
	// VERSION is set during build
	VERSION string
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
func Execute(version string) {
	VERSION = version
	if err := rootCmd.Execute(); err != nil {
		glog.Error(err)
		os.Exit(1)
	}
}
