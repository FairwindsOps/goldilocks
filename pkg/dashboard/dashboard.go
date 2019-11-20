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
	"bytes"
	"encoding/json"
	"html/template"
	"net/http"
	"strings"

	packr "github.com/gobuffalo/packr/v2"
	"github.com/gorilla/mux"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/klog"

	"github.com/fairwindsops/goldilocks/pkg/summary"
)

const (
	// MainTemplateName is the main template
	MainTemplateName = "main.gohtml"
	// HeadTemplateName contains styles and meta info
	HeadTemplateName = "head.gohtml"
	// NavbarTemplateName contains the navbar
	NavbarTemplateName = "navbar.gohtml"
	// PreambleTemplateName contains an empty preamble that can be overridden
	PreambleTemplateName = "preamble.gohtml"
	// DashboardTemplateName contains the content of the dashboard
	DashboardTemplateName = "dashboard.gohtml"
	// FooterTemplateName contains the footer
	FooterTemplateName = "footer.gohtml"
	// CheckDetailsTemplateName is a page for rendering details about a given check
	CheckDetailsTemplateName = "check-details.gohtml"
)

var (
	templateBox = (*packr.Box)(nil)
	assetBox    = (*packr.Box)(nil)
	markdownBox = (*packr.Box)(nil)
)

// GetAssetBox returns a binary-friendly set of assets packaged from disk
func GetAssetBox() *packr.Box {
	if assetBox == (*packr.Box)(nil) {
		assetBox = packr.New("Assets", "assets")
	}
	return assetBox
}

// GetTemplateBox returns a binary-friendly set of templates for rendering the dash
func GetTemplateBox() *packr.Box {
	if templateBox == (*packr.Box)(nil) {
		templateBox = packr.New("Templates", "templates")
	}
	return templateBox
}

// GetMarkdownBox returns a binary-friendly set of markdown files with error details
func GetMarkdownBox() *packr.Box {
	if markdownBox == (*packr.Box)(nil) {
		markdownBox = packr.New("Markdown", "../../docs")
	}
	return markdownBox
}

// templateData is passed to the dashboard HTML template
type templateData struct {
	BasePath string
	VPAData  summary.Summary
	JSON     template.JS
}

// GetBaseTemplate puts together the dashboard template. Individual pieces can be overridden before rendering.
func GetBaseTemplate(name string) (*template.Template, error) {
	tmpl := template.New(name).Funcs(template.FuncMap{
		"printResource":  printResource,
		"getStatus":      getStatus,
		"getStatusRange": getStatusRange,
		"resourceName":   resourceName,
		"getUUID":        getUUID,
	})

	templateFileNames := []string{
		DashboardTemplateName,
		HeadTemplateName,
		NavbarTemplateName,
		PreambleTemplateName,
		FooterTemplateName,
		MainTemplateName,
	}
	return parseTemplateFiles(tmpl, templateFileNames)
}

func parseTemplateFiles(tmpl *template.Template, templateFileNames []string) (*template.Template, error) {
	templateBox := GetTemplateBox()
	for _, fname := range templateFileNames {
		templateFile, err := templateBox.Find(fname)
		if err != nil {
			return nil, err
		}

		tmpl, err = tmpl.Parse(string(templateFile))
		if err != nil {
			return nil, err
		}
	}
	return tmpl, nil
}

func writeTemplate(tmpl *template.Template, data *templateData, w http.ResponseWriter) {
	buf := &bytes.Buffer{}
	err := tmpl.Execute(buf, data)
	if err != nil {
		klog.Errorf("Error executing template: %v", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	_, err = buf.WriteTo(w)
	if err != nil {
		klog.Errorf("Error writing template: %v", err)
	}
}

// GetRouter returns a mux router serving all routes necessary for the dashboard
func GetRouter(port int, basePath string, vpaLabels map[string]string, excludeContainers string) *mux.Router {
	router := mux.NewRouter()
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte("OK"))
		if err != nil {
			klog.Errorf("Error writing healthcheck: %v", err)
		}
	})
	router.HandleFunc("/favicon.ico", func(w http.ResponseWriter, r *http.Request) {
		favicon, err := GetAssetBox().Find("images/favicon-32x32.png")
		if err != nil {
			klog.Errorf("Error getting favicon: %v", err)
			http.Error(w, "Error getting favicon", http.StatusInternalServerError)
			return
		}
		_, err = w.Write(favicon)
		if err != nil {
			klog.Errorf("Error writing favicon: %v", err)
		}
	})

	fileServer := http.FileServer(GetAssetBox())
	router.PathPrefix("/static/").Handler(http.StripPrefix("/static/", fileServer))
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		summarizer := summary.NewSummarizer(
			summary.ForVpasWithLabels(vpaLabels),
			summary.ExcludeContainers(sets.NewString(strings.Split(excludeContainers, ",")...)),
		)

		data, err := summarizer.GetSummary()
		if err != nil {
			klog.Errorf("Error getting data: %v", err)
			http.Error(w, "Error running summary.", 500)
			return
		}
		MainHandler(w, r, data, basePath)
	})
	return router
}

// MainHandler gets template data and renders the dashboard with it.
func MainHandler(w http.ResponseWriter, r *http.Request, vpaData summary.Summary, basePath string) {
	jsonData, err := json.Marshal(vpaData)
	if err != nil {
		http.Error(w, "Error serializing summary data", 500)
		return
	}

	data := templateData{
		BasePath: basePath,
		VPAData:  vpaData,
		JSON:     template.JS(jsonData),
	}
	tmpl, err := GetBaseTemplate("main")
	if err != nil {
		klog.Errorf("Error getting template data %v", err)
		http.Error(w, "Error getting template data", 500)
		return
	}
	writeTemplate(tmpl, &data, w)
}
