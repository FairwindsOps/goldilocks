package cmd

import (
	"github.com/spf13/cobra"

	"github.com/reactiveops/vpa-analysis/pkg/summary"
)

func init() {
	rootCmd.AddCommand(summaryCmd)
}

var summaryCmd = &cobra.Command{
	Use:   "summary",
	Short: "Genarate a summary of the vpa recommendations in a namespace.",
	Long:  `Gather all the vpa data in a namespace and generaate a summary of the recommendations.`,
	Run: func(cmd *cobra.Command, args []string) {
		summary.Run(namespace, &kubeconfig, vpaLabels)
	},
}
