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
	createCmd.PersistentFlags().BoolVarP(&runonce, "run-once", "", false, "Only run once and do not loop.")
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
