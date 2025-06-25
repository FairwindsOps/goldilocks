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
	"io/fs"
	"net/http"
	"path"
	"strings"

	"github.com/gorilla/mux"
	"k8s.io/klog/v2"
)

// GetRouter returns a mux router serving all routes necessary for the dashboard
func GetRouter(setters ...Option) *mux.Router {
	opts := defaultOptions()
	for _, setter := range setters {
		setter(opts)
	}

	router := mux.NewRouter().PathPrefix(strings.TrimSuffix(opts.BasePath, "/")).Subrouter().StrictSlash(true)

	// health
	router.Handle("/health", Health("OK"))
	router.Handle("/healthz", Healthz())

	// assets
	router.Handle("/favicon.ico", Asset("assets/images/favicon-32x32.png"))

	subAssetsFS, err := fs.Sub(assetsFS, "assets")
	if err != nil {
		klog.Fatalf("error creating sub filesystem for assets: %v", err)
	}

	if subAssetsFS == nil {
		klog.Fatal("subAssetsFS is nil, this should not happen")
	}

	fileServer := http.FileServerFS(subAssetsFS)
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
