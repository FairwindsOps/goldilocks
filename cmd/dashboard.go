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
	"net/http"
	"strings"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog/v2"

	"github.com/fairwindsops/goldilocks/pkg/dashboard"
)

var (
	serverPort  int
	showAllVPAs bool
	basePath    string
)

func init() {
	rootCmd.AddCommand(dashboardCmd)
	dashboardCmd.PersistentFlags().IntVarP(&serverPort, "port", "p", 8080, "The port to serve the dashboard on.")
	dashboardCmd.PersistentFlags().StringVarP(&excludeContainers, "exclude-containers", "e", "", "Comma delimited list of containers to exclude from recommendations.")
	dashboardCmd.PersistentFlags().BoolVar(&onByDefault, "on-by-default", false, "Display every namespace that isn't explicitly excluded.")
	dashboardCmd.PersistentFlags().BoolVar(&showAllVPAs, "show-all", false, "Display every VPA, even if it isn't managed by Goldilocks")
	dashboardCmd.PersistentFlags().StringVar(&basePath, "base-path", "/", "Path on which the dashboard is served.")
}

var dashboardCmd = &cobra.Command{
	Use:   "dashboard",
	Short: "Run the goldilocks dashboard that will show recommendations.",
	Long:  `Run the goldilocks dashboard that will show recommendations.`,
	Run: func(cmd *cobra.Command, args []string) {
		var validBasePath = validateBasePath(basePath)
		router := dashboard.GetRouter(
			dashboard.OnPort(serverPort),
			dashboard.BasePath(validBasePath),
			dashboard.ExcludeContainers(sets.NewString(strings.Split(excludeContainers, ",")...)),
			dashboard.OnByDefault(onByDefault),
			dashboard.ShowAllVPAs(showAllVPAs),
		)
		http.Handle("/", router)
		klog.Infof("Starting goldilocks dashboard server on port %d and basePath %v", serverPort, validBasePath)
		klog.Fatalf("%v", http.ListenAndServe(fmt.Sprintf(":%d", serverPort), nil))
	},
}

func validateBasePath(path string) string {
	if path == "" || path == "/" {
		return "/"
	}

	if !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	return strings.TrimSuffix(path, "/")
}
