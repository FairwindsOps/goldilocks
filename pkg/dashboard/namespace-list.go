package dashboard

import (
	"context"
	"net/http"
	"sort"

	"github.com/fairwindsops/goldilocks/pkg/kube"
	"github.com/fairwindsops/goldilocks/pkg/utils"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/klog/v2"
)

// NamespaceList replies with the rendered namespace list of all goldilocks enabled namespaces
func NamespaceList(opts Options) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var namespaceNames []string

		if opts.OnByDefault || opts.ShowAllVPAs {
			// When on-by-default is enabled, the controller creates VPA objects for
			// namespaces that are not explicitly labelled, so label-based discovery
			// returns nothing. Derive the namespace list from VPA objects instead,
			// which is the same source the /dashboard page uses.
			vpaLabels := opts.VpaLabels
			if opts.ShowAllVPAs {
				vpaLabels = nil
			}
			labelSelector := labels.Set(vpaLabels).String()
			vpas, err := kube.GetVPAInstance().Client.AutoscalingV1().VerticalPodAutoscalers("").List(
				context.TODO(), v1.ListOptions{LabelSelector: labelSelector},
			)
			if err != nil {
				klog.Errorf("Error listing VPAs for namespace discovery: %v", err)
				http.Error(w, "Error listing VPAs", http.StatusInternalServerError)
				return
			}
			seen := make(map[string]bool)
			for _, vpa := range vpas.Items {
				if !seen[vpa.Namespace] {
					seen[vpa.Namespace] = true
					namespaceNames = append(namespaceNames, vpa.Namespace)
				}
			}
			sort.Strings(namespaceNames)
		} else {
			listOptions := v1.ListOptions{
				LabelSelector: labels.Set(map[string]string{
					utils.VpaEnabledLabel: "true",
				}).String(),
			}
			namespacesList, err := kube.GetInstance().Client.CoreV1().Namespaces().List(context.TODO(), listOptions)
			if err != nil {
				klog.Errorf("Error getting namespace list: %v", err)
				http.Error(w, "Error getting namespace list", http.StatusInternalServerError)
				return
			}
			for _, ns := range namespacesList.Items {
				namespaceNames = append(namespaceNames, ns.Name)
			}
		}

		tmpl, err := getTemplate("namespace_list", opts,
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
			Namespaces []struct {
				Name string
			}
		}{}

		for _, name := range namespaceNames {
			item := struct {
				Name string
			}{
				Name: name,
			}
			data.Namespaces = append(data.Namespaces, item)
		}

		writeTemplate(tmpl, opts, &data, w)
	})
}
