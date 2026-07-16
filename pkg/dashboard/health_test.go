package dashboard

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"
)

func TestKubernetesReadinessCheckSuccess(t *testing.T) {
	client := fake.NewSimpleClientset()

	if err := kubernetesReadinessCheck(client)(context.Background()); err != nil {
		t.Fatalf("expected namespace-list readiness check to pass: %v", err)
	}
	if actions := client.Actions(); len(actions) != 1 || !actions[0].Matches("list", "namespaces") {
		t.Fatalf("expected one namespace list action, got %#v", actions)
	}
}

func TestKubernetesReadinessCheckFailure(t *testing.T) {
	client := fake.NewSimpleClientset()
	permissionError := errors.New("namespace list is forbidden")
	client.PrependReactor("list", "namespaces", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, permissionError
	})

	if err := kubernetesReadinessCheck(client)(context.Background()); !errors.Is(err, permissionError) {
		t.Fatalf("expected Kubernetes permission error, got %v", err)
	}
}

func TestReadyzSuccess(t *testing.T) {
	handler := Readyz(func(ctx context.Context) error {
		deadline, ok := ctx.Deadline()
		if !ok {
			return errors.New("readiness context has no deadline")
		}
		if remaining := time.Until(deadline); remaining <= 0 || remaining > 100*time.Millisecond {
			return errors.New("readiness deadline is outside the configured bound")
		}
		return nil
	}, 100*time.Millisecond)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/readyz", nil))

	if response.Code != http.StatusOK {
		t.Fatalf("expected ready response, got %d", response.Code)
	}
	if response.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("expected readiness response to disable caching")
	}
}

func TestReadyzFailure(t *testing.T) {
	dependencyError := errors.New("forbidden: sensitive cluster detail")
	handler := Readyz(func(context.Context) error {
		return dependencyError
	}, 100*time.Millisecond)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/readyz", nil))

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected not-ready response, got %d", response.Code)
	}
	if strings.Contains(response.Body.String(), dependencyError.Error()) {
		t.Fatal("readiness response leaked the Kubernetes API error")
	}
}

func TestReadyzTimesOut(t *testing.T) {
	handler := Readyz(func(ctx context.Context) error {
		<-ctx.Done()
		return ctx.Err()
	}, 10*time.Millisecond)
	response := httptest.NewRecorder()
	started := time.Now()

	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/readyz", nil))

	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected timed-out readiness response, got %d", response.Code)
	}
	if elapsed := time.Since(started); elapsed > time.Second {
		t.Fatalf("readiness check was not bounded: %v", elapsed)
	}
}
