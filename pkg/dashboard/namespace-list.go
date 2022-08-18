package dashboard

import (
	"context"
	"fmt"
	"net/http"

	"github.com/fairwindsops/goldilocks/pkg/kube"
	"github.com/fairwindsops/goldilocks/pkg/utils"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/klog/v2"
)

// NamespaceList replies with the rendered namespace list of all goldilocks enabled namespaces
func NamespaceList(opts Options) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var listOptions v1.ListOptions
		if opts.onByDefault || opts.showAllVPAs {
			listOptions = v1.ListOptions{
				LabelSelector: fmt.Sprintf("%s!=false", utils.VpaEnabledLabel),
			}
		} else {
			listOptions = v1.ListOptions{
				LabelSelector: labels.Set(map[string]string{
					utils.VpaEnabledLabel: "true",
				}).String(),
			}
		}
		namespacesList, err := kube.GetInstance().Client.CoreV1().Namespaces().List(context.TODO(), listOptions)
		if err != nil {
			klog.Errorf("Error getting namespace list: %v", err)
			http.Error(w, "Error getting namespace list", http.StatusInternalServerError)
			return
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
