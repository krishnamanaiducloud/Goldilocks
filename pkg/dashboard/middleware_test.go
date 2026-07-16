package dashboard

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSecurityHeadersMiddleware(t *testing.T) {
	handler := SecurityHeadersMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	request := httptest.NewRequest(http.MethodGet, "https://goldilocks.example.test/", nil)
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", response.Code)
	}
	if !strings.Contains(response.Header().Get("Content-Security-Policy"), "frame-ancestors 'none'") {
		t.Error("missing restrictive frame-ancestors policy")
	}
	if response.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Error("missing nosniff header")
	}
	if response.Header().Get("X-Request-Id") == "" {
		t.Error("missing request ID")
	}
	if response.Header().Get("Strict-Transport-Security") == "" {
		t.Error("missing HSTS on HTTPS")
	}
}

func TestSecurityHeadersRejectsUntrustedRequestID(t *testing.T) {
	handler := SecurityHeadersMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("X-Request-Id", "bad request id\n")
	response := httptest.NewRecorder()

	handler.ServeHTTP(response, request)

	if response.Header().Get("X-Request-Id") == "bad request id\n" {
		t.Error("unsafe request ID was reflected")
	}
}

func TestReadOnlyMiddleware(t *testing.T) {
	handler := ReadOnlyMiddleware(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/api/namespaces", nil))

	if response.Code != http.StatusMethodNotAllowed {
		t.Fatalf("expected 405, got %d", response.Code)
	}
	if response.Header().Get("Allow") != "GET, HEAD" {
		t.Fatalf("unexpected Allow header: %q", response.Header().Get("Allow"))
	}
}

func TestRouterSupportsRootHealthAndPathPrefixedUI(t *testing.T) {
	router := GetRouter(BasePath("/goldilocks"), WithVersion("test-version"))

	health := httptest.NewRecorder()
	router.ServeHTTP(health, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if health.Code != http.StatusOK {
		t.Fatalf("expected root health endpoint to return 200, got %d", health.Code)
	}

	version := httptest.NewRecorder()
	router.ServeHTTP(version, httptest.NewRequest(http.MethodGet, "/goldilocks/api/version", nil))
	if version.Code != http.StatusOK {
		t.Fatalf("expected path-prefixed API to return 200, got %d", version.Code)
	}
	if !strings.Contains(version.Body.String(), "test-version") {
		t.Fatalf("unexpected version response: %s", version.Body.String())
	}
	if version.Header().Get("Content-Security-Policy") == "" {
		t.Error("router did not apply security middleware to the application subrouter")
	}

	outsidePrefix := httptest.NewRecorder()
	router.ServeHTTP(outsidePrefix, httptest.NewRequest(http.MethodGet, "/api/version", nil))
	if outsidePrefix.Code != http.StatusNotFound {
		t.Fatalf("expected API outside base path to return 404, got %d", outsidePrefix.Code)
	}
}

func TestRouterSupportsArbitraryNestedRuntimePrefixes(t *testing.T) {
	tests := []string{
		"/teams/platform/tools/goldilocks",
		"/products/finance/recommendations/",
	}
	for _, prefix := range tests {
		t.Run(prefix, func(t *testing.T) {
			normalized := validateBasePath(prefix)
			router := GetRouter(BasePath(prefix), WithVersion("runtime-prefix-test"))

			version := httptest.NewRecorder()
			router.ServeHTTP(version, httptest.NewRequest(http.MethodGet, normalized+"api/version", nil))
			if version.Code != http.StatusOK {
				t.Fatalf("expected API below %q to return 200, got %d", normalized, version.Code)
			}

			asset := httptest.NewRecorder()
			router.ServeHTTP(asset, httptest.NewRequest(http.MethodGet, normalized+"static/js/main.js", nil))
			if asset.Code != http.StatusOK {
				t.Fatalf("expected static asset below %q to return 200, got %d", normalized, asset.Code)
			}

			unprefixed := httptest.NewRecorder()
			router.ServeHTTP(unprefixed, httptest.NewRequest(http.MethodGet, "/api/version", nil))
			if unprefixed.Code != http.StatusNotFound {
				t.Fatalf("expected unprefixed API to remain unavailable, got %d", unprefixed.Code)
			}
		})
	}
}

func TestValidateBasePathRejectsTraversal(t *testing.T) {
	if got := validateBasePath("goldilocks"); got != "/goldilocks/" {
		t.Fatalf("unexpected normalized base path %q", got)
	}
	if got := validateBasePath("/../admin"); got != "/" {
		t.Fatalf("expected traversal path to be rejected, got %q", got)
	}
}
