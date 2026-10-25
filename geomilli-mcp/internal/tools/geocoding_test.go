package tools

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/geomilli/geomilli-mcp/internal/apiclient"
	"github.com/geomilli/geomilli-mcp/internal/config"
	"github.com/geomilli/geomilli-mcp/internal/taxonomy"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type mockTransport func(req *http.Request) (*http.Response, error)

func (f mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

type geoTestEnv struct {
	clientSession *mcp.ClientSession
	lastRequest   *http.Request
	returnError   bool
}

func setupGeocodingTestEnv(t *testing.T, handler http.HandlerFunc) *geoTestEnv {
	t.Helper()

	env := &geoTestEnv{}
	transport := mockTransport(func(r *http.Request) (*http.Response, error) {
		env.lastRequest = r
		if env.returnError {
			body := `{"error": {"code": "internal_error", "message": "something failed", "status": 500}}`
			return &http.Response{
				StatusCode: http.StatusInternalServerError,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(strings.NewReader(body)),
			}, nil
		}

		rec := httptest.NewRecorder()
		handler(rec, r)
		return rec.Result(), nil
	})

	api := apiclient.NewClient("http://mock-geomilli", "", 5*time.Second, &http.Client{Transport: transport})
	tax := taxonomy.NewCache(api, 10*time.Minute)

	deps := Deps{
		API:      api,
		Taxonomy: tax,
		Config:   &config.Config{DefaultLang: "en"},
		Logger:   slog.New(slog.NewJSONHandler(io.Discard, nil)),
	}

	mcpServer := mcp.NewServer(&mcp.Implementation{Name: "test-server", Version: "1.0.0"}, nil)
	registerGeocoding(mcpServer, deps)

	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	serverSession, err := mcpServer.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatalf("failed to connect server session: %v", err)
	}
	t.Cleanup(func() {
		_ = serverSession.Close()
	})

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1.0.0"}, nil)
	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("failed to connect client session: %v", err)
	}
	t.Cleanup(func() {
		_ = clientSession.Close()
	})

	env.clientSession = clientSession
	return env
}

func TestGeocode_Annotations(t *testing.T) {
	env := setupGeocodingTestEnv(t, func(w http.ResponseWriter, r *http.Request) {})
	ctx := context.Background()

	toolsList, err := env.clientSession.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools failed: %v", err)
	}

	var geocodeTool *mcp.Tool
	for _, tool := range toolsList.Tools {
		if tool.Name == "geocode" {
			geocodeTool = tool
			break
		}
	}

	if geocodeTool == nil {
		t.Fatalf("tool 'geocode' not found")
	}

	if geocodeTool.Annotations == nil {
		t.Fatalf("expected annotations on geocode")
	}
	if !geocodeTool.Annotations.ReadOnlyHint {
		t.Errorf("expected ReadOnlyHint=true")
	}
	if !geocodeTool.Annotations.IdempotentHint {
		t.Errorf("expected IdempotentHint=true")
	}
	if geocodeTool.Annotations.OpenWorldHint == nil || !*geocodeTool.Annotations.OpenWorldHint {
		t.Errorf("expected OpenWorldHint=true")
	}
}

func TestGeocode_Validation(t *testing.T) {
	env := setupGeocodingTestEnv(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"query": "Atomium", "total_hits": 0, "results": []}`))
	})
	ctx := context.Background()

	tests := []struct {
		name        string
		args        map[string]any
		expectError string
	}{
		{
			name:        "missing query",
			args:        map[string]any{"query": "   "},
			expectError: "query parameter is required",
		},
		{
			name:        "invalid country code length",
			args:        map[string]any{"query": "Brussels", "country": "BELGIUM"},
			expectError: "country must be a 2-letter ISO",
		},
		{
			name:        "unknown class",
			args:        map[string]any{"query": "Brussels", "class": "GalacticPort"},
			expectError: "unknown class",
		},
		{
			name:        "lat without lng",
			args:        map[string]any{"query": "Brussels", "lat": 50.85},
			expectError: "both lat and lng must be provided",
		},
		{
			name:        "invalid lat range",
			args:        map[string]any{"query": "Brussels", "lat": 95.0, "lng": 4.35},
			expectError: "lat must be between -90 and 90",
		},
		{
			name:        "invalid lng range",
			args:        map[string]any{"query": "Brussels", "lat": 50.85, "lng": 195.0},
			expectError: "lng must be between -180 and 180",
		},
		{
			name:        "unsupported language",
			args:        map[string]any{"query": "Brussels", "lang": "klingon"},
			expectError: "unsupported language code",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			res, err := env.clientSession.CallTool(ctx, &mcp.CallToolParams{
				Name:      "geocode",
				Arguments: tc.args,
			})
			if err != nil {
				t.Fatalf("unexpected protocol error: %v", err)
			}
			if !res.IsError {
				t.Fatalf("expected tool error for %s, got success: %+v", tc.name, res)
			}
			txt := res.Content[0].(*mcp.TextContent).Text
			if !strings.Contains(txt, tc.expectError) {
				t.Errorf("expected error message to contain %q, got %q", tc.expectError, txt)
			}
		})
	}
}

func TestGeocode_SuccessAndClamping(t *testing.T) {
	var capturedQuery string
	var capturedLimit string
	var capturedCountry string
	var capturedClass string
	var capturedLat string

	env := setupGeocodingTestEnv(t, func(w http.ResponseWriter, r *http.Request) {
		capturedQuery = r.URL.Query().Get("q")
		capturedLimit = r.URL.Query().Get("limit")
		capturedCountry = r.URL.Query().Get("country")
		capturedClass = r.URL.Query().Get("class")
		capturedLat = r.URL.Query().Get("lat")

		resp := `{
			"query": "Atomium",
			"total_hits": 1,
			"results": [
				{
					"id": "osm_n123",
					"class": "TouristAttraction",
					"subclass": "monument",
					"name_display": "Atomium",
					"country_code": "BE",
					"country_name": "Belgium",
					"admin_hierarchy": ["City of Brussels", "Brussels", "Belgium"],
					"_geo": {"lat": 50.8949, "lng": 4.3415},
					"wikidata_id": "Q207",
					"popularity_score": 0.95
				}
			]
		}`
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(resp))
	})
	ctx := context.Background()

	// Call with limit 50 (should clamp to 10), country "be" (should uppercase), class "touristattraction" (should canonicalize)
	res, err := env.clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "geocode",
		Arguments: map[string]any{
			"query":   "Atomium",
			"country": "be",
			"class":   "touristattraction",
			"lat":     50.89,
			"lng":     4.34,
			"limit":   50,
		},
	})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected tool error: %s", res.Content[0].(*mcp.TextContent).Text)
	}

	if capturedQuery != "Atomium" {
		t.Errorf("expected query 'Atomium', got %q", capturedQuery)
	}
	if capturedLimit != "10" {
		t.Errorf("expected limit clamped to '10', got %q", capturedLimit)
	}
	if capturedCountry != "BE" {
		t.Errorf("expected country 'BE', got %q", capturedCountry)
	}
	if capturedClass != "TouristAttraction" {
		t.Errorf("expected canonical class 'TouristAttraction', got %q", capturedClass)
	}
	if capturedLat != "50.89" {
		t.Errorf("expected lat '50.89', got %q", capturedLat)
	}

	// Verify text summary contains Atomium
	txt := res.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(txt, "Atomium") {
		t.Errorf("expected text summary to contain 'Atomium', got %q", txt)
	}

	// Verify structured content
	structured, err := json.Marshal(res.StructuredContent)
	if err != nil {
		t.Fatalf("marshaling StructuredContent: %v", err)
	}
	var out GeocodeOutput
	if err := json.Unmarshal(structured, &out); err != nil {
		t.Fatalf("unmarshaling GeocodeOutput: %v", err)
	}
	if out.TotalHits != 1 || len(out.Places) != 1 {
		t.Fatalf("expected 1 place, got %+v", out)
	}
	if out.Places[0].Name != "Atomium" || out.Places[0].Class != "TouristAttraction" {
		t.Errorf("unexpected place data: %+v", out.Places[0])
	}
}

func TestGeocode_APIError(t *testing.T) {
	env := setupGeocodingTestEnv(t, func(w http.ResponseWriter, r *http.Request) {})
	env.returnError = true
	ctx := context.Background()

	res, err := env.clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      "geocode",
		Arguments: map[string]any{"query": "Brussels"},
	})
	if err != nil {
		t.Fatalf("unexpected protocol error: %v", err)
	}
	if !res.IsError {
		t.Fatalf("expected tool error on 500, got success")
	}
	txt := res.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(txt, "failed to geocode") {
		t.Errorf("expected failure message, got %q", txt)
	}
}

func TestReverseGeocode_Annotations(t *testing.T) {
	env := setupGeocodingTestEnv(t, func(w http.ResponseWriter, r *http.Request) {})
	ctx := context.Background()

	toolsList, err := env.clientSession.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools failed: %v", err)
	}

	var revTool *mcp.Tool
	for _, tool := range toolsList.Tools {
		if tool.Name == "reverse_geocode" {
			revTool = tool
			break
		}
	}

	if revTool == nil {
		t.Fatalf("tool 'reverse_geocode' not found")
	}
	if revTool.Annotations == nil {
		t.Fatalf("expected annotations on reverse_geocode")
	}
	if !revTool.Annotations.ReadOnlyHint {
		t.Errorf("expected ReadOnlyHint=true")
	}
	if !revTool.Annotations.IdempotentHint {
		t.Errorf("expected IdempotentHint=true")
	}
	if revTool.Annotations.OpenWorldHint == nil || *revTool.Annotations.OpenWorldHint != false {
		t.Errorf("expected OpenWorldHint=false for reverse_geocode")
	}
}

func TestReverseGeocode_ValidationAndClamping(t *testing.T) {
	var capturedRadius string

	env := setupGeocodingTestEnv(t, func(w http.ResponseWriter, r *http.Request) {
		capturedRadius = r.URL.Query().Get("radius_meters")
		resp := `{
			"lat": 50.85,
			"lng": 4.35,
			"radius_meters": 50000,
			"address": {
				"city": "Brussels",
				"region": "Brussels-Capital",
				"country": "Belgium",
				"country_code": "BE"
			},
			"admin_hierarchy": ["Brussels", "Belgium"],
			"closest_place": {
				"id": "osm_r123",
				"class": "City",
				"name_display": "Brussels",
				"country_code": "BE",
				"country_name": "Belgium",
				"_geo": {"lat": 50.8503, "lng": 4.3517}
			},
			"nearest_places": []
		}`
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(resp))
	})
	ctx := context.Background()

	// 1. Validation error: invalid lat
	res, err := env.clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "reverse_geocode",
		Arguments: map[string]any{
			"lat": 95.0,
			"lng": 4.35,
		},
	})
	if err != nil {
		t.Fatalf("unexpected protocol error: %v", err)
	}
	if !res.IsError {
		t.Fatalf("expected tool error for lat > 90")
	}

	// 2. Validation error: invalid lng
	res, err = env.clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "reverse_geocode",
		Arguments: map[string]any{
			"lat": 50.85,
			"lng": -190.0,
		},
	})
	if err != nil {
		t.Fatalf("unexpected protocol error: %v", err)
	}
	if !res.IsError {
		t.Fatalf("expected tool error for lng < -180")
	}

	// 3. Radius default clamping: radius 0 -> 50000
	res, err = env.clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "reverse_geocode",
		Arguments: map[string]any{
			"lat": 50.85,
			"lng": 4.35,
		},
	})
	if err != nil || res.IsError {
		t.Fatalf("CallTool failed: err=%v, res=%+v", err, res)
	}
	if capturedRadius != "50000" {
		t.Errorf("expected default radius 50000, got %q", capturedRadius)
	}

	// 4. Radius max clamping: radius 200000 -> 100000
	res, err = env.clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "reverse_geocode",
		Arguments: map[string]any{
			"lat":           50.85,
			"lng":           4.35,
			"radius_meters": 200000,
		},
	})
	if err != nil || res.IsError {
		t.Fatalf("CallTool failed: err=%v, res=%+v", err, res)
	}
	if capturedRadius != "100000" {
		t.Errorf("expected clamped radius 100000, got %q", capturedRadius)
	}

	// Verify structured output
	structured, err := json.Marshal(res.StructuredContent)
	if err != nil {
		t.Fatalf("marshaling StructuredContent: %v", err)
	}
	var out ReverseGeocodeOutput
	if err := json.Unmarshal(structured, &out); err != nil {
		t.Fatalf("unmarshaling ReverseGeocodeOutput: %v", err)
	}
	if out.Address.City != "Brussels" || out.Address.Country != "Belgium" {
		t.Errorf("unexpected address: %+v", out.Address)
	}
	if out.ClosestPlace == nil || out.ClosestPlace.Name != "Brussels" {
		t.Errorf("unexpected closest place: %+v", out.ClosestPlace)
	}

	// Verify text summary
	txt := res.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(txt, "Brussels") || !strings.Contains(txt, "Belgium") {
		t.Errorf("unexpected summary text: %q", txt)
	}
}
