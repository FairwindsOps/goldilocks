package dashboard

import (
	"k8s.io/client-go/tools/clientcmd/api"
	"net/http"
	"sort"

	"github.com/fairwindsops/goldilocks/pkg/kube"
	"k8s.io/klog"
)

var lastCluster string

// ClusterDetails contains all information to be passed to other functions
type ClusterDetails struct {
	Contexts         map[string]string
	CurrentContext   string
	SubmittedCluster string
	CurrentCluster   string
	ClientCfg        *api.Config
}

// ClusterList replies with the rendered cluster list of all clusters found via the kube config
func ClusterList(opts Options) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var Clusters ClusterDetails

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

		tmpl, err := getTemplate("cluster_list",
			"filter_cluster",
			"cluster_list",
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
			ClusterContext map[string]string
			CurrentCluster string
		}{ClusterContext: Clusters.Contexts, CurrentCluster: Clusters.CurrentCluster}

		writeTemplate(tmpl, opts, &data, w)
	})
}

// setLastCluster keeps track of the last submitted clusters was
// it updates the variable if it changes
// and it resets the kube client if it changes so that it can load again
func setLastCluster(currentCluster string, submittedCluster string) {
	// reset k8s client if needed
	if lastCluster == "" {
		kube.ResetInstance()
		lastCluster = currentCluster
	} else if lastCluster != submittedCluster {
		kube.ResetInstance()
		lastCluster = submittedCluster
	}
}

func makeContextClusterMap(clientCfg *api.Config) map[string]string {
	// creating map of clustername and context
	contexts := make(map[string]string)
	for v, c := range clientCfg.Contexts {
		contexts[c.Cluster] = v
	}
	return contexts
}

// getClusterAndContext sets the currentCluster and currentContext to the one,
// belonging to the submitted cluster via dashboard ui
func getClusterAndContext(Clusters *ClusterDetails) {
	if Clusters.SubmittedCluster != "" {
		Clusters.CurrentCluster = Clusters.SubmittedCluster
		Clusters.CurrentContext = Clusters.Contexts[Clusters.CurrentCluster]
	} else {
		allContexts := Clusters.ClientCfg.Contexts
		Clusters.CurrentContext = Clusters.ClientCfg.CurrentContext

		// if not currentContext is set select the first context sorted alphabetically
		if Clusters.CurrentContext == "" {
			// get alphabetically first item from contexts
			mk := make([]string, len(Clusters.ClientCfg.Contexts))
			i := 0
			for k := range Clusters.ClientCfg.Contexts {
				mk[i] = k
				i++
			}
			sort.Strings(mk)
			Clusters.CurrentContext = mk[0]
		}
		if val, ok := allContexts[Clusters.CurrentContext]; ok {
			Clusters.CurrentCluster = val.Cluster
		}
	}
}
