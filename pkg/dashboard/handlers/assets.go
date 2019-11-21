package handlers

import (
	"net/http"

	"github.com/gobuffalo/packr/v2"
	"k8s.io/klog"
)

var assetBox = (*packr.Box)(nil)

// getAssetBox returns a binary-friendly set of assets packaged from disk
func getAssetBox() *packr.Box {
	if assetBox == (*packr.Box)(nil) {
		assetBox = packr.New("Assets", "assets")
	}
	return assetBox
}

// Asset replies with the contents of the loaded asset from disk
func Asset(assetPath string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		asset, err := getAssetBox().Find(assetPath)
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

// StaticAssets replies with a FileServer for all assets, the prefix is used to strip the URL path
func StaticAssets(prefix string) http.Handler {
	return http.StripPrefix(prefix, http.FileServer(getAssetBox()))
}
