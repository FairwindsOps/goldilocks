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

package dashboard

import (
	"github.com/fairwindsops/goldilocks/pkg/kube"
	"net/http"

	"github.com/gorilla/mux"
	"k8s.io/klog"

	"github.com/fairwindsops/goldilocks/pkg/summary"
)

// Dashboard replies with the rendered dashboard (on the basePath) for the summarizer
func Dashboard(opts Options) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		var namespace string
		if val, ok := vars["namespace"]; ok {
			namespace = val
		}

		var submittedCluster string
		if val, ok := vars["cluster"]; ok {
			submittedCluster = val
		}

		// get all kube config contexts
		var currentCluster string
		var currentContext string
		contexts := make(map[string]string)
		clientCfg, err := kube.GetClientCfg(opts.kubeconfigPath)
		if err != nil {
			klog.Warning("Error getting k8s client config: %v, using inClusterConfig", err)
		} else {
			// adding the clustername and context name to the map
			for v, c := range clientCfg.Contexts {
				contexts[c.Cluster] = v
			}

			if submittedCluster != "" {
				currentCluster = submittedCluster
				currentContext = contexts[currentCluster]
			} else {
				allContexts := clientCfg.Contexts
				currentContext := clientCfg.CurrentContext
				if val, ok := allContexts[currentContext]; ok {
					currentCluster = val.Cluster
				} else {
					currentCluster = ""
				}
			}
		}

		if usedCluster == "" {
			kube.ResetInstance()
			usedCluster = currentCluster
		} else if usedCluster != submittedCluster {
			kube.ResetInstance()
			usedCluster = submittedCluster
		}

		// TODO [hkatz] add caching or refresh button support
		summarizer := summary.NewSummarizer(
			currentContext,
			summary.ForNamespace(namespace),
			summary.ForVPAsWithLabels(opts.vpaLabels),
			summary.ExcludeContainers(opts.excludedContainers),
		)

		vpaData, err := summarizer.GetSummary()
		if err != nil {
			klog.Errorf("Error getting vpaData: %v", err)
		}

		data := struct {
			Summary        summary.Summary
			ClusterContext map[string]string
			CurrentCluster string
		}{Summary: vpaData, ClusterContext: contexts, CurrentCluster: currentCluster}

		tmpl, err := getTemplate("dashboard",
			"container",
			"namespace",
			"dashboard",
		)
		if err != nil {
			klog.Errorf("Error getting template data %v", err)
			http.Error(w, "Error getting template data", http.StatusInternalServerError)
			return
		}

		writeTemplate(tmpl, opts, &data, w)
	})
}
