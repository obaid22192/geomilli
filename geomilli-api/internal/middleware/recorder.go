package middleware

import (
	"context"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/geomilli/geomilli-api/internal/metrics"
	"github.com/go-chi/chi/v5"
)

type hitsKey struct{}

// SetResultHits attaches the number of returned hits to the request context.
func SetResultHits(r *http.Request, hits int) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), hitsKey{}, hits))
}

// GetResultHits extracts the number of hits from context, defaulting to 1 on 200 OK.
func GetResultHits(r *http.Request) int {
	if v := r.Context().Value(hitsKey{}); v != nil {
		if hits, ok := v.(int); ok {
			return hits
		}
	}
	return 1
}

// responseWriter wraps http.ResponseWriter to capture status code.
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Flush() {
	if flusher, ok := rw.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

// TelemetryRecorder captures response metrics and feeds them to the Collector.
func TelemetryRecorder(collector *metrics.Collector) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			next.ServeHTTP(rw, r)

			duration := time.Since(start)

			// Determine route pattern (e.g. /v1/autocomplete, /v1/places/{id})
			routePattern := r.URL.Path
			rctx := chi.RouteContext(r.Context())
			if rctx != nil && rctx.RoutePattern() != "" {
				routePattern = rctx.RoutePattern()
			}

			// Clean up route pattern
			if idx := strings.Index(routePattern, "?"); idx != -1 {
				routePattern = routePattern[:idx]
			}

			query := r.URL.Query().Get("q")
			hits := GetResultHits(r)

			collector.Record(routePattern, rw.statusCode, duration, query, hits)
		})
	}
}

// RequestLogger logs incoming HTTP requests with structured output.
func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		next.ServeHTTP(rw, r)

		duration := time.Since(start)
		log.Printf("[HTTP] %s %s -> %d (%s)",
			r.Method,
			r.URL.Path,
			rw.statusCode,
			duration.Round(time.Microsecond),
		)
	})
}
