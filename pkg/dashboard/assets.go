package dashboard

import (
	"embed"
	"net/http"

	"k8s.io/klog/v2"
)

//go:embed all:templates
var templatesFS embed.FS

// Asset replies with the contents of the loaded asset from disk
func Asset(assetPath string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		asset, err := templatesFS.ReadFile("templates/" + assetPath)
		if err != nil {
			klog.Errorf("Error getting asset: %v", err)
			http.Error(w, "Error getting asset", http.StatusInternalServerError)
			return
		}
		_, err = w.Write(asset)
		if err != nil {
			klog.Errorf("Error writing asset: %v", err)
		}
	})
}
