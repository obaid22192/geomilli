package handlers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/geomilli/geomilli-api/internal/config"
	"github.com/geomilli/geomilli-api/internal/importer"
	"github.com/geomilli/geomilli-api/internal/meili"
	"github.com/geomilli/geomilli-api/internal/metrics"
	"github.com/go-chi/chi/v5"
)

func TestErrorEnvelopeShape(t *testing.T) {
	tests := []struct {
		name           string
		status         int
		code           ErrorCode
		message        string
		expectedStatus int
	}{
		{
			name:           "missing_parameter",
			status:         http.StatusBadRequest,
			code:           ErrMissingParameter,
			message:        "Parameter 'lat' is required",
			expectedStatus: 400,
		},
		{
			name:           "invalid_parameter",
			status:         http.StatusBadRequest,
			code:           ErrInvalidParameter,
			message:        "Parameter 'lat' must be between -90 and 90",
			expectedStatus: 400,
		},
		{
			name:           "invalid_body",
			status:         http.StatusBadRequest,
			code:           ErrInvalidBody,
			message:        "Invalid JSON payload: unexpected EOF",
			expectedStatus: 400,
		},
		{
			name:           "not_found",
			status:         http.StatusNotFound,
			code:           ErrNotFound,
			message:        "Place not found for ID: node:123",
			expectedStatus: 404,
		},
		{
			name:           "upstream_unavailable 502",
			status:         http.StatusBadGateway,
			code:           ErrUpstreamUnavailable,
			message:        "Meilisearch connection timeout",
			expectedStatus: 502,
		},
		{
			name:           "upstream_unavailable 503",
			status:         http.StatusServiceUnavailable,
			code:           ErrUpstreamUnavailable,
			message:        "Harvester service client not configured",
			expectedStatus: 503,
		},
		{
			name:           "internal_error",
			status:         http.StatusInternalServerError,
			code:           ErrInternalError,
			message:        "Failed to load taxonomy config",
			expectedStatus: 500,
		},
		{
			name:           "unauthorized reserved",
			status:         http.StatusUnauthorized,
			code:           ErrUnauthorized,
			message:        "Bearer token invalid",
			expectedStatus: 401,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := httptest.NewRecorder()
			writeError(rec, tt.status, tt.code, tt.message)

			if rec.Code != tt.expectedStatus {
				t.Fatalf("expected HTTP status %d, got %d", tt.expectedStatus, rec.Code)
			}

			contentType := rec.Header().Get("Content-Type")
			if contentType != "application/json; charset=utf-8" {
				t.Errorf("expected Content-Type application/json; charset=utf-8, got %s", contentType)
			}

			var env ErrorEnvelope
			if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
				t.Fatalf("failed to decode ErrorEnvelope JSON: %v, body: %s", err, rec.Body.String())
			}

			if env.Error.Code != tt.code {
				t.Errorf("expected code %s, got %s", tt.code, env.Error.Code)
			}
			if env.Error.Message != tt.message {
				t.Errorf("expected message %q, got %q", tt.message, env.Error.Message)
			}
			if env.Error.Status != tt.expectedStatus {
				t.Errorf("expected inner status %d, got %d", tt.expectedStatus, env.Error.Status)
			}

			// Validate raw JSON structure: {"error":{"code":"...","message":"...","status":...}}
			var raw map[string]any
			if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
				t.Fatalf("failed to decode raw JSON: %v", err)
			}
			errObj, ok := raw["error"].(map[string]any)
			if !ok {
				t.Fatalf("expected 'error' key to be a JSON object, got %T", raw["error"])
			}
			if errObj["code"] != string(tt.code) {
				t.Errorf("expected raw code %s, got %v", tt.code, errObj["code"])
			}
			if errObj["message"] != tt.message {
				t.Errorf("expected raw message %s, got %v", tt.message, errObj["message"])
			}
			if int(errObj["status"].(float64)) != tt.expectedStatus {
				t.Errorf("expected raw status %d, got %v", tt.expectedStatus, errObj["status"])
			}
		})
	}
}

func TestEndpointErrorEnvelopes(t *testing.T) {
	r, _ := setupTestRouter()

	t.Run("missing_parameter on reverse", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/reverse", nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
		var env ErrorEnvelope
		if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
			t.Fatalf("failed to decode: %v", err)
		}
		if env.Error.Code != ErrMissingParameter {
			t.Errorf("expected %s, got %s", ErrMissingParameter, env.Error.Code)
		}
		if env.Error.Status != 400 {
			t.Errorf("expected status 400, got %d", env.Error.Status)
		}
	})

	t.Run("invalid_parameter on nearby", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/nearby?lat=999&lng=4", nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
		var env ErrorEnvelope
		if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
			t.Fatalf("failed to decode: %v", err)
		}
		if env.Error.Code != ErrInvalidParameter {
			t.Errorf("expected %s, got %s", ErrInvalidParameter, env.Error.Code)
		}
	})

	t.Run("invalid_body on importer jobs", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodPost, "/v1/importer/jobs", bytes.NewBufferString("{invalid-json"))
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Fatalf("expected 400, got %d", rec.Code)
		}
		var env ErrorEnvelope
		if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
			t.Fatalf("failed to decode: %v", err)
		}
		if env.Error.Code != ErrInvalidBody {
			t.Errorf("expected %s, got %s", ErrInvalidBody, env.Error.Code)
		}
	})

	t.Run("not_found on places", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/places/node:999999", nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusNotFound {
			t.Fatalf("expected 404, got %d", rec.Code)
		}
		var env ErrorEnvelope
		if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
			t.Fatalf("failed to decode: %v", err)
		}
		if env.Error.Code != ErrNotFound {
			t.Errorf("expected %s, got %s", ErrNotFound, env.Error.Code)
		}
		if env.Error.Status != 404 {
			t.Errorf("expected status 404, got %d", env.Error.Status)
		}
	})

	t.Run("upstream_unavailable on meili error", func(t *testing.T) {
		cfg := &config.Config{
			Port:           "8080",
			MeiliHost:      "http://mock-meili:7700",
			MeiliIndex:     "places",
			HarvesterURL:   "http://mock-harvester:8085",
			IsCloud:        false,
			AllowedOrigins: []string{"*"},
		}
		failingClient := &http.Client{
			Transport: mockTransport(func(req *http.Request) (*http.Response, error) {
				return nil, errors.New("connection refused to Meilisearch")
			}),
		}
		meiliClient := meili.NewClientWithHTTPClient(cfg, failingClient)
		importerClient := importer.NewClientWithHTTPClient(cfg, failingClient)
		collector := metrics.NewCollector()
		handler := NewHandler(cfg, meiliClient, importerClient, collector)
		failRouter := chi.NewRouter()
		handler.RegisterRoutes(failRouter)

		req := httptest.NewRequest(http.MethodGet, "/v1/search?q=test", nil)
		rec := httptest.NewRecorder()
		failRouter.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadGateway {
			t.Fatalf("expected 502 Bad Gateway, got %d", rec.Code)
		}
		var env ErrorEnvelope
		if err := json.Unmarshal(rec.Body.Bytes(), &env); err != nil {
			t.Fatalf("failed to decode: %v", err)
		}
		if env.Error.Code != ErrUpstreamUnavailable {
			t.Errorf("expected %s, got %s", ErrUpstreamUnavailable, env.Error.Code)
		}
		if env.Error.Status != 502 {
			t.Errorf("expected status 502, got %d", env.Error.Status)
		}
	})
}

func TestRadiusClampingAndEcho(t *testing.T) {
	r, _ := setupTestRouter()

	t.Run("reverse radius echo default", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/reverse?lat=50.8503&lng=4.3517", nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		var resp ReverseResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode: %v", err)
		}
		if resp.RadiusMeters != DefaultReverseRadius {
			t.Errorf("expected default radius %d, got %d", DefaultReverseRadius, resp.RadiusMeters)
		}
	})

	t.Run("reverse radius cap at 100000", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/reverse?lat=50.8503&lng=4.3517&radius_meters=200000", nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		var resp ReverseResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode: %v", err)
		}
		if resp.RadiusMeters != MaxReverseRadius {
			t.Errorf("expected capped radius %d, got %d", MaxReverseRadius, resp.RadiusMeters)
		}
	})

	t.Run("reverse custom radius within range", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/reverse?lat=50.8503&lng=4.3517&radius_meters=3000", nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		var resp ReverseResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode: %v", err)
		}
		if resp.RadiusMeters != 3000 {
			t.Errorf("expected radius 3000, got %d", resp.RadiusMeters)
		}
	})

	t.Run("nearby radius cap at 100000", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/nearby?lat=50.85&lng=4.35&radius_meters=250000", nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		var resp NearbyResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode: %v", err)
		}
		if resp.RadiusMeters != MaxNearbyRadius {
			t.Errorf("expected capped radius %d, got %d", MaxNearbyRadius, resp.RadiusMeters)
		}
	})

	t.Run("nearby custom radius echo", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/v1/nearby?lat=50.85&lng=4.35&radius_meters=3000", nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Fatalf("expected 200, got %d", rec.Code)
		}
		var resp NearbyResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("failed to decode: %v", err)
		}
		if resp.RadiusMeters != 3000 {
			t.Errorf("expected radius 3000, got %d", resp.RadiusMeters)
		}
	})
}
