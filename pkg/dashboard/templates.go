package dashboard

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"github.com/fairwindsops/goldilocks/pkg/dashboard/helpers"
	"k8s.io/klog/v2"
)

// templates
const (
	ContainerTemplateName   = "container.gohtml"
	DashboardTemplateName   = "dashboard.gohtml"
	FilterTemplateName      = "filter.gohtml"
	FooterTemplateName      = "footer.gohtml"
	HeadTemplateName        = "head.gohtml"
	NamespaceTemplateName   = "namespace.gohtml"
	NavigationTemplateName  = "navigation.gohtml"
	EmailTemplateName       = "email.gohtml"
	ApiTokenTemplateName    = "api_token.gohtml"
	CostSettingTemplateName = "cost_settings.gohtml"
)

var (
	// templates with these names are included by default in getTemplate()
	defaultIncludedTemplates = []string{
		"head",
		"navigation",
		"footer",
	}
)

// to be included in data structs fo
type baseTemplateData struct {
	// BasePath is the base URL that goldilocks is being served on, used in templates for html base
	BasePath string

	// Data is the data struct passed to writeTemplate()
	Data interface{}

	// JSON is the json version of Data
	JSON template.JS
}

// getTemplate puts together a template. Individual pieces can be overridden before rendering.
func getTemplate(name string, opts Options, includedTemplates ...string) (*template.Template, error) {
	tmpl := template.New(name).Funcs(template.FuncMap{
		"printResource":  helpers.PrintResource,
		"getStatus":      helpers.GetStatus,
		"getStatusRange": helpers.GetStatusRange,
		"resourceName":   helpers.ResourceName,
		"getUUID":        helpers.GetUUID,
		"hasField":       helpers.HasField,

		"opts": func() Options {
			return opts
		},
	})

	// join the default templates and included templates
	templatesToParse := make([]string, 0, len(includedTemplates)+len(defaultIncludedTemplates))
	templatesToParse = append(templatesToParse, defaultIncludedTemplates...)
	templatesToParse = append(templatesToParse, includedTemplates...)

	return parseTemplateFiles(tmpl, templatesToParse)
}

// parseTemplateFiles combines the template with the included templates into one parsed template
func parseTemplateFiles(tmpl *template.Template, includedTemplates []string) (*template.Template, error) {
	for _, fname := range includedTemplates {
		templateFile, err := templatesFS.ReadFile("templates/" + fmt.Sprintf("%s.gohtml", fname))
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

// writeTemplate executes the given template with the data and writes to the writer.
func writeTemplate(tmpl *template.Template, opts Options, data interface{}, w http.ResponseWriter) {
	buf := &bytes.Buffer{}
	jsonData, err := json.Marshal(data)
	if err != nil {
		http.Error(w, "Error serializing template jsonData", http.StatusInternalServerError)
		return
	}
	err = tmpl.Execute(buf, baseTemplateData{
		BasePath: validateBasePath(opts.BasePath),
		Data:     data,
		JSON:     template.JS(jsonData),
	})
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

func validateBasePath(path string) string {
	if path == "/" {
		return path
	}

	if !strings.HasSuffix(path, "/") {
		path = path + "/"
	}

	return path
}
