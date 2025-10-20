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
	"sync"

	"github.com/fairwindsops/goldilocks/pkg/dashboard/helpers"
	"k8s.io/klog/v2"
)

// ✅ Embed all templates
//go:embed templates/*.gohtml
var embeddedTemplates embed.FS

// to cache parsed templates
var (
	templateCache   = make(map[string]*template.Template)
	templateCacheMu sync.RWMutex
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

// ✅ getTemplate replaces getTemplateBox + parseTemplateFiles
func getTemplate(name string, opts Options, includedTemplates ...string) (*template.Template, error) {
	cacheKey := name + "|" + strings.Join(includedTemplates, ",")

	// ✅ Check cache first
	templateCacheMu.RLock()
	if t, ok := templateCache[cacheKey]; ok {
		templateCacheMu.RUnlock()
		return t, nil
	}
	templateCacheMu.RUnlock()

	// ✅ Create a new template with funcs
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

	// ✅ Determine all templates to include (default + specific)
	templatesToParse := make([]string, 0, len(includedTemplates)+len(defaultIncludedTemplates))
	templatesToParse = append(templatesToParse, defaultIncludedTemplates...)
	templatesToParse = append(templatesToParse, includedTemplates...)

	// ✅ Parse each template file from embed
	for _, fname := range templatesToParse {
		filePath := fmt.Sprintf("templates/%s.gohtml", fname)
		content, err := fs.ReadFile(embeddedTemplates, filePath)
		if err != nil {
			return nil, fmt.Errorf("error reading template %s: %w", filePath, err)
		}
		tmpl, err = tmpl.Parse(string(content))
		if err != nil {
			return nil, fmt.Errorf("error parsing template %s: %w", filePath, err)
		}
	}

	// ✅ Cache the template
	templateCacheMu.Lock()
	templateCache[cacheKey] = tmpl
	templateCacheMu.Unlock()

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

