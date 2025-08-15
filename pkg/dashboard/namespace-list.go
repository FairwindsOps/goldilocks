package dashboard

import (
	"fmt"
	"net/http"
	"time"

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
		if opts.OnByDefault || opts.ShowAllVPAs {
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
		ctx, cancel := utils.CreateContextWithTimeout(r.Context(), 30*time.Second)
		defer cancel()
		namespacesList, err := kube.GetInstance().Client.CoreV1().Namespaces().List(ctx, listOptions)
		if err != nil {
			klog.Errorf("Error getting namespace list: %v", err)
			http.Error(w, "Error getting namespace list", http.StatusInternalServerError)
			return
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

		for _, ns := range namespacesList.Items {
			item := struct {
				Name string
			}{
				Name: ns.Name,
			}
			data.Namespaces = append(data.Namespaces, item)
		}

		writeTemplate(tmpl, opts, &data, w)
	})
}
