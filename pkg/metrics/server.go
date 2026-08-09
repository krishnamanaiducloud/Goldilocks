package metrics

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	vpaclient "k8s.io/autoscaler/vertical-pod-autoscaler/pkg/client/clientset/versioned"
)

// NewServer creates a dedicated metrics server and registry. A dedicated
// registry prevents unrelated global collectors from being registered twice.
func NewServer(address string, client vpaclient.Interface) *http.Server {
	registry := prometheus.NewRegistry()
	registry.MustRegister(
		prometheus.NewGoCollector(),
		prometheus.NewProcessCollector(prometheus.ProcessCollectorOpts{}),
		NewVPACollector(client),
	)
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.HandlerFor(registry, promhttp.HandlerOpts{}))
	return &http.Server{
		Addr:              address,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}

func RunServer(server *http.Server, stop <-chan struct{}, reportError func(error)) {
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			reportError(err)
		}
	}()
	go func() {
		<-stop
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(ctx); err != nil {
			reportError(err)
		}
	}()
}
