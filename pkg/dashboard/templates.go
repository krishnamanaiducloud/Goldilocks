package dashboard

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"strings"

	"github.com/fairwindsops/goldilocks/pkg/dashboard/helpers"
	"k8s.io/klog/v2"
)

//
// ✅ Embed server-side templates
//go:embed templates/*.gohtml
var templatesFS embed.FS

var (
	// templates with these names are included by default in getTemplate()
	defaultIncludedTemplates = []string{
		"head",
		"navigation",
		"footer",
	}
)

// to be included in data structs
type baseTemplateData struct {
	BasePath string
	Data     interface{}
	JSON     template.JS
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
		"opts": func() Options { return opts },
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
		b, err := fs.ReadFile(templatesFS, fmt.Sprintf("templates/%s.gohtml", fname))
		if err != nil {
			return nil, err
		}
		tmpl, err = tmpl.Parse(string(b))
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
	if _, err = buf.WriteTo(w); err != nil {
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

