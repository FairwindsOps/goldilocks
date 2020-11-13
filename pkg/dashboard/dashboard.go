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
	"k8s.io/client-go/tools/clientcmd"
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

		// TODO [hkatz] add caching or refresh button support
		summarizer := summary.NewSummarizer(
			summary.ForNamespace(namespace),
			summary.ForVPAsWithLabels(opts.vpaLabels),
			summary.ExcludeContainers(opts.excludedContainers),
		)

		vpaData, err := summarizer.GetSummary()
		if err != nil {
			klog.Errorf("Error getting vpaData: %v", err)
			http.Error(w, "Error running summary.", http.StatusInternalServerError)
			return
		}

        // get all kube config contexts
		contexts := kube.GetContexts(opts.kubeconfigPath)

		clientCfg, err := clientcmd.NewDefaultClientConfigLoadingRules().Load()
		if err != nil {
			klog.Errorf("Error getting current k8s context: %v", err)
			http.Error(w, "Error running summary.", http.StatusInternalServerError)
			return
		}
		klog.Infof("Current %s", clientCfg.CurrentContext)

		data := struct {
			Summary        summary.Summary
			ClusterContext map[string]string
			DefaultCluster string
		}{Summary: vpaData, ClusterContext: contexts, DefaultCluster: clientCfg.CurrentContext}

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
