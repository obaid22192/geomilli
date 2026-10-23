package handlers

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/geomilli/geomilli-api/internal/config"
	"github.com/geomilli/geomilli-api/internal/importer"
	"github.com/geomilli/geomilli-api/internal/meili"
	"github.com/geomilli/geomilli-api/internal/metrics"
	"github.com/geomilli/geomilli-api/internal/schema"
	"github.com/go-chi/chi/v5"
)

type mockTransport func(req *http.Request) (*http.Response, error)

func (f mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func setupTestRouter() (*chi.Mux, *metrics.Collector) {
	cfg := &config.Config{
		Port:           "8080",
		MeiliHost:      "http://mock-meili:7700",
		MeiliIndex:     "places",
		HarvesterURL:   "http://mock-harvester:8085",
		IsCloud:        false,
		AllowedOrigins: []string{"*"},
	}

	meiliHTTPClient := &http.Client{
		Transport: mockTransport(func(req *http.Request) (*http.Response, error) {
			path := req.URL.Path

			if path == "/health" {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(`{"status":"available"}`)),
					Header:     make(http.Header),
				}, nil
			}

			if path == "/indexes/places/stats" {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(`{"numberOfDocuments":12345,"isIndexing":false}`)),
					Header:     make(http.Header),
				}, nil
			}

			if path == "/indexes/places/documents/node:123" {
				p := schema.Place{
					ID:          "node:123",
					EntityID:    "node:123",
					Class:       "TouristAttraction",
					NameDisplay: "Atomium",
					CountryCode: "BE",
					Geo:         schema.GeoPoint{Lat: 50.8949, Lng: 4.3415},
				}
				b, _ := json.Marshal(p)
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader(b)),
					Header:     make(http.Header),
				}, nil
			}

			if path == "/indexes/places/search" {
				var searchReq meili.SearchRequest
				_ = json.NewDecoder(req.Body).Decode(&searchReq)

				filterStr := fmt.Sprintf("%v", searchReq.Filter)
				if strings.Contains(filterStr, "999999") {
					respObj := meili.SearchResponse{
						Query: searchReq.Q,
						Hits:  []json.RawMessage{},
					}
					out, _ := json.Marshal(respObj)
					return &http.Response{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(bytes.NewReader(out)),
						Header:     make(http.Header),
					}, nil
				}

				samplePlace := schema.Place{
					ID:              "node:1",
					EntityID:        "node:1",
					Class:           "City",
					NameDisplay:     "Brussels",
					CountryCode:     "BE",
					CountryName:     "Belgium",
					NamesByLanguage: map[string]string{"fr": "Bruxelles", "nl": "Brussel"},
					IATACode:        "BRU",
					Geo:             schema.GeoPoint{Lat: 50.8503, Lng: 4.3517},
					PopularityScore: 0.95,
				}

				if searchReq.Filter != "" && searchReq.Filter == `class = "Airport"` {
					samplePlace.Class = "Airport"
					samplePlace.NameDisplay = "Brussels Airport"
				}

				respObj := meili.SearchResponse{
					Query:            searchReq.Q,
					ProcessingTimeMs: 3,
					Limit:            searchReq.Limit,
					TotalHits:        1,
					FacetDistribution: map[string]map[string]int{
						"country_code": {"BE": 100, "FR": 50},
					},
					Hits: []json.RawMessage{},
				}
				b, _ := json.Marshal(samplePlace)
				respObj.Hits = append(respObj.Hits, json.RawMessage(b))

				out, _ := json.Marshal(respObj)
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewReader(out)),
					Header:     make(http.Header),
				}, nil
			}

			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(bytes.NewBufferString(`{"message":"not found"}`)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	harvesterHTTPClient := &http.Client{
		Transport: mockTransport(func(req *http.Request) (*http.Response, error) {
			if req.URL.Path == "/api/v1/status" {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(`{"state":"idle","resources":{"heap_alloc_mb":25.0}}`)),
					Header:     make(http.Header),
				}, nil
			}
			if req.URL.Path == "/api/v1/countries" {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(`{"countries":[{"code":"BE","name":"Belgium","is_indexed":true}]}`)),
					Header:     make(http.Header),
				}, nil
			}
			if req.URL.Path == "/api/v1/jobs" && req.Method == http.MethodPost {
				return &http.Response{
					StatusCode: http.StatusAccepted,
					Body:       io.NopCloser(bytes.NewBufferString(`{"job_id":"job_test_01","status":"started"}`)),
					Header:     make(http.Header),
				}, nil
			}
			if req.URL.Path == "/api/v1/queue" && req.Method == http.MethodGet {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(`{"concurrency":2,"active_workers":[],"queued_tasks":[],"country_locks":{}}`)),
					Header:     make(http.Header),
				}, nil
			}
			if req.URL.Path == "/api/v1/queue" && req.Method == http.MethodDelete {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(`{"cleared":true}`)),
					Header:     make(http.Header),
				}, nil
			}
			if req.URL.Path == "/api/v1/queue/dequeue" && req.Method == http.MethodPost {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(`{"dequeued":true,"country":"NL"}`)),
					Header:     make(http.Header),
				}, nil
			}
			if req.URL.Path == "/api/v1/jobs/cancel" && req.Method == http.MethodPost {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(`{"cancelled":true,"message":"worker stopped"}`)),
					Header:     make(http.Header),
				}, nil
			}
			if req.URL.Path == "/api/v1/languages" && req.Method == http.MethodPut {
				return &http.Response{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(bytes.NewBufferString(`{"updated":true,"languages":[{"code":"th","name":"Thai","enabled":true}]}`)),
					Header:     make(http.Header),
				}, nil
			}
			return &http.Response{
				StatusCode: http.StatusNotFound,
				Body:       io.NopCloser(bytes.NewBufferString(`{}`)),
				Header:     make(http.Header),
			}, nil
		}),
	}

	meiliClient := meili.NewClientWithHTTPClient(cfg, meiliHTTPClient)
	importerClient := importer.NewClientWithHTTPClient(cfg, harvesterHTTPClient)
	collector := metrics.NewCollector()

	handler := NewHandler(cfg, meiliClient, importerClient, collector)
	r := chi.NewRouter()
	handler.RegisterRoutes(r)

	return r, collector
}

func TestAutocompleteHandler(t *testing.T) {
	r, _ := setupTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/v1/autocomplete?q=bru&lang=fr&country=BE&lat=50.85&lng=4.35", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d, body: %s", rec.Code, rec.Body.String())
	}

	var resp AutocompleteResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	if len(resp.Results) != 1 {
		t.Fatalf("expected 1 result, got %d", len(resp.Results))
	}
	// Check language localization to French
	if resp.Results[0].NameDisplay != "Bruxelles" {
		t.Fatalf("expected localized name Bruxelles, got %s", resp.Results[0].NameDisplay)
	}
	if resp.Results[0].DistanceMeters == nil {
		t.Fatalf("expected DistanceMeters to be populated")
	}
}

func TestSearchHandler(t *testing.T) {
	r, _ := setupTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/v1/search?q=brussels&facets=*", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp SearchResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode search response: %v", err)
	}
	if len(resp.Results) == 0 {
		t.Fatalf("expected search hits")
	}
	if resp.FacetDistribution == nil || len(resp.FacetDistribution["country_code"]) == 0 {
		t.Fatalf("expected facet distribution for country_code")
	}
}

func TestReverseHandler(t *testing.T) {
	r, _ := setupTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/v1/reverse?lat=50.8503&lng=4.3517", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp ReverseResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode reverse response: %v", err)
	}
	if resp.ClosestPlace == nil {
		t.Fatalf("expected closest place")
	}
	if resp.Address.City != "Brussels" {
		t.Fatalf("expected City Brussels, got %s", resp.Address.City)
	}
}

func TestNearbyHandler(t *testing.T) {
	r, _ := setupTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/v1/nearby?lat=50.85&lng=4.35&radius_meters=3000", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp NearbyResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode nearby response: %v", err)
	}
	if len(resp.Results) == 0 {
		t.Fatalf("expected nearby results")
	}
}

func TestBBoxHandler(t *testing.T) {
	r, _ := setupTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/v1/bbox?min_lat=50.8&min_lng=4.3&max_lat=50.9&max_lng=4.4", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp BBoxResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode bbox response: %v", err)
	}
	if len(resp.Results) == 0 {
		t.Fatalf("expected bbox results")
	}
}

func TestPlaceDetailsHandler(t *testing.T) {
	r, _ := setupTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/v1/places/node:123?include_nearby=true", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp PlaceDetailsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode place response: %v", err)
	}
	if resp.NameDisplay != "Atomium" {
		t.Fatalf("expected Atomium, got %s", resp.NameDisplay)
	}
}

func TestAirportsHandler(t *testing.T) {
	r, _ := setupTestRouter()

	req := httptest.NewRequest(http.MethodGet, "/v1/airports?q=brussels", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}

	var resp AirportsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to decode airports response: %v", err)
	}
	if len(resp.Results) == 0 {
		t.Fatalf("expected airports result")
	}
}

func TestTaxonomyAndLanguagesHandlers(t *testing.T) {
	r, _ := setupTestRouter()

	// Taxonomy
	reqTax := httptest.NewRequest(http.MethodGet, "/v1/taxonomy", nil)
	recTax := httptest.NewRecorder()
	r.ServeHTTP(recTax, reqTax)
	if recTax.Code != http.StatusOK {
		t.Fatalf("expected 200 for taxonomy, got %d", recTax.Code)
	}

	// Languages GET
	reqLang := httptest.NewRequest(http.MethodGet, "/v1/languages", nil)
	recLang := httptest.NewRecorder()
	r.ServeHTTP(recLang, reqLang)
	if recLang.Code != http.StatusOK {
		t.Fatalf("expected 200 for languages, got %d", recLang.Code)
	}

	// Languages PUT
	putBody := bytes.NewBufferString(`{"code":"th","name":"Thai","enabled":true}`)
	reqPutLang := httptest.NewRequest(http.MethodPut, "/v1/languages", putBody)
	reqPutLang.Header.Set("Content-Type", "application/json")
	recPutLang := httptest.NewRecorder()
	r.ServeHTTP(recPutLang, reqPutLang)
	if recPutLang.Code != http.StatusOK {
		t.Fatalf("expected 200 for languages PUT, got %d", recPutLang.Code)
	}
}

func TestObservabilityHandlers(t *testing.T) {
	r, _ := setupTestRouter()

	// Health
	reqHealth := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
	recHealth := httptest.NewRecorder()
	r.ServeHTTP(recHealth, reqHealth)
	if recHealth.Code != http.StatusOK {
		t.Fatalf("expected 200 for health, got %d", recHealth.Code)
	}

	// Metrics
	reqMetrics := httptest.NewRequest(http.MethodGet, "/v1/metrics", nil)
	recMetrics := httptest.NewRecorder()
	r.ServeHTTP(recMetrics, reqMetrics)
	if recMetrics.Code != http.StatusOK {
		t.Fatalf("expected 200 for metrics, got %d", recMetrics.Code)
	}

	// OpenAPI
	reqOpenAPI := httptest.NewRequest(http.MethodGet, "/v1/openapi.json", nil)
	recOpenAPI := httptest.NewRecorder()
	r.ServeHTTP(recOpenAPI, reqOpenAPI)
	if recOpenAPI.Code != http.StatusOK {
		t.Fatalf("expected 200 for openapi, got %d", recOpenAPI.Code)
	}
}

func TestImporterProxyHandlers(t *testing.T) {
	r, _ := setupTestRouter()

	// Status
	reqStatus := httptest.NewRequest(http.MethodGet, "/v1/importer/status", nil)
	recStatus := httptest.NewRecorder()
	r.ServeHTTP(recStatus, reqStatus)
	if recStatus.Code != http.StatusOK {
		t.Fatalf("expected 200 for importer status, got %d", recStatus.Code)
	}

	// Countries
	reqCountries := httptest.NewRequest(http.MethodGet, "/v1/importer/countries", nil)
	recCountries := httptest.NewRecorder()
	r.ServeHTTP(recCountries, reqCountries)
	if recCountries.Code != http.StatusOK {
		t.Fatalf("expected 200 for importer countries, got %d", recCountries.Code)
	}

	// Jobs POST
	jobBody := bytes.NewBufferString(`{"countries":["BE"],"online_enrich":false}`)
	reqJobs := httptest.NewRequest(http.MethodPost, "/v1/importer/jobs", jobBody)
	reqJobs.Header.Set("Content-Type", "application/json")
	recJobs := httptest.NewRecorder()
	r.ServeHTTP(recJobs, reqJobs)
	if recJobs.Code != http.StatusAccepted {
		t.Fatalf("expected 202 Accepted for importer job, got %d: %s", recJobs.Code, recJobs.Body.String())
	}

	// Queue GET
	reqQueue := httptest.NewRequest(http.MethodGet, "/v1/importer/queue", nil)
	recQueue := httptest.NewRecorder()
	r.ServeHTTP(recQueue, reqQueue)
	if recQueue.Code != http.StatusOK {
		t.Fatalf("expected 200 for importer queue, got %d", recQueue.Code)
	}

	// Queue Dequeue POST
	reqDeq := httptest.NewRequest(http.MethodPost, "/v1/importer/queue/dequeue?country=NL", nil)
	recDeq := httptest.NewRecorder()
	r.ServeHTTP(recDeq, reqDeq)
	if recDeq.Code != http.StatusOK {
		t.Fatalf("expected 200 for dequeue, got %d", recDeq.Code)
	}

	// Jobs Cancel POST
	cancelBody := bytes.NewBufferString(`{"worker_id": 1}`)
	reqCancel := httptest.NewRequest(http.MethodPost, "/v1/importer/cancel", cancelBody)
	reqCancel.Header.Set("Content-Type", "application/json")
	recCancel := httptest.NewRecorder()
	r.ServeHTTP(recCancel, reqCancel)
	if recCancel.Code != http.StatusOK {
		t.Fatalf("expected 200 for cancel, got %d", recCancel.Code)
	}

	// Queue Clear DELETE
	reqClear := httptest.NewRequest(http.MethodDelete, "/v1/importer/queue", nil)
	recClear := httptest.NewRecorder()
	r.ServeHTTP(recClear, reqClear)
	if recClear.Code != http.StatusOK {
		t.Fatalf("expected 200 for clear queue, got %d", recClear.Code)
	}
}

func TestValidationAndEdgeCases(t *testing.T) {
	r, _ := setupTestRouter()

	// 1. Reverse geocoding with invalid lat/lng (missing params)
	reqRevBad := httptest.NewRequest(http.MethodGet, "/v1/reverse", nil)
	recRevBad := httptest.NewRecorder()
	r.ServeHTTP(recRevBad, reqRevBad)
	if recRevBad.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing reverse params, got %d", recRevBad.Code)
	}

	// 2. Reverse geocoding with out-of-range coordinates
	reqRevRange := httptest.NewRequest(http.MethodGet, "/v1/reverse?lat=150.0&lng=200.0", nil)
	recRevRange := httptest.NewRecorder()
	r.ServeHTTP(recRevRange, reqRevRange)
	if recRevRange.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for out of range lat/lng, got %d", recRevRange.Code)
	}

	// 3. Nearby with invalid coordinates
	reqNearBad := httptest.NewRequest(http.MethodGet, "/v1/nearby?lat=abc&lng=def", nil)
	recNearBad := httptest.NewRecorder()
	r.ServeHTTP(recNearBad, reqNearBad)
	if recNearBad.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid nearby lat/lng, got %d", recNearBad.Code)
	}

	// 4. BBox with invalid bbox format
	reqBBoxBad := httptest.NewRequest(http.MethodGet, "/v1/bbox?bbox=invalid", nil)
	recBBoxBad := httptest.NewRecorder()
	r.ServeHTTP(recBBoxBad, reqBBoxBad)
	if recBBoxBad.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid bbox, got %d", recBBoxBad.Code)
	}

	// 5. Place lookup not found (404)
	reqPlace404 := httptest.NewRequest(http.MethodGet, "/v1/places/node:999999", nil)
	recPlace404 := httptest.NewRecorder()
	r.ServeHTTP(recPlace404, reqPlace404)
	if recPlace404.Code != http.StatusNotFound {
		t.Errorf("expected 404 for unknown place, got %d", recPlace404.Code)
	}

	// 6. Wikidata lookup by QID
	reqWiki := httptest.NewRequest(http.MethodGet, "/v1/places/wikidata/Q123", nil)
	recWiki := httptest.NewRecorder()
	r.ServeHTTP(recWiki, reqWiki)
	if recWiki.Code != http.StatusOK {
		t.Errorf("expected 200 for wikidata qid, got %d", recWiki.Code)
	}

	// 7. Airports lookup by code
	reqAirportCode := httptest.NewRequest(http.MethodGet, "/v1/airports/BRU", nil)
	recAirportCode := httptest.NewRecorder()
	r.ServeHTTP(recAirportCode, reqAirportCode)
	if recAirportCode.Code != http.StatusOK {
		t.Errorf("expected 200 for airport code BRU, got %d", recAirportCode.Code)
	}

	// 8. Admin hierarchy navigation
	reqAdminCountries := httptest.NewRequest(http.MethodGet, "/v1/countries", nil)
	recAdminCountries := httptest.NewRecorder()
	r.ServeHTTP(recAdminCountries, reqAdminCountries)
	if recAdminCountries.Code != http.StatusOK {
		t.Errorf("expected 200 for /v1/countries, got %d", recAdminCountries.Code)
	}

	reqAdminRegions := httptest.NewRequest(http.MethodGet, "/v1/countries/BE/regions", nil)
	recAdminRegions := httptest.NewRecorder()
	r.ServeHTTP(recAdminRegions, reqAdminRegions)
	if recAdminRegions.Code != http.StatusOK {
		t.Errorf("expected 200 for /v1/countries/BE/regions, got %d", recAdminRegions.Code)
	}

	reqAdminCities := httptest.NewRequest(http.MethodGet, "/v1/countries/BE/cities", nil)
	recAdminCities := httptest.NewRecorder()
	r.ServeHTTP(recAdminCities, reqAdminCities)
	if recAdminCities.Code != http.StatusOK {
		t.Errorf("expected 200 for /v1/countries/BE/cities, got %d", recAdminCities.Code)
	}
}
