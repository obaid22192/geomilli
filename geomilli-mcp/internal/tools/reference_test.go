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

type mockReferenceTransport func(req *http.Request) (*http.Response, error)

func (m mockReferenceTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return m(req)
}

func setupReferenceTest(t *testing.T, handler http.HandlerFunc) (*mcp.ClientSession, func()) {
	client := apiclient.NewClient("http://fake-api", "", 5*time.Second, &http.Client{
		Transport: mockReferenceTransport(func(req *http.Request) (*http.Response, error) {
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
	registerReference(server, deps)

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

func TestListPlaceCategories_Success(t *testing.T) {
	clientSession, cleanup := setupReferenceTest(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/taxonomy" {
			http.NotFound(w, r)
			return
		}

		resp := `{
			"version": 1,
			"classes": [
				{
					"name": "Restaurant",
					"base_score": 0.7,
					"layer": "poi",
					"match": [{"key": "amenity", "values": ["restaurant", "bistro"]}],
					"attributes": []
				},
				{
					"name": "Airport",
					"base_score": 0.9,
					"layer": "poi",
					"match": [{"key": "aeroway", "values": ["aerodrome", "terminal"]}],
					"attributes": []
				},
				{
					"name": "City",
					"base_score": 0.8,
					"layer": "admin",
					"match": [{"key": "place", "values": ["city", "town"]}],
					"attributes": []
				}
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
	var catTool *mcp.Tool
	for _, tool := range toolsList.Tools {
		if tool.Name == "list_place_categories" {
			catTool = tool
			break
		}
	}
	if catTool == nil {
		t.Fatalf("tool list_place_categories not found")
	}
	if catTool.Annotations == nil || !catTool.Annotations.ReadOnlyHint || catTool.Annotations.OpenWorldHint == nil || *catTool.Annotations.OpenWorldHint {
		t.Errorf("expected ReadOnly annotation, got %+v", catTool.Annotations)
	}

	callRes, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      "list_place_categories",
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
	var out ListPlaceCategoriesOutput
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("failed to unmarshal output: %v", err)
	}

	if out.TotalClasses != 3 || len(out.Categories) != 3 {
		t.Fatalf("unexpected output: %+v", out)
	}

	// Should be sorted alphabetically: Airport, City, Restaurant
	if out.Categories[0].Class != "Airport" || out.Categories[0].Layer != "poi" || out.Categories[0].BaseScore != 0.9 {
		t.Errorf("unexpected category 0: %+v", out.Categories[0])
	}
	if len(out.Categories[0].Subclasses) != 2 || out.Categories[0].Subclasses[0] != "aerodrome" {
		t.Errorf("unexpected subclasses for Airport: %v", out.Categories[0].Subclasses)
	}

	if out.Categories[1].Class != "City" || out.Categories[1].Layer != "admin" {
		t.Errorf("unexpected category 1: %+v", out.Categories[1])
	}

	if out.Categories[2].Class != "Restaurant" {
		t.Errorf("unexpected category 2: %+v", out.Categories[2])
	}

	text := callRes.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(text, "Taxonomy contains 3 canonical place categories") {
		t.Errorf("summary does not contain category count: %s", text)
	}
}

func TestListPlaceCategories_UpstreamError(t *testing.T) {
	clientSession, cleanup := setupReferenceTest(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`{"error": {"code": "internal_error", "message": "cannot load taxonomy.yaml", "status": 500}}`))
	})
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      "list_place_categories",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("unexpected protocol error: %v", err)
	}
	if !res.IsError {
		t.Fatalf("expected IsError=true on upstream failure")
	}
	text := res.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(text, "failed to get taxonomy") {
		t.Errorf("expected actionable error message, got %q", text)
	}
}

func TestMeasureDistance_CoordinatesSuccess(t *testing.T) {
	clientSession, cleanup := setupReferenceTest(t, func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
	})
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Verify tool annotations
	toolsList, err := clientSession.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("failed to list tools: %v", err)
	}
	var distTool *mcp.Tool
	for _, tool := range toolsList.Tools {
		if tool.Name == "measure_distance" {
			distTool = tool
			break
		}
	}
	if distTool == nil {
		t.Fatalf("tool measure_distance not found")
	}
	if distTool.Annotations == nil || !distTool.Annotations.ReadOnlyHint || distTool.Annotations.OpenWorldHint == nil || *distTool.Annotations.OpenWorldHint {
		t.Errorf("expected ReadOnly annotation, got %+v", distTool.Annotations)
	}

	// Brussels Grand Place (50.8467, 4.3525) to Brussels Airport (50.9010, 4.4844)
	callRes, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "measure_distance",
		Arguments: map[string]any{
			"from_lat": 50.8467,
			"from_lng": 4.3525,
			"to_lat":   50.9010,
			"to_lng":   4.4844,
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
	var out MeasureDistanceOutput
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("failed to unmarshal output: %v", err)
	}

	// Great-circle distance is approx 11.05 km (11050.4 meters)
	if out.DistanceMeters < 11000 || out.DistanceMeters > 11100 {
		t.Errorf("unexpected distance meters: %f", out.DistanceMeters)
	}
	if out.DistanceKm < 11.0 || out.DistanceKm > 11.1 {
		t.Errorf("unexpected distance km: %f", out.DistanceKm)
	}
	if out.DistanceMiles < 6.8 || out.DistanceMiles > 7.0 {
		t.Errorf("unexpected distance miles: %f", out.DistanceMiles)
	}

	text := callRes.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(text, "km") || !strings.Contains(text, "miles") {
		t.Errorf("summary does not contain distance units: %s", text)
	}
}

func TestMeasureDistance_PlaceIDsSuccess(t *testing.T) {
	clientSession, cleanup := setupReferenceTest(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/v1/places/osm_grand_place":
			_, _ = w.Write([]byte(`{
				"id": "osm_grand_place",
				"name_display": "Grand Place",
				"class": "Attraction",
				"_geo": {"lat": 50.8467, "lng": 4.3525}
			}`))
		case "/v1/places/osm_bru_airport":
			_, _ = w.Write([]byte(`{
				"id": "osm_bru_airport",
				"name_display": "Brussels Airport",
				"class": "Airport",
				"_geo": {"lat": 50.9010, "lng": 4.4844}
			}`))
		default:
			http.NotFound(w, r)
		}
	})
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	callRes, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "measure_distance",
		Arguments: map[string]any{
			"from_place_id": "osm_grand_place",
			"to_place_id":   "osm_bru_airport",
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
	var out MeasureDistanceOutput
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("failed to unmarshal output: %v", err)
	}

	if out.From.PlaceID != "osm_grand_place" || out.From.Name != "Grand Place" {
		t.Errorf("unexpected from location: %+v", out.From)
	}
	if out.To.PlaceID != "osm_bru_airport" || out.To.Name != "Brussels Airport" {
		t.Errorf("unexpected to location: %+v", out.To)
	}
	if out.DistanceKm < 11.0 || out.DistanceKm > 11.1 {
		t.Errorf("unexpected distance km: %f", out.DistanceKm)
	}

	text := callRes.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(text, "Grand Place") || !strings.Contains(text, "Brussels Airport") {
		t.Errorf("summary does not mention place names: %s", text)
	}
}

func TestMeasureDistance_MixedSuccess(t *testing.T) {
	clientSession, cleanup := setupReferenceTest(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/places/osm_grand_place" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{
				"id": "osm_grand_place",
				"name_display": "Grand Place",
				"class": "Attraction",
				"_geo": {"lat": 50.8467, "lng": 4.3525}
			}`))
			return
		}
		http.NotFound(w, r)
	})
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	callRes, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "measure_distance",
		Arguments: map[string]any{
			"from_place_id": "osm_grand_place",
			"to_lat":        50.9010,
			"to_lng":        4.4844,
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
	var out MeasureDistanceOutput
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("failed to unmarshal output: %v", err)
	}

	if out.From.Name != "Grand Place" || out.To.Point.Lat != 50.9010 {
		t.Errorf("unexpected endpoints: %+v -> %+v", out.From, out.To)
	}
	if out.DistanceKm < 11.0 || out.DistanceKm > 11.1 {
		t.Errorf("unexpected distance km: %f", out.DistanceKm)
	}
}

func TestMeasureDistance_ValidationErrors(t *testing.T) {
	clientSession, cleanup := setupReferenceTest(t, func(w http.ResponseWriter, r *http.Request) {
		http.NotFound(w, r)
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
			name:      "missing from point",
			args:      map[string]any{"to_lat": 50.0, "to_lng": 4.0},
			errSubstr: "Starting location requires either 'from_place_id' or both 'from_lat' and 'from_lng'",
		},
		{
			name:      "missing to point",
			args:      map[string]any{"from_lat": 50.0, "from_lng": 4.0},
			errSubstr: "Destination location requires either 'to_place_id' or both 'to_lat' and 'to_lng'",
		},
		{
			name:      "from lat out of range",
			args:      map[string]any{"from_lat": 95.0, "from_lng": 4.0, "to_lat": 50.0, "to_lng": 4.0},
			errSubstr: "Starting latitude must be between -90 and 90",
		},
		{
			name:      "from lng out of range",
			args:      map[string]any{"from_lat": 50.0, "from_lng": -200.0, "to_lat": 50.0, "to_lng": 4.0},
			errSubstr: "Starting longitude must be between -180 and 180",
		},
		{
			name:      "to lat out of range",
			args:      map[string]any{"from_lat": 50.0, "from_lng": 4.0, "to_lat": -95.0, "to_lng": 4.0},
			errSubstr: "Destination latitude must be between -90 and 90",
		},
		{
			name:      "to lng out of range",
			args:      map[string]any{"from_lat": 50.0, "from_lng": 4.0, "to_lat": 50.0, "to_lng": 190.0},
			errSubstr: "Destination longitude must be between -180 and 180",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
				Name:      "measure_distance",
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

func TestMeasureDistance_PlaceNotFound(t *testing.T) {
	clientSession, cleanup := setupReferenceTest(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error": {"code": "not_found", "message": "Place not found for ID: non_existent", "status": 404}}`))
	})
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "measure_distance",
		Arguments: map[string]any{
			"from_place_id": "non_existent",
			"to_lat":        50.0,
			"to_lng":        4.0,
		},
	})
	if err != nil {
		t.Fatalf("unexpected protocol error: %v", err)
	}
	if !res.IsError {
		t.Fatalf("expected IsError=true when place cannot be resolved")
	}
	text := res.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(text, "failed to resolve starting place ID \"non_existent\"") {
		t.Errorf("expected not found resolution message, got %q", text)
	}
}
