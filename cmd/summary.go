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
