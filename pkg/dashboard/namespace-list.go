package dashboard

import (
	"context"
	"github.com/fairwindsops/goldilocks/pkg/kube"
	"github.com/fairwindsops/goldilocks/pkg/utils"
	"github.com/gorilla/mux"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/klog"
	"net/http"
)

// NamespaceList replies with the rendered namespace list of all goldilocks enabled namespaces
func NamespaceList(opts Options) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		var Clusters ClusterDetails

		// the clusters which was submitted via dashboard ui
		if val, ok := vars["cluster"]; ok {
			Clusters.SubmittedCluster = val
		}

		// get all kube config contexts
		useKubeConfig := true
		clientCfg, err := kube.GetClientCfg(opts.kubeconfigPath)
		if err != nil {
			klog.Warningf("Error getting k8s client config: %v, using InClusterConfig", err)
			useKubeConfig = false
		}

		if useKubeConfig && len(clientCfg.Contexts) > 0 {
			Clusters.ClientCfg = clientCfg
			Clusters.Contexts = makeContextClusterMap(clientCfg)
			getClusterAndContext(&Clusters)
		}

		setLastCluster(Clusters.CurrentCluster, Clusters.SubmittedCluster)

		namespacesList, err := kube.GetInstanceWithContext(Clusters.CurrentContext).Client.CoreV1().Namespaces().List(context.TODO(), v1.ListOptions{
			LabelSelector: labels.Set(map[string]string{
				utils.VpaEnabledLabel: "true",
			}).String(),
		})
		if err != nil {
			klog.Errorf("Error getting namespace list: %v", err)
		}

		tmpl, err := getTemplate("namespace_list",
			"filter",
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
			CurrentCluster string
		}{CurrentCluster: Clusters.CurrentCluster}

		for _, ns := range namespacesList.Items {
			data.Name = append(data.Name, ns.Name)
		}

		writeTemplate(tmpl, opts, &data, w)
	})
}
