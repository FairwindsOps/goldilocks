package dashboard

import (
	"context"
	"net/http"

	"github.com/fairwindsops/goldilocks/pkg/kube"
	"github.com/fairwindsops/goldilocks/pkg/utils"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/klog"
)

// NamespaceList replies with the rendered namespace list of all goldilocks enabled namespaces
func NamespaceList(opts Options) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		namespacesList, err := kube.GetInstanceWithContext("tooling-west-1-admin").Client.CoreV1().Namespaces().List(context.TODO(), v1.ListOptions{
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
		data := []struct {
			Name string
		}{}

		for _, ns := range namespacesList.Items {
			item := struct {
				Name string
			}{
				Name: ns.Name,
			}
			data = append(data, item)
		}

		writeTemplate(tmpl, opts, &data, w)
	})
}
