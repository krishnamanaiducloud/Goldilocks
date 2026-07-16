package dashboard

import (
	"bytes"
	"embed"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"
	"strings"

	"github.com/fairwindsops/goldilocks/pkg/dashboard/helpers"
	"k8s.io/klog/v2"
)

// ✅ Embed server-side templates
//
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
		"opts":           func() Options { return opts },
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
	err := tmpl.Execute(buf, baseTemplateData{
		BasePath: validateBasePath(opts.BasePath),
		Data:     data,
	})
	if err != nil {
		klog.Errorf("Error executing template: %v", err)
		http.Error(w, "Error rendering dashboard", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	if _, err = buf.WriteTo(w); err != nil {
		klog.Errorf("Error writing template: %v", err)
	}
}

func validateBasePath(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || value == "/" {
		return "/"
	}
	segments := strings.Split(strings.Trim(value, "/"), "/")
	for _, segment := range segments {
		if segment == "" || segment == "." || segment == ".." {
			return "/"
		}
		for _, r := range segment {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || strings.ContainsRune("._~-", r) {
				continue
			}
			return "/"
		}
	}
	return "/" + strings.Join(segments, "/") + "/"
}
