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

type discoveryTestEnv struct {
	clientSession *mcp.ClientSession
	lastRequest   *http.Request
	returnError   bool
	errorCode     int
}

func setupDiscoveryTestEnv(t *testing.T, handler http.HandlerFunc) *discoveryTestEnv {
	t.Helper()

	env := &discoveryTestEnv{}
	transport := mockTransport(func(r *http.Request) (*http.Response, error) {
		env.lastRequest = r
		if env.returnError {
			status := http.StatusInternalServerError
			if env.errorCode > 0 {
				status = env.errorCode
			}
			body := `{"error": {"code": "api_error", "message": "operation failed", "status": ` + string(rune(status)) + `}}`
			if status == http.StatusNotFound {
				body = `{"error": {"code": "not_found", "message": "place not found", "status": 404}}`
			}
			return &http.Response{
				StatusCode: status,
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
	registerDiscovery(mcpServer, deps)

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

func TestDiscovery_Annotations(t *testing.T) {
	env := setupDiscoveryTestEnv(t, func(w http.ResponseWriter, r *http.Request) {})
	ctx := context.Background()

	toolsList, err := env.clientSession.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools failed: %v", err)
	}

	toolMap := make(map[string]*mcp.Tool)
	for _, tool := range toolsList.Tools {
		toolMap[tool.Name] = tool
	}

	expectedTools := map[string]struct {
		readOnly  bool
		openWorld bool
	}{
		"search_places":  {readOnly: true, openWorld: true},
		"find_nearby":    {readOnly: true, openWorld: true},
		"places_in_area": {readOnly: true, openWorld: true},
		"get_place":      {readOnly: true, openWorld: false},
	}

	for name, exp := range expectedTools {
		tool, ok := toolMap[name]
		if !ok {
			t.Errorf("tool %q not registered", name)
			continue
		}
		if tool.Annotations == nil {
			t.Errorf("tool %q has nil annotations", name)
			continue
		}
		if tool.Annotations.ReadOnlyHint != exp.readOnly {
			t.Errorf("tool %q: expected ReadOnlyHint=%v, got %v", name, exp.readOnly, tool.Annotations.ReadOnlyHint)
		}
		if tool.Annotations.OpenWorldHint == nil || *tool.Annotations.OpenWorldHint != exp.openWorld {
			t.Errorf("tool %q: expected OpenWorldHint=%v", name, exp.openWorld)
		}
	}
}

func TestSearchPlaces_ValidationAndClamping(t *testing.T) {
	var capturedFilter string
	var capturedLimit string
	var capturedOffset string
	var capturedQuery string

	env := setupDiscoveryTestEnv(t, func(w http.ResponseWriter, r *http.Request) {
		capturedFilter = r.URL.Query().Get("filter")
		capturedLimit = r.URL.Query().Get("limit")
		capturedOffset = r.URL.Query().Get("offset")
		capturedQuery = r.URL.Query().Get("q")

		resp := `{
			"query": "pizza",
			"limit": 10,
			"offset": 0,
			"total_hits": 1,
			"results": [
				{
					"id": "osm_n999",
					"class": "Restaurant",
					"subclass": "pizzeria",
					"name_display": "Chez Gino",
					"country_code": "BE",
					"_geo": {"lat": 50.84, "lng": 4.35},
					"cuisine": "italian"
				}
			]
		}`
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(resp))
	})
	ctx := context.Background()

	// 1. Validation: unknown class
	res, err := env.clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      "search_places",
		Arguments: map[string]any{"class": "AlienBase"},
	})
	if err != nil || !res.IsError {
		t.Fatalf("expected tool error for unknown class, got: err=%v, res=%+v", err, res)
	}

	// 2. Validation: invalid country
	res, err = env.clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      "search_places",
		Arguments: map[string]any{"country": "Atlantis"},
	})
	if err != nil || !res.IsError {
		t.Fatalf("expected tool error for unrecognized country")
	}

	// 3. Validation: star_rating > 5
	res, err = env.clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      "search_places",
		Arguments: map[string]any{"star_rating": 6},
	})
	if err != nil || !res.IsError {
		t.Fatalf("expected tool error for star_rating > 5")
	}

	// 4. Clamping and safe filter construction
	res, err = env.clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "search_places",
		Arguments: map[string]any{
			"query":       "pizza",
			"class":       "restaurant",
			"subclass":    "pizzeria",
			"country":     "be",
			"cuisine":     "italian",
			"star_rating": 4,
			"limit":       100, // should clamp to 25
			"offset":      -5,  // should clamp to 0
		},
	})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected tool error: %s", res.Content[0].(*mcp.TextContent).Text)
	}

	if capturedQuery != "pizza" {
		t.Errorf("expected query 'pizza', got %q", capturedQuery)
	}
	if capturedLimit != "25" {
		t.Errorf("expected limit clamped to 25, got %q", capturedLimit)
	}
	if capturedOffset != "" { // offset 0 is omitted by url.Values
		t.Errorf("expected offset 0 omitted, got %q", capturedOffset)
	}

	expectedFilterParts := []string{
		`class = "Restaurant"`,
		`subclass = "pizzeria"`,
		`country_code = "BE"`,
		`cuisine = "italian"`,
		`star_rating >= 4`,
	}
	for _, part := range expectedFilterParts {
		if !strings.Contains(capturedFilter, part) {
			t.Errorf("expected filter %q to contain %q", capturedFilter, part)
		}
	}

	// Verify structured output
	structured, err := json.Marshal(res.StructuredContent)
	if err != nil {
		t.Fatalf("marshaling StructuredContent: %v", err)
	}
	var out SearchPlacesOutput
	if err := json.Unmarshal(structured, &out); err != nil {
		t.Fatalf("unmarshaling SearchPlacesOutput: %v", err)
	}
	if out.TotalHits != 1 || len(out.Places) != 1 {
		t.Fatalf("expected 1 place in output, got %+v", out)
	}
	if out.Places[0].Name != "Chez Gino" || out.Places[0].Cuisine != "italian" {
		t.Errorf("unexpected place data: %+v", out.Places[0])
	}
}

func TestSearchPlaces_FilterEscaping(t *testing.T) {
	var capturedFilter string

	env := setupDiscoveryTestEnv(t, func(w http.ResponseWriter, r *http.Request) {
		capturedFilter = r.URL.Query().Get("filter")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"query": "", "results": []}`))
	})
	ctx := context.Background()

	res, err := env.clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "search_places",
		Arguments: map[string]any{
			"cuisine": `french "bistro"`,
		},
	})
	if err != nil || res.IsError {
		t.Fatalf("CallTool failed: err=%v, res=%+v", err, res)
	}

	expectedEscaped := `cuisine = "french \"bistro\""`
	if !strings.Contains(capturedFilter, expectedEscaped) {
		t.Errorf("expected escaped filter %q, got %q", expectedEscaped, capturedFilter)
	}
}

func TestFindNearby_ValidationAndExecution(t *testing.T) {
	var capturedRadius string
	var capturedLimit string
	var capturedLat string
	var capturedLng string

	env := setupDiscoveryTestEnv(t, func(w http.ResponseWriter, r *http.Request) {
		capturedRadius = r.URL.Query().Get("radius_meters")
		capturedLimit = r.URL.Query().Get("limit")
		capturedLat = r.URL.Query().Get("lat")
		capturedLng = r.URL.Query().Get("lng")

		resp := `{
			"lat": 50.85,
			"lng": 4.35,
			"radius_meters": 5000,
			"total_hits": 2,
			"results": [
				{
					"id": "osm_n1",
					"class": "Restaurant",
					"name_display": "Grand Place Bistro",
					"_geo": {"lat": 50.8467, "lng": 4.3525},
					"distance_meters": 120.5
				},
				{
					"id": "osm_n2",
					"class": "Hotel",
					"name_display": "Hotel Amigo",
					"_geo": {"lat": 50.8465, "lng": 4.3510},
					"distance_meters": 1500.0
				}
			]
		}`
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(resp))
	})
	ctx := context.Background()

	// 1. Validation: invalid lat
	res, err := env.clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      "find_nearby",
		Arguments: map[string]any{"lat": 95.0, "lng": 4.35},
	})
	if err != nil || !res.IsError {
		t.Fatalf("expected tool error for lat > 90")
	}

	// 2. Validation: invalid min_stars
	res, err = env.clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      "find_nearby",
		Arguments: map[string]any{"lat": 50.85, "lng": 4.35, "min_stars": 6},
	})
	if err != nil || !res.IsError {
		t.Fatalf("expected tool error for min_stars > 5")
	}

	// 3. Clamping: radius 0 -> 5000, limit 50 -> 25
	res, err = env.clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "find_nearby",
		Arguments: map[string]any{
			"lat":   50.85,
			"lng":   4.35,
			"limit": 50,
		},
	})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected tool error: %s", res.Content[0].(*mcp.TextContent).Text)
	}

	if capturedRadius != "5000" {
		t.Errorf("expected default radius 5000, got %q", capturedRadius)
	}
	if capturedLimit != "25" {
		t.Errorf("expected limit clamped to 25, got %q", capturedLimit)
	}
	if capturedLat != "50.85" || capturedLng != "4.35" {
		t.Errorf("unexpected coords: %s, %s", capturedLat, capturedLng)
	}

	// 4. Clamping: radius 100000 -> 50000
	_, _ = env.clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "find_nearby",
		Arguments: map[string]any{
			"lat":           50.85,
			"lng":           4.35,
			"radius_meters": 100000,
		},
	})
	if capturedRadius != "50000" {
		t.Errorf("expected clamped radius 50000, got %q", capturedRadius)
	}

	// Verify structured output has distance_meters
	structured, err := json.Marshal(res.StructuredContent)
	if err != nil {
		t.Fatalf("marshaling StructuredContent: %v", err)
	}
	var out FindNearbyOutput
	if err := json.Unmarshal(structured, &out); err != nil {
		t.Fatalf("unmarshaling FindNearbyOutput: %v", err)
	}
	if len(out.Places) != 2 {
		t.Fatalf("expected 2 places, got %d", len(out.Places))
	}
	if out.Places[0].DistanceMeters == nil || *out.Places[0].DistanceMeters != 120.5 {
		t.Errorf("expected distance 120.5m, got %v", out.Places[0].DistanceMeters)
	}

	// Verify text summary formatting (m vs km)
	txt := res.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(txt, "120 m away") {
		t.Errorf("expected summary to contain '120 m away', got %q", txt)
	}
	if !strings.Contains(txt, "1.5 km away") {
		t.Errorf("expected summary to contain '1.5 km away', got %q", txt)
	}
}

func TestPlacesInArea_ValidationAndExecution(t *testing.T) {
	var capturedMinLat, capturedMinLng, capturedMaxLat, capturedMaxLng string
	var capturedLimit string

	env := setupDiscoveryTestEnv(t, func(w http.ResponseWriter, r *http.Request) {
		capturedMinLat = r.URL.Query().Get("min_lat")
		capturedMinLng = r.URL.Query().Get("min_lng")
		capturedMaxLat = r.URL.Query().Get("max_lat")
		capturedMaxLng = r.URL.Query().Get("max_lng")
		capturedLimit = r.URL.Query().Get("limit")

		resp := `{
			"min_lat": 50.8,
			"min_lng": 4.3,
			"max_lat": 50.9,
			"max_lng": 4.4,
			"total_hits": 1,
			"results": [
				{
					"id": "osm_n77",
					"class": "Park",
					"name_display": "Parc de Bruxelles",
					"_geo": {"lat": 50.845, "lng": 4.363}
				}
			]
		}`
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(resp))
	})
	ctx := context.Background()

	// 1. Sanity: min_lat >= max_lat
	res, err := env.clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "places_in_area",
		Arguments: map[string]any{
			"min_lat": 51.0,
			"max_lat": 50.0,
			"min_lng": 4.0,
			"max_lng": 5.0,
		},
	})
	if err != nil || !res.IsError {
		t.Fatalf("expected tool error when min_lat >= max_lat")
	}

	// 2. Sanity: min_lng >= max_lng
	res, err = env.clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "places_in_area",
		Arguments: map[string]any{
			"min_lat": 50.0,
			"max_lat": 51.0,
			"min_lng": 5.0,
			"max_lng": 4.0,
		},
	})
	if err != nil || !res.IsError {
		t.Fatalf("expected tool error when min_lng >= max_lng")
	}

	// 3. Out-of-bounds coordinates
	res, err = env.clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "places_in_area",
		Arguments: map[string]any{
			"min_lat": -95.0,
			"max_lat": 51.0,
			"min_lng": 4.0,
			"max_lng": 5.0,
		},
	})
	if err != nil || !res.IsError {
		t.Fatalf("expected tool error when min_lat < -90")
	}

	// 4. Clamping & execution: limit 0 -> default 20
	res, err = env.clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "places_in_area",
		Arguments: map[string]any{
			"min_lat": 50.8,
			"min_lng": 4.3,
			"max_lat": 50.9,
			"max_lng": 4.4,
		},
	})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected tool error: %s", res.Content[0].(*mcp.TextContent).Text)
	}

	if capturedMinLat != "50.8" || capturedMinLng != "4.3" || capturedMaxLat != "50.9" || capturedMaxLng != "4.4" {
		t.Errorf("unexpected bounding box query params: %s, %s, %s, %s", capturedMinLat, capturedMinLng, capturedMaxLat, capturedMaxLng)
	}
	if capturedLimit != "20" {
		t.Errorf("expected default limit 20, got %q", capturedLimit)
	}

	// 5. Clamping: limit 50 -> clamped to 25
	_, _ = env.clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "places_in_area",
		Arguments: map[string]any{
			"min_lat": 50.8,
			"min_lng": 4.3,
			"max_lat": 50.9,
			"max_lng": 4.4,
			"limit":   50,
		},
	})
	if capturedLimit != "25" {
		t.Errorf("expected clamped limit 25, got %q", capturedLimit)
	}

	// Verify structured output
	structured, err := json.Marshal(res.StructuredContent)
	if err != nil {
		t.Fatalf("marshaling StructuredContent: %v", err)
	}
	var out PlacesInAreaOutput
	if err := json.Unmarshal(structured, &out); err != nil {
		t.Fatalf("unmarshaling PlacesInAreaOutput: %v", err)
	}
	if out.TotalHits != 1 || len(out.Places) != 1 {
		t.Fatalf("expected 1 place in bbox, got %+v", out)
	}
	if out.Places[0].Name != "Parc de Bruxelles" {
		t.Errorf("unexpected place name: %q", out.Places[0].Name)
	}
}

func TestGetPlace_ValidationAndExecution(t *testing.T) {
	var capturedPath string
	var capturedIncludeNearby string

	env := setupDiscoveryTestEnv(t, func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		capturedIncludeNearby = r.URL.Query().Get("include_nearby")

		resp := `{
			"id": "osm_n123",
			"class": "TouristAttraction",
			"subclass": "monument",
			"name_display": "Atomium",
			"names_by_language": {"en": "Atomium", "fr": "L'Atomium", "nl": "Het Atomium"},
			"name_alt_active": ["Atomium Brussels"],
			"country_code": "BE",
			"country_name": "Belgium",
			"admin_hierarchy": ["Brussels", "Belgium"],
			"_geo": {"lat": 50.8949, "lng": 4.3415},
			"wikidata_id": "Q207",
			"popularity_score": 0.95,
			"source": "osm",
			"updated_at": "2026-10-24T12:00:00Z",
			"nearby_places": [
				{
					"id": "osm_n124",
					"class": "Park",
					"name_display": "Mini-Europe",
					"_geo": {"lat": 50.8940, "lng": 4.3400}
				}
			]
		}`
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(resp))
	})
	ctx := context.Background()

	// 1. Validation: neither id nor wikidata_id
	res, err := env.clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      "get_place",
		Arguments: map[string]any{},
	})
	if err != nil || !res.IsError {
		t.Fatalf("expected tool error when neither id nor wikidata_id provided")
	}

	// 2. Validation: both id and wikidata_id
	res, err = env.clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      "get_place",
		Arguments: map[string]any{"id": "osm_n123", "wikidata_id": "Q207"},
	})
	if err != nil || !res.IsError {
		t.Fatalf("expected tool error when both id and wikidata_id provided")
	}

	// 3. Execution by ID (compact mode: verbose=false)
	res, err = env.clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "get_place",
		Arguments: map[string]any{
			"id":             "osm_n123",
			"include_nearby": true,
			"verbose":        false,
		},
	})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected tool error: %s", res.Content[0].(*mcp.TextContent).Text)
	}

	if capturedPath != "/v1/places/osm_n123" {
		t.Errorf("expected path '/v1/places/osm_n123', got %q", capturedPath)
	}
	if capturedIncludeNearby != "true" {
		t.Errorf("expected include_nearby=true, got %q", capturedIncludeNearby)
	}

	structured, err := json.Marshal(res.StructuredContent)
	if err != nil {
		t.Fatalf("marshaling StructuredContent: %v", err)
	}
	var outCompact GetPlaceOutput
	if err := json.Unmarshal(structured, &outCompact); err != nil {
		t.Fatalf("unmarshaling GetPlaceOutput: %v", err)
	}
	if outCompact.Place.Name != "Atomium" {
		t.Errorf("expected place name Atomium, got %q", outCompact.Place.Name)
	}
	// In compact mode, bulky alias lists and FullEntity should be empty
	if len(outCompact.Place.NamesByLanguage) > 0 {
		t.Errorf("expected NamesByLanguage to be omitted in compact mode")
	}
	if outCompact.FullEntity != nil {
		t.Errorf("expected FullEntity to be nil in compact mode")
	}
	if len(outCompact.NearbyPlaces) != 1 || outCompact.NearbyPlaces[0].Name != "Mini-Europe" {
		t.Errorf("unexpected nearby places: %+v", outCompact.NearbyPlaces)
	}

	// 4. Execution by Wikidata ID (verbose mode: verbose=true)
	res, err = env.clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "get_place",
		Arguments: map[string]any{
			"wikidata_id": "q207", // should uppercase to Q207
			"verbose":     true,
		},
	})
	if err != nil {
		t.Fatalf("CallTool failed: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected tool error: %s", res.Content[0].(*mcp.TextContent).Text)
	}

	if capturedPath != "/v1/places/wikidata/Q207" {
		t.Errorf("expected path '/v1/places/wikidata/Q207', got %q", capturedPath)
	}

	structuredVerbose, err := json.Marshal(res.StructuredContent)
	if err != nil {
		t.Fatalf("marshaling StructuredContent: %v", err)
	}
	var outVerbose GetPlaceOutput
	if err := json.Unmarshal(structuredVerbose, &outVerbose); err != nil {
		t.Fatalf("unmarshaling GetPlaceOutput: %v", err)
	}
	// In verbose mode, NamesByLanguage and FullEntity are populated
	if len(outVerbose.Place.NamesByLanguage) == 0 {
		t.Errorf("expected NamesByLanguage populated in verbose mode")
	}
	if outVerbose.FullEntity == nil || outVerbose.FullEntity.NameDisplay != "Atomium" {
		t.Errorf("expected FullEntity populated in verbose mode, got %+v", outVerbose.FullEntity)
	}
}

func TestGetPlace_NotFound(t *testing.T) {
	env := setupDiscoveryTestEnv(t, func(w http.ResponseWriter, r *http.Request) {})
	env.returnError = true
	env.errorCode = http.StatusNotFound
	ctx := context.Background()

	res, err := env.clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      "get_place",
		Arguments: map[string]any{"id": "nonexistent_id"},
	})
	if err != nil {
		t.Fatalf("unexpected protocol error: %v", err)
	}
	if !res.IsError {
		t.Fatalf("expected tool error on 404")
	}
	txt := res.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(txt, "failed to get place") || !strings.Contains(txt, "404") {
		t.Errorf("expected 404 error details in message, got %q", txt)
	}
}
