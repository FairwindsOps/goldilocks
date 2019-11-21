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
	"fmt"
	"net/http"

	packr "github.com/gobuffalo/packr/v2"
	"github.com/gorilla/mux"

	"github.com/fairwindsops/goldilocks/pkg/dashboard/handlers"
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

// GetRouter returns a mux router serving all routes necessary for the dashboard
func GetRouter(port int, basePath string, vpaLabels map[string]string, excludeContainers string) *mux.Router {
	router := mux.NewRouter()

	// health
	router.Handle("/health", handlers.Health("OK"))
	router.Handle("/healthz", handlers.Healthz())

	// assets
	router.Handle("/favicon.ico", handlers.Asset("images/favicon-32x32.png"))
	router.PathPrefix("/static/").Handler(handlers.StaticAssets("/static/"))

	// dashboard
	router.Handle("/dashboard", handlers.DashboardAllNamespaces(basePath, vpaLabels, excludeContainers))

	// root
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		// catch all other paths that weren't matched
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		// default redirect on root path
		http.Redirect(w, r, fmt.Sprintf("%s://%s/dashboard", r.URL.Scheme, r.URL.Host), http.StatusMovedPermanently)
	})

	return router
}
