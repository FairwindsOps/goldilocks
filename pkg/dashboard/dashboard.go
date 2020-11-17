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
    "github.com/gorilla/mux"
    "k8s.io/klog"
    "net/http"

    "github.com/fairwindsops/goldilocks/pkg/summary"
)

// Dashboard replies with the rendered dashboard (on the basePath) for the summarizer
func Dashboard(opts Options) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
        var Clusters ClusterDetails

		var namespace string
		if val, ok := vars["namespace"]; ok {
			namespace = val
		}

		// the clusters which was submitted via dashboard ui
		if val, ok := vars["cluster"]; ok {
		    Clusters.SubmittedCluster = val
		}

        // get all kube config contexts
		useKubeConfig := true
		clientCfg, err := kube.GetClientCfg(opts.kubeconfigPath)
		if err != nil {
			klog.Warning("Error getting k8s client config: %v, using inClusterConfig", err)
			useKubeConfig = false
		}

		if useKubeConfig && len(clientCfg.Contexts) > 0 {
            Clusters.ClientCfg = clientCfg
            Clusters.Contexts = makeContextClusterMap(clientCfg)
            getClusterAndContext(&Clusters)
        }

        setLastCluster(Clusters.CurrentCluster, Clusters.SubmittedCluster)

		// TODO [hkatz] add caching or refresh button support
		summarizer := summary.NewSummarizer(
			Clusters.CurrentContext,
			summary.ForNamespace(namespace),
			summary.ForVPAsWithLabels(opts.vpaLabels),
			summary.ExcludeContainers(opts.excludedContainers),
		)

		vpaData, err := summarizer.GetSummary()
		if err != nil {
			klog.Errorf("Error getting vpaData: %v", err)
		}

        klog.Infof("Contexts: %v, currentCluster %s, currentContext", Clusters.Contexts, Clusters.CurrentCluster, Clusters.CurrentContext)

		data := struct {
			Summary        summary.Summary
			ClusterContext map[string]string
			CurrentCluster string
		}{Summary: vpaData, ClusterContext: Clusters.Contexts, CurrentCluster: Clusters.CurrentCluster}

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
