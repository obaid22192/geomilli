package middleware_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/geomilli/geomilli-api/internal/metrics"
	"github.com/geomilli/geomilli-api/internal/middleware"
	"github.com/go-chi/chi/v5"
)

func TestCORSPreflightAndHeaders(t *testing.T) {
	corsMw := middleware.CORS([]string{"https://geomilli.example.com"})

	handler := corsMw(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("ok"))
	}))

	// Test Preflight OPTIONS
	req := httptest.NewRequest("OPTIONS", "/v1/search", nil)
	req.Header.Set("Origin", "https://geomilli.example.com")
	req.Header.Set("Access-Control-Request-Method", "GET")
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Header().Get("Access-Control-Allow-Origin") != "https://geomilli.example.com" {
		t.Errorf("expected Access-Control-Allow-Origin to match, got %s", rec.Header().Get("Access-Control-Allow-Origin"))
	}
	if rec.Code != http.StatusOK && rec.Code != http.StatusNoContent {
		t.Errorf("expected 200 or 204 for OPTIONS preflight, got %d", rec.Code)
	}

	// Test Non-Allowed Origin
	req2 := httptest.NewRequest("GET", "/v1/search", nil)
	req2.Header.Set("Origin", "https://unauthorized.com")
	rec2 := httptest.NewRecorder()
	handler.ServeHTTP(rec2, req2)
	if rec2.Header().Get("Access-Control-Allow-Origin") == "https://unauthorized.com" {
		t.Errorf("unauthorized origin should not receive allow header")
	}
}

func TestTelemetryRecorder(t *testing.T) {
	collector := metrics.NewCollector()
	recMw := middleware.TelemetryRecorder(collector)

	r := chi.NewRouter()
	r.Use(recMw)

	r.Get("/v1/search", func(w http.ResponseWriter, req *http.Request) {
		*req = *middleware.SetResultHits(req, 5)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"hits": 5}`))
	})

	r.Get("/v1/zero-hits", func(w http.ResponseWriter, req *http.Request) {
		*req = *middleware.SetResultHits(req, 0)
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"hits": 0}`))
	})

	// Perform search request with query
	req := httptest.NewRequest("GET", "/v1/search?q=brussels", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	// Perform zero hits query
	reqZero := httptest.NewRequest("GET", "/v1/zero-hits?q=nonexistentplace123", nil)
	recZero := httptest.NewRecorder()
	r.ServeHTTP(recZero, reqZero)

	if recZero.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", recZero.Code)
	}

	snap := collector.Snapshot(metrics.MeiliInfo{})
	if snap.TotalRequests != 2 {
		t.Errorf("expected 2 requests recorded, got %d", snap.TotalRequests)
	}
	if snap.StatusCodes[200] != 2 {
		t.Errorf("expected status code 200 count = 2, got %d", snap.StatusCodes[200])
	}

	// Verify top queries
	foundBrussels := false
	for _, tq := range snap.TopQueries {
		if tq.Query == "brussels" {
			foundBrussels = true
			break
		}
	}
	if !foundBrussels {
		t.Errorf("expected 'brussels' in top queries, got %v", snap.TopQueries)
	}

	// Verify zero-hit query was logged
	foundZero := false
	for _, zq := range snap.ZeroHitQueries {
		if zq.Query == "nonexistentplace123" {
			foundZero = true
			break
		}
	}
	if !foundZero {
		t.Errorf("expected 'nonexistentplace123' in zero-hit queries, got %v", snap.ZeroHitQueries)
	}
}
