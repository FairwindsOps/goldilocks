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
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog/v2"

	"github.com/fairwindsops/goldilocks/pkg/summary"
)

var excludeContainers string
var outputFile string
var namespace string

func init() {
	rootCmd.AddCommand(summaryCmd)
	summaryCmd.PersistentFlags().StringVarP(&excludeContainers, "exclude-containers", "e", "", "Comma delimited list of containers to exclude from recommendations.")
	summaryCmd.PersistentFlags().StringVarP(&outputFile, "output-file", "f", "", "File to write output from audit.")
	summaryCmd.PersistentFlags().StringVarP(&namespace, "namespace", "n", "", "Limit the summary to only a single Namespace.")
}

var summaryCmd = &cobra.Command{
	Use:   "summary",
	Short: "Generate a summary of vpa recommendations.",
	Long: `Gather all the vpa data generate a summary of the recommendations.
By default the summary will be about all VPAs in all namespaces.`,
	Args: cobra.ArbitraryArgs,
	Run: func(cmd *cobra.Command, args []string) {
		var opts []summary.Option

		// limit to a single namespace
		if namespace != "" {
			opts = append(opts, summary.ForNamespace(namespace))
		}

		// exclude containers from the summary
		if excludeContainers != "" {
			opts = append(opts, summary.ExcludeContainers(sets.NewString(strings.Split(excludeContainers, ",")...)))
		}

		summarizer := summary.NewSummarizer(opts...)
		data, err := summarizer.GetSummary()
		if err != nil {
			klog.Fatalf("Error getting summary: %v", err)
		}

		summaryJSON, err := json.Marshal(data)
		if err != nil {
			klog.Fatalf("Error marshalling JSON: %v", err)
		}

		if outputFile != "" {
			err := os.WriteFile(outputFile, summaryJSON, 0644)
			if err != nil {
				klog.Fatalf("Failed to write summary to file: %v", err)
			}

			fmt.Println("Summary has been written to", outputFile)

		} else {
			fmt.Println(string(summaryJSON))
		}
	},
}
