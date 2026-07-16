package dashboard

import (
	"context"
	"net/http"
	"time"

	"github.com/fairwindsops/goldilocks/pkg/kube"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
)

const readinessTimeout = 2 * time.Second

// ReadinessCheck verifies that a dependency required to serve dashboard data is available.
type ReadinessCheck func(context.Context) error

// Health replies with the status messages given for healthy
func Health(healthyMessage string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := w.Write([]byte(healthyMessage))
		if err != nil {
			klog.Errorf("Error writing healthcheck: %v", err)
		}
	})
}

// Healthz replies with a zero byte 200 response
func Healthz() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
}

// KubernetesReadinessCheck verifies both Kubernetes API connectivity and the dashboard's
// permission to list namespaces. Namespace list permission is already required by the UI.
func KubernetesReadinessCheck(ctx context.Context) error {
	return kubernetesReadinessCheck(kube.GetInstance().Client)(ctx)
}

func kubernetesReadinessCheck(client kubernetes.Interface) ReadinessCheck {
	return func(ctx context.Context) error {
		_, err := client.CoreV1().Namespaces().List(ctx, metav1.ListOptions{Limit: 1})
		return err
	}
}

// Readyz runs a readiness check with a strict deadline. Dependency errors are intentionally
// not returned to clients because Kubernetes API errors can contain sensitive details.
func Readyz(check ReadinessCheck, timeout time.Duration) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		if check == nil {
			http.Error(w, "not ready", http.StatusServiceUnavailable)
			return
		}

		ctx, cancel := context.WithTimeout(r.Context(), timeout)
		defer cancel()
		if err := check(ctx); err != nil {
			klog.Warningf("Readiness check failed: %v", err)
			http.Error(w, "not ready", http.StatusServiceUnavailable)
			return
		}

		w.WriteHeader(http.StatusOK)
	})
}
