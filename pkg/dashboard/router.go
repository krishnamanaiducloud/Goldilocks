package dashboard

import (
	"net/http"
	"path"
	"strings"

	"k8s.io/klog/v2"
	"github.com/gorilla/mux"
)

// GetRouter returns a mux router serving all routes necessary for the dashboard
func GetRouter(setters ...Option) *mux.Router {
	opts := defaultOptions()
	for _, setter := range setters {
		setter(opts)
	}

	router := mux.NewRouter().
		PathPrefix(strings.TrimSuffix(opts.BasePath, "/")).
		Subrouter().
		StrictSlash(true)

	// health
	router.Handle("/health", Health("OK"))
	router.Handle("/healthz", Healthz())

	// ✅ Favicon
	router.Handle("/favicon.ico", Asset("/images/favicon-32x32.png"))

	// ✅ Static assets (replaces Packr + GetAssetBox)
	// Previously: http.FileServer(GetAssetBox())
	// Now using embed via StaticAssets()
	router.PathPrefix("/static/").Handler(
		http.StripPrefix(
			path.Join(opts.BasePath, "/static/"),
			StaticAssets("/static/"),
		),
	)

	// dashboard
	router.Handle("/dashboard", Dashboard(*opts))
	router.Handle("/dashboard/{namespace:[a-zA-Z0-9-]+}", Dashboard(*opts))

	// namespace list
	router.Handle("/namespaces", NamespaceList(*opts))

	// root
	router.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" && r.URL.Path != opts.BasePath && r.URL.Path != opts.BasePath+"/" {
			klog.Infof("404: %s", r.URL.Path)
			http.NotFound(w, r)
			return
		}

		klog.Infof("redirecting to %v", path.Join(opts.BasePath, "/namespaces"))
		http.Redirect(w, r, path.Join(opts.BasePath, "/namespaces"), http.StatusMovedPermanently)
	})

	// api
	router.Handle("/api/{namespace:[a-zA-Z0-9-]+}", API(*opts))

	// ✅ Markdown (docs) support - TODO: convert Packr to embed in next step
	// Previous: GetMarkdownBox() with packr("../../docs")
	// If needed, we will embed docs in a separate step.

	return router
}

