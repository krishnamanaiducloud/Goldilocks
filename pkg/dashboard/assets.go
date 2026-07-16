package dashboard

import (
	"bytes"
	"embed"
	"io/fs"
	"mime"
	"net/http"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"k8s.io/klog/v2"
)

// Embed everything under pkg/dashboard/assets/
//
//go:embed assets/**/*
var embeddedAssets embed.FS

var (
	assetsFS   http.FileSystem
	assetsOnce sync.Once
)

// getAssetsFS returns an http.FileSystem for serving embedded assets.
func getAssetsFS() http.FileSystem {
	assetsOnce.Do(func() {
		sub, err := fs.Sub(embeddedAssets, "assets")
		if err != nil {
			klog.Errorf("Error loading embedded assets: %v", err)
			assetsFS = http.FS(embeddedAssets)
			return
		}
		assetsFS = http.FS(sub)
	})
	return assetsFS
}

// normalize: strip leading "/" and optional "static/" prefix
func normalize(p string) string {
	p = strings.TrimPrefix(p, "/")
	if strings.HasPrefix(p, "static/") {
		p = strings.TrimPrefix(p, "static/")
	}
	return p
}

// Asset serves a single embedded asset by path.
func Asset(assetPath string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cleaned := normalize(assetPath)

		data, err := fs.ReadFile(embeddedAssets, "assets/"+cleaned)
		if err != nil {
			klog.Errorf("Error getting asset %s: %v", cleaned, err)
			http.NotFound(w, r)
			return
		}

		// Set a correct content type when possible.
		if ct := mime.TypeByExtension(filepath.Ext(cleaned)); ct != "" {
			w.Header().Set("Content-Type", ct)
		}

		// Cache for a day
		w.Header().Set("Cache-Control", "public, max-age=86400")

		http.ServeContent(w, r, cleaned, time.Time{}, bytes.NewReader(data))
	})
}

// StaticAssets serves all embedded static assets under /static/
func StaticAssets(prefix string) http.Handler {
	klog.V(3).Infof("Serving embedded static assets with prefix: %s", prefix)
	files := http.StripPrefix(prefix, http.FileServer(getAssetsFS()))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "public, max-age=86400")
		files.ServeHTTP(w, r)
	})
}
