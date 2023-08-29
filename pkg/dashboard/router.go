// Copyright 2019 FairwindsOps Inc
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package dashboard

import (
	"net/http"
	"path"

	"k8s.io/klog/v2"

	packr "github.com/gobuffalo/packr/v2"
	"github.com/gorilla/mux"
)

var (
	markdownBox = (*packr.Box)(nil)
)

// GetMarkdownBox returns a binary-friendly set of markdown files with error details
func GetMarkdownBox() *packr.Box {
	if markdownBox == (*packr.Box)(nil) {
		markdownBox = packr.New("Markdown", "../../docs")
	}
	return markdownBox
}

func GetAssetBox() *packr.Box {
	if assetBox == (*packr.Box)(nil) {
		assetBox = packr.New("Assets", "assets")
	}
	return assetBox
}

// GetRouter returns a mux router serving all routes necessary for the dashboard
func GetRouter(setters ...Option) *mux.Router {
	opts := defaultOptions()
	for _, setter := range setters {
		setter(opts)
	}

	router := mux.NewRouter().PathPrefix(opts.BasePath).Subrouter()

	// health
	router.Handle("/health", Health("OK"))
	router.Handle("/healthz", Healthz())

	// assets
	router.Handle("/favicon.ico", Asset("/images/favicon-32x32.png"))
	fileServer := http.FileServer(GetAssetBox())
	router.PathPrefix("/static/").Handler(http.StripPrefix(path.Join(opts.BasePath, "/static/"), fileServer))

	// dashboard
	router.Handle("/dashboard", Dashboard(*opts))
	router.Handle("/dashboard/{namespace:[a-zA-Z0-9-]+}", Dashboard(*opts))

	// namespace list
	router.Handle("/namespaces", NamespaceList(*opts))

	// root
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// catch all other paths that weren't matched
		if r.URL.Path != "/" && r.URL.Path != opts.BasePath && r.URL.Path != opts.BasePath+"/" {
			klog.Infof("404: %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}

		klog.Infof("redirecting to %v", path.Join(opts.BasePath, "/namespaces"))
		// default redirect on root path
		http.Redirect(w, r, path.Join(opts.BasePath, "/namespaces"), http.StatusMovedPermanently)
	})

	// api
	router.Handle("/api/{namespace:[a-zA-Z0-9-]+}", API(*opts))
	return router
}
