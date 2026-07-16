package dashboard

import (
	"compress/gzip"
	"crypto/rand"
	"encoding/hex"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"k8s.io/klog/v2"
)

// SecurityHeadersMiddleware applies a browser-security baseline and a request ID.
func SecurityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-Id")
		if !validRequestID(requestID) {
			requestID = newRequestID()
		}
		w.Header().Set("X-Request-Id", requestID)
		w.Header().Set("Cache-Control", "no-store")
		w.Header().Set("Content-Security-Policy", strings.Join([]string{
			"default-src 'self'",
			"base-uri 'self'",
			"connect-src 'self'",
			"font-src 'self'",
			"form-action 'self'",
			"frame-ancestors 'none'",
			"img-src 'self' data:",
			"object-src 'none'",
			"script-src 'self'",
			"style-src 'self'",
		}, "; "))
		w.Header().Set("Cross-Origin-Resource-Policy", "same-origin")
		w.Header().Set("Permissions-Policy", "camera=(), geolocation=(), microphone=()")
		w.Header().Set("Referrer-Policy", "no-referrer")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		if r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https") {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		}
		next.ServeHTTP(w, r)
	})
}

// ReadOnlyMiddleware rejects mutation methods; the dashboard is a read-only UI/API.
func ReadOnlyMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			w.Header().Set("Allow", "GET, HEAD")
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func validRequestID(value string) bool {
	if len(value) < 1 || len(value) > 100 {
		return false
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || strings.ContainsRune("._:-", r) {
			continue
		}
		return false
	}
	return true
}

func newRequestID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return "request-id-unavailable"
	}
	return hex.EncodeToString(bytes)
}

// gzipResponseWriter wraps http.ResponseWriter with gzip compression
type gzipResponseWriter struct {
	io.Writer
	http.ResponseWriter
	statusCode int
}

func (w *gzipResponseWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

func (w *gzipResponseWriter) Write(b []byte) (int, error) {
	return w.Writer.Write(b)
}

var gzipPool = sync.Pool{
	New: func() interface{} {
		gz, _ := gzip.NewWriterLevel(nil, gzip.BestSpeed)
		return gz
	},
}

// GzipMiddleware compresses responses for clients that accept gzip encoding.
// It skips compression for responses smaller than typical HTTP headers overhead.
func GzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Accept-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}

		gz := gzipPool.Get().(*gzip.Writer)
		defer gzipPool.Put(gz)
		gz.Reset(w)

		w.Header().Set("Content-Encoding", "gzip")
		w.Header().Set("Vary", "Accept-Encoding")
		w.Header().Del("Content-Length")

		gzw := &gzipResponseWriter{Writer: gz, ResponseWriter: w}
		next.ServeHTTP(gzw, r)
		gz.Close()
	})
}

// LoggingMiddleware logs each HTTP request with method, path, status, and duration.
func LoggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		lrw := &loggingResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(lrw, r)

		duration := time.Since(start)
		klog.V(2).Infof("request_id=%s method=%s path=%s status=%d duration=%s", lrw.Header().Get("X-Request-Id"), r.Method, r.URL.Path, lrw.statusCode, duration)
	})
}

type loggingResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (lrw *loggingResponseWriter) WriteHeader(code int) {
	lrw.statusCode = code
	lrw.ResponseWriter.WriteHeader(code)
}
