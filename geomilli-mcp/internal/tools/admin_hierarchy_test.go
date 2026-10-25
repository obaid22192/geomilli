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

type mockAdminTransport func(req *http.Request) (*http.Response, error)

func (m mockAdminTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return m(req)
}

func setupAdminHierarchyTest(t *testing.T, handler http.HandlerFunc) (*mcp.ClientSession, func()) {
	client := apiclient.NewClient("http://fake-api", "", 5*time.Second, &http.Client{
		Transport: mockAdminTransport(func(req *http.Request) (*http.Response, error) {
			rec := httptest.NewRecorder()
			handler(rec, req)
			return rec.Result(), nil
		}),
	})
	tax := taxonomy.NewCache(client, 10*time.Minute)

	deps := Deps{
		API:      client,
		Taxonomy: tax,
		Config:   &config.Config{},
		Logger:   slog.New(slog.NewJSONHandler(io.Discard, nil)),
	}

	server := mcp.NewServer(&mcp.Implementation{Name: "geomilli-test", Version: "1.0.0"}, nil)
	registerAdminHierarchy(server, deps)

	serverTransport, clientTransport := mcp.NewInMemoryTransports()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatalf("failed to connect server session: %v", err)
	}

	mcpClient := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1.0.0"}, nil)
	clientSession, err := mcpClient.Connect(ctx, clientTransport, nil)
	if err != nil {
		serverSession.Close()
		t.Fatalf("failed to connect client session: %v", err)
	}

	cleanup := func() {
		clientSession.Close()
		serverSession.Close()
	}

	return clientSession, cleanup
}

func TestListCountries_Success(t *testing.T) {
	clientSession, cleanup := setupAdminHierarchyTest(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/countries" {
			http.NotFound(w, r)
			return
		}

		resp := `{
			"total_countries": 2,
			"countries": [
				{"country_code": "BE", "country_name": "Belgium", "place_count": 100},
				{"country_code": "NL", "country_name": "Netherlands", "place_count": 80}
			]
		}`
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(resp))
	})
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	callRes, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      "list_countries",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("unexpected call error: %v", err)
	}
	if callRes.IsError {
		t.Fatalf("call returned tool error: %+v", callRes)
	}

	raw, err := json.Marshal(callRes.StructuredContent)
	if err != nil {
		t.Fatalf("failed to marshal StructuredContent: %v", err)
	}
	var out ListCountriesOutput
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("failed to unmarshal output: %v", err)
	}

	if out.TotalCountries != 2 || len(out.Countries) != 2 {
		t.Errorf("unexpected output: %+v", out)
	}
}

func TestListRegions_Success(t *testing.T) {
	clientSession, cleanup := setupAdminHierarchyTest(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/countries/BE/regions" {
			http.NotFound(w, r)
			return
		}

		resp := `{
			"country_code": "BE",
			"total_hits": 3,
			"regions": [
				{"id": "osm_r1", "name_display": "Flemish Region", "class": "Region", "country_code": "BE", "_geo": {"lat": 51.0, "lng": 4.0}},
				{"id": "osm_r2", "name_display": "Walloon Region", "class": "Region", "country_code": "BE", "_geo": {"lat": 50.4, "lng": 4.5}},
				{"id": "osm_r3", "name_display": "Brussels-Capital", "class": "Region", "country_code": "BE", "_geo": {"lat": 50.85, "lng": 4.35}}
			]
		}`
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(resp))
	})
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Verify tool annotations
	toolsList, err := clientSession.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("failed to list tools: %v", err)
	}
	var regionTool *mcp.Tool
	for _, tool := range toolsList.Tools {
		if tool.Name == "list_regions" {
			regionTool = tool
			break
		}
	}
	if regionTool == nil {
		t.Fatalf("tool list_regions not found")
	}
	if regionTool.Annotations == nil || !regionTool.Annotations.ReadOnlyHint || regionTool.Annotations.OpenWorldHint == nil || *regionTool.Annotations.OpenWorldHint {
		t.Errorf("expected ReadOnly annotation, got %+v", regionTool.Annotations)
	}

	callRes, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "list_regions",
		Arguments: map[string]any{
			"country_code": "be",
		},
	})
	if err != nil {
		t.Fatalf("unexpected call error: %v", err)
	}
	if callRes.IsError {
		t.Fatalf("call returned tool error: %+v", callRes)
	}

	raw, err := json.Marshal(callRes.StructuredContent)
	if err != nil {
		t.Fatalf("failed to marshal StructuredContent: %v", err)
	}
	var out ListRegionsOutput
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("failed to unmarshal output: %v", err)
	}

	if out.CountryCode != "BE" || out.TotalHits != 3 || len(out.Regions) != 3 {
		t.Fatalf("unexpected output: %+v", out)
	}
	if out.Regions[0].Name != "Flemish Region" {
		t.Errorf("expected first region 'Flemish Region', got %q", out.Regions[0].Name)
	}

	text := callRes.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(text, "Found 3 region(s) in country BE.") {
		t.Errorf("expected summary text, got %q", text)
	}
}

func TestListRegions_ValidationErrors(t *testing.T) {
	clientSession, cleanup := setupAdminHierarchyTest(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"country_code":"BE","total_hits":0,"regions":[]}`))
	})
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	testCases := []struct {
		name      string
		args      map[string]any
		errSubstr string
	}{
		{
			name:      "empty country code",
			args:      map[string]any{"country_code": ""},
			errSubstr: "Country code is required",
		},
		{
			name:      "country code too long",
			args:      map[string]any{"country_code": "BEL"},
			errSubstr: "Country code must be a 2-letter ISO",
		},
		{
			name:      "unsupported language",
			args:      map[string]any{"country_code": "BE", "lang": "badlang"},
			errSubstr: "Unsupported language code",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
				Name:      "list_regions",
				Arguments: tc.args,
			})
			if err != nil {
				t.Fatalf("unexpected protocol error: %v", err)
			}
			if !res.IsError {
				t.Fatalf("expected IsError=true, got false")
			}
			text := res.Content[0].(*mcp.TextContent).Text
			if !strings.Contains(text, tc.errSubstr) {
				t.Errorf("expected error containing %q, got %q", tc.errSubstr, text)
			}
		})
	}
}

func TestListRegions_UpstreamError(t *testing.T) {
	clientSession, cleanup := setupAdminHierarchyTest(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error": {"code": "internal_error", "message": "database disk failure", "status": 500}}`))
	})
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      "list_regions",
		Arguments: map[string]any{"country_code": "BE"},
	})
	if err != nil {
		t.Fatalf("unexpected protocol error: %v", err)
	}
	if !res.IsError {
		t.Fatalf("expected IsError=true on upstream failure")
	}
	text := res.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(text, "failed to list regions for country \"BE\"") {
		t.Errorf("expected actionable error message, got %q", text)
	}
}

func TestListCities_Success(t *testing.T) {
	var requestedLimit string
	clientSession, cleanup := setupAdminHierarchyTest(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/countries/BE/cities" {
			http.NotFound(w, r)
			return
		}
		requestedLimit = r.URL.Query().Get("limit")

		resp := `{
			"country_code": "BE",
			"limit": 2,
			"offset": 0,
			"total_hits": 2,
			"cities": [
				{"id": "osm_c1", "name_display": "Brussels", "class": "City", "country_code": "BE", "population": 1200000, "_geo": {"lat": 50.85, "lng": 4.35}},
				{"id": "osm_c2", "name_display": "Antwerp", "class": "City", "country_code": "BE", "population": 520000, "_geo": {"lat": 51.22, "lng": 4.40}}
			]
		}`
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(resp))
	})
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Verify tool annotations
	toolsList, err := clientSession.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("failed to list tools: %v", err)
	}
	var cityTool *mcp.Tool
	for _, tool := range toolsList.Tools {
		if tool.Name == "list_cities" {
			cityTool = tool
			break
		}
	}
	if cityTool == nil {
		t.Fatalf("tool list_cities not found")
	}
	if cityTool.Annotations == nil || !cityTool.Annotations.ReadOnlyHint || cityTool.Annotations.OpenWorldHint == nil || *cityTool.Annotations.OpenWorldHint {
		t.Errorf("expected ReadOnly annotation, got %+v", cityTool.Annotations)
	}

	callRes, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "list_cities",
		Arguments: map[string]any{
			"country_code": "be",
			"limit":        2,
			"offset":       0,
		},
	})
	if err != nil {
		t.Fatalf("unexpected call error: %v", err)
	}
	if callRes.IsError {
		t.Fatalf("call returned tool error: %+v", callRes)
	}

	if requestedLimit != "2" {
		t.Errorf("expected requested limit=2, got %s", requestedLimit)
	}

	raw, err := json.Marshal(callRes.StructuredContent)
	if err != nil {
		t.Fatalf("failed to marshal StructuredContent: %v", err)
	}
	var out ListCitiesOutput
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("failed to unmarshal output: %v", err)
	}

	if out.CountryCode != "BE" || out.TotalHits != 2 || len(out.Cities) != 2 {
		t.Fatalf("unexpected output: %+v", out)
	}
	if out.Cities[0].Population != 1200000 || out.Cities[0].Name != "Brussels" {
		t.Errorf("unexpected first city: %+v", out.Cities[0])
	}

	text := callRes.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(text, "Brussels (pop: 1200000)") {
		t.Errorf("summary does not contain city population info: %s", text)
	}
}

func TestListCities_LimitClampingAndDefaults(t *testing.T) {
	var capturedLimit string
	clientSession, cleanup := setupAdminHierarchyTest(t, func(w http.ResponseWriter, r *http.Request) {
		capturedLimit = r.URL.Query().Get("limit")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"country_code":"BE","limit":20,"offset":0,"total_hits":0,"cities":[]}`))
	})
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 1. Default when limit <= 0 -> 20
	_, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      "list_cities",
		Arguments: map[string]any{"country_code": "BE", "limit": 0},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedLimit != "20" {
		t.Errorf("expected default limit=20, got %s", capturedLimit)
	}

	// 2. Clamped when limit > 100 -> 100
	_, err = clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      "list_cities",
		Arguments: map[string]any{"country_code": "BE", "limit": 500},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if capturedLimit != "100" {
		t.Errorf("expected clamped limit=100, got %s", capturedLimit)
	}
}

func TestListCities_ValidationErrors(t *testing.T) {
	clientSession, cleanup := setupAdminHierarchyTest(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"country_code":"BE","total_hits":0,"cities":[]}`))
	})
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	testCases := []struct {
		name      string
		args      map[string]any
		errSubstr string
	}{
		{
			name:      "empty country code",
			args:      map[string]any{"country_code": ""},
			errSubstr: "Country code is required",
		},
		{
			name:      "country code too long",
			args:      map[string]any{"country_code": "BELGIUM"},
			errSubstr: "Country code must be a 2-letter ISO",
		},
		{
			name:      "unsupported language",
			args:      map[string]any{"country_code": "BE", "lang": "badlang"},
			errSubstr: "Unsupported language code",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
				Name:      "list_cities",
				Arguments: tc.args,
			})
			if err != nil {
				t.Fatalf("unexpected protocol error: %v", err)
			}
			if !res.IsError {
				t.Fatalf("expected IsError=true, got false")
			}
			text := res.Content[0].(*mcp.TextContent).Text
			if !strings.Contains(text, tc.errSubstr) {
				t.Errorf("expected error containing %q, got %q", tc.errSubstr, text)
			}
		})
	}
}

func TestListCities_UpstreamError(t *testing.T) {
	clientSession, cleanup := setupAdminHierarchyTest(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"error": {"code": "upstream_unavailable", "message": "meilisearch connection error", "status": 502}}`))
	})
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      "list_cities",
		Arguments: map[string]any{"country_code": "BE"},
	})
	if err != nil {
		t.Fatalf("unexpected protocol error: %v", err)
	}
	if !res.IsError {
		t.Fatalf("expected IsError=true on upstream failure")
	}
	text := res.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(text, "failed to list cities for country \"BE\"") {
		t.Errorf("expected actionable error message, got %q", text)
	}
}
