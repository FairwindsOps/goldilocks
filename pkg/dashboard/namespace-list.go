package dashboard

import (
	"context"
	"github.com/gorilla/mux"
	"net/http"

	"github.com/fairwindsops/goldilocks/pkg/kube"
	"github.com/fairwindsops/goldilocks/pkg/utils"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/klog"
)

var usedCluster string

// NamespaceList replies with the rendered namespace list of all goldilocks enabled namespaces
func NamespaceList(opts Options) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)

		var submittedCluster string
		if val, ok := vars["cluster"]; ok {
			submittedCluster = val
		}

		// get all kube config contexts
		clientCfg, err := kube.GetClientCfg(opts.kubeconfigPath)
		if err != nil {
			klog.Errorf("Error getting k8s client config: %v", err)
			http.Error(w, "Error getting k8s client config.", http.StatusInternalServerError)
			return
		}

		// adding the clustername and context name to the map
		contexts := make(map[string]string)
		for v, c := range clientCfg.Contexts {
			contexts[c.Cluster] = v
		}

		var currentCluster string
		var currentContext string
		if submittedCluster != "" {
			currentCluster = submittedCluster
			currentContext = contexts[currentCluster]
		} else {
			allContexts := clientCfg.Contexts
			currentContext := clientCfg.CurrentContext
			if val, ok := allContexts[currentContext]; ok {
				currentCluster = val.Cluster
			} else {
				currentContext = ""
			}
		}

		if usedCluster == "" {
			kube.ResetInstance()
			usedCluster = currentCluster
		} else if usedCluster != submittedCluster {
			kube.ResetInstance()
			usedCluster = submittedCluster
		}

		namespacesList, err := kube.GetInstanceWithContext(currentContext).Client.CoreV1().Namespaces().List(context.TODO(), v1.ListOptions{
			LabelSelector: labels.Set(map[string]string{
				utils.VpaEnabledLabel: "true",
			}).String(),
		})
		if err != nil {
			klog.Errorf("Error getting namespace list: %v", err)
			http.Error(w, "Error getting namespace list", http.StatusInternalServerError)
		}

		tmpl, err := getTemplate("namespace_list",
			"namespace_list",
		)
		if err != nil {
			klog.Errorf("Error getting template data: %v", err)
			http.Error(w, "Error getting template data", http.StatusInternalServerError)
			return
		}

		// only expose the needed data from Namespace
		// this helps to not leak additional information like
		// annotations, labels, metadata about the Namespace to the
		// client UI source code or javascript console
		data := struct {
			Name           []string
			ClusterContext map[string]string
			CurrentCluster string
		}{ClusterContext: contexts, CurrentCluster: currentCluster}

		for _, ns := range namespacesList.Items {
			data.Name = append(data.Name, ns.Name)
		}

		writeTemplate(tmpl, opts, &data, w)
	})
}
