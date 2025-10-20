package dashboard

import (
	"embed"
	"io/fs"
	"net/http"
	"k8s.io/klog/v2"
)

var embeddedAssets embed.FS

var assetsFS http.FileSystem

func getAssetsFS() http.FileSystem {
	if assetsFS == nil {
		sub, err := fs.Sub(embeddedAssets, "assets")
		if err != nil {
			klog.Errorf("Error loading embedded assets: %v", err)
			return http.FS(embeddedAssets)
		}
		assetsFS = http.FS(sub)
	}
	return assetsFS
}

func Asset(assetPath string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		asset, err := fs.ReadFile(embeddedAssets, "assets/"+assetPath)
		if err != nil {
			klog.Errorf("Error getting asset: %v", err)
			http.Error(w, "Error getting asset", http.StatusInternalServerError)
			return
		}
		_, err = w.Write(asset)
		if err != nil {
			klog.Errorf("Error writing asset: %v", err)
		}
	})
}

func StaticAssets(prefix string) http.Handler {
	klog.V(3).Infof("stripping prefix: %s", prefix)
	return http.StripPrefix(prefix, http.FileServer(getAssetsFS()))
}

