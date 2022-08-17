package dashboard

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"github.com/fairwindsops/goldilocks/pkg/dashboard/helpers"
	"github.com/gobuffalo/packr/v2"
	"k8s.io/klog/v2"
)

var templateBox = (*packr.Box)(nil)

// templates
const (
	ContainerTemplateName = "container.gohtml"
	DashboardTemplateName = "dashboard.gohtml"
	FilterTemplateName    = "filter.gohtml"
	FooterTemplateName    = "footer.gohtml"
	HeadTemplateName      = "head.gohtml"
	NamespaceTemplateName = "namespace.gohtml"
	NavbarTemplateName    = "navbar.gohtml"
	EmailTemplateName     = "email.gohtml"
	ApiTokenTemplateName  = "api_token.gohtml"
)

var (
	// templates with these names are included by default in getTemplate()
	defaultIncludedTemplates = []string{
		"head",
		"navbar",
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

// getTemplateBox returns a binary-friendly set of templates for rendering the dash
func getTemplateBox() *packr.Box {
	if templateBox == (*packr.Box)(nil) {
		templateBox = packr.New("Templates", "templates")
	}
	return templateBox
}

// getTemplate puts together a template. Individual pieces can be overridden before rendering.
func getTemplate(name string, includedTemplates ...string) (*template.Template, error) {
	tmpl := template.New(name).Funcs(template.FuncMap{
		"printResource":  helpers.PrintResource,
		"getStatus":      helpers.GetStatus,
		"getStatusRange": helpers.GetStatusRange,
		"resourceName":   helpers.ResourceName,
		"getUUID":        helpers.GetUUID,
	})

	// join the default templates and included templates
	templatesToParse := make([]string, 0, len(includedTemplates)+len(defaultIncludedTemplates))
	templatesToParse = append(templatesToParse, defaultIncludedTemplates...)
	templatesToParse = append(templatesToParse, includedTemplates...)

	return parseTemplateFiles(tmpl, templatesToParse)
}

// parseTemplateFiles combines the template with the included templates into one parsed template
func parseTemplateFiles(tmpl *template.Template, includedTemplates []string) (*template.Template, error) {
	templateBox := getTemplateBox()
	for _, fname := range includedTemplates {
		templateFile, err := templateBox.Find(fmt.Sprintf("%s.gohtml", fname))
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
		BasePath: validateBasePath(opts.basePath),
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
