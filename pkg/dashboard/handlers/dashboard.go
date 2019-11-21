package handlers

import (
	"bytes"
	"encoding/json"
	"html/template"
	"net/http"

	"code.squarespace.net/sre/goldilocks/pkg/dashboard/helpers"
	"code.squarespace.net/sre/goldilocks/pkg/summary"
	"github.com/gobuffalo/packr/v2"
	"k8s.io/klog"
)

// templates
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
	// NamespaceTemplateName contains the content for a namespace
	NamespaceTemplateName = "namespace.gohtml"
	// ContainerTemplateName contains the content for a container
	ContainerTemplateName = "container.gohtml"
	// FooterTemplateName contains the footer
	FooterTemplateName = "footer.gohtml"
	// CheckDetailsTemplateName is a page for rendering details about a given check
	CheckDetailsTemplateName = "check-details.gohtml"
)

var templateBox = (*packr.Box)(nil)

// templateData is passed to the dashboard HTML template
type templateData struct {
	// BasePath is the base URL that goldilocks is being served on, used in templates for html base
	BasePath string

	// VPAData is the summary information to display on the rendered dashboard
	VPAData summary.Summary

	// JSON is the json version of the summary data provided to the html window object for debugging and user interaction
	JSON template.JS
}

// getTemplateBox returns a binary-friendly set of templates for rendering the dash
func getTemplateBox() *packr.Box {
	if templateBox == (*packr.Box)(nil) {
		templateBox = packr.New("Templates", "templates")
	}
	return templateBox
}

// DashboardAllNamespaces replies with the rendered dashboard (on the basePath) for the summarizer
func Dashboard(basePath string, summarizer *summary.Summarizer) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// TODO [hkatz] add caching or refresh button support
		err := summarizer.UpdateVPAs()
		if err != nil {
			klog.Errorf("Error updating vpas: %v", err)
			// don't give up, just use the cached vpas on the summarizer
		}

		vpaData, err := summarizer.GetSummary()
		if err != nil {
			klog.Errorf("Error getting vpaData: %v", err)
			http.Error(w, "Error running summary.", http.StatusInternalServerError)
			return
		}
		jsonData, err := json.Marshal(vpaData)
		if err != nil {
			http.Error(w, "Error serializing summary vpaData", http.StatusInternalServerError)
			return
		}

		data := templateData{
			BasePath: basePath,
			VPAData:  vpaData,
			JSON:     template.JS(jsonData),
		}
		tmpl, err := getTemplate("main")
		if err != nil {
			klog.Errorf("Error getting template data %v", err)
			http.Error(w, "Error getting template data", http.StatusInternalServerError)
			return
		}

		writeTemplate(tmpl, &data, w)
	})
}

// getTemplate puts together the dashboard template. Individual pieces can be overridden before rendering.
func getTemplate(name string) (*template.Template, error) {
	tmpl := template.New(name).Funcs(template.FuncMap{
		"printResource":  helpers.PrintResource,
		"getStatus":      helpers.GetStatus,
		"getStatusRange": helpers.GetStatusRange,
		"resourceName":   helpers.ResourceName,
		"getUUID":        helpers.GetUUID,
	})

	templateFileNames := []string{
		DashboardTemplateName,
		NamespaceTemplateName,
		ContainerTemplateName,
		HeadTemplateName,
		NavbarTemplateName,
		PreambleTemplateName,
		FooterTemplateName,
		MainTemplateName,
	}

	return parseTemplateFiles(tmpl, templateFileNames)
}

func parseTemplateFiles(tmpl *template.Template, templateFileNames []string) (*template.Template, error) {
	templateBox := getTemplateBox()
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
