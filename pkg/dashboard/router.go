// Copyright 2019 FairwindsOps
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

	// ✅ Serve real ICO (browsers fetch /favicon.ico explicitly)
	router.Handle("/favicon.ico", Asset("/images/favicon.ico"))

	// static assets
	router.PathPrefix("/static/").
		Handler(StaticAssets(path.Join(opts.BasePath, "/static/")))

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
		http.Redirect(w, r, path.Join(opts.BasePath, "/namespaces"), http.StatusMovedPermanently)
	})

	// api
	router.Handle("/api/{namespace:[a-zA-Z0-9-]+}", API(*opts))
	return router
}

