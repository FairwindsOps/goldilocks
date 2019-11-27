package web

import (
	"net/http"

	"code.squarespace.net/sre/goldilocks/pkg/kube"
	"code.squarespace.net/sre/goldilocks/pkg/utils"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/klog"
)

// NamespaceList replies with the rendered namespace list of all goldilocks enabled namespaces
func NamespceList(opts Options) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		namespacesList, err := kube.GetInstance().Client.CoreV1().Namespaces().List(v1.ListOptions{
			LabelSelector: labels.Set(map[string]string{
				utils.GoldilocksNameFor(utils.NamespaceVPAEnabledLabel): "true",
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
