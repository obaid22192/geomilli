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

type mockAviationTransport func(req *http.Request) (*http.Response, error)

func (m mockAviationTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return m(req)
}

func setupAviationTest(t *testing.T, handler http.HandlerFunc) (*mcp.ClientSession, func()) {
	client := apiclient.NewClient("http://fake-api", "", 5*time.Second, &http.Client{
		Transport: mockAviationTransport(func(req *http.Request) (*http.Response, error) {
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
	registerAviation(server, deps)

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

func TestFindAirports_Success(t *testing.T) {
	clientSession, cleanup := setupAviationTest(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/airports" {
			http.NotFound(w, r)
			return
		}

		q := r.URL.Query()
		if q.Get("q") != "Brussels" || q.Get("country") != "BE" || q.Get("iata") != "BRU" {
			t.Errorf("unexpected query params: %v", q)
		}
		if q.Get("limit") != "10" {
			t.Errorf("expected limit=10, got %s", q.Get("limit"))
		}

		resp := `{
			"query": "Brussels",
			"took_ms": 3,
			"total_hits": 1,
			"results": [{
				"id": "osm_a123",
				"name_display": "Brussels Airport",
				"class": "Airport",
				"subclass": "international",
				"country_code": "BE",
				"country_name": "Belgium",
				"iata_code": "BRU",
				"icao_code": "EBBR",
				"_geo": {"lat": 50.9010, "lng": 4.4844},
				"popularity_score": 0.95
			}]
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
	var findTool *mcp.Tool
	for _, tool := range toolsList.Tools {
		if tool.Name == "find_airports" {
			findTool = tool
			break
		}
	}
	if findTool == nil {
		t.Fatalf("tool find_airports not found")
	}
	if findTool.Annotations == nil || !findTool.Annotations.ReadOnlyHint || findTool.Annotations.OpenWorldHint == nil || !*findTool.Annotations.OpenWorldHint {
		t.Errorf("expected ReadOnlyOpenWorld annotation, got %+v", findTool.Annotations)
	}

	callRes, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name: "find_airports",
		Arguments: map[string]any{
			"query":   "Brussels",
			"country": "be",
			"iata":    "bru",
			"limit":   10,
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
	var out FindAirportsOutput
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("failed to unmarshal output: %v", err)
	}

	if out.TotalHits != 1 || len(out.Airports) != 1 {
		t.Fatalf("unexpected output: %+v", out)
	}
	if out.Airports[0].Name != "Brussels Airport" || out.Airports[0].IATACode != "BRU" || out.Airports[0].ICAOCode != "EBBR" {
		t.Errorf("unexpected airport: %+v", out.Airports[0])
	}

	textContent := callRes.Content[0].(*mcp.TextContent)
	if !strings.Contains(textContent.Text, "Brussels Airport") {
		t.Errorf("summary does not contain airport name: %s", textContent.Text)
	}
}

func TestFindAirports_ValidationErrors(t *testing.T) {
	clientSession, cleanup := setupAviationTest(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"total_hits": 0, "results": []}`))
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
			name:      "invalid country code length",
			args:      map[string]any{"country": "BEL"},
			errSubstr: "Country code must be a 2-letter ISO",
		},
		{
			name:      "invalid iata length",
			args:      map[string]any{"iata": "B"},
			errSubstr: "IATA airport code must be a 3-letter code",
		},
		{
			name:      "invalid icao length",
			args:      map[string]any{"icao": "EBB"},
			errSubstr: "ICAO airport code must be a 4-letter code",
		},
		{
			name:      "lat without lng",
			args:      map[string]any{"lat": 50.8467},
			errSubstr: "Both lat and lng coordinates must be provided together",
		},
		{
			name:      "lat out of bounds",
			args:      map[string]any{"lat": 95.0, "lng": 4.35},
			errSubstr: "Latitude must be between -90 and 90",
		},
		{
			name:      "lng out of bounds",
			args:      map[string]any{"lat": 50.0, "lng": 185.0},
			errSubstr: "Longitude must be between -180 and 180",
		},
		{
			name:      "unsupported language",
			args:      map[string]any{"lang": "unsupported_lang"},
			errSubstr: "Unsupported language code",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
				Name:      "find_airports",
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

func TestFindAirports_UpstreamError(t *testing.T) {
	clientSession, cleanup := setupAviationTest(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"error": {"code": "upstream_unavailable", "message": "meilisearch connection refused", "status": 502}}`))
	})
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      "find_airports",
		Arguments: map[string]any{"query": "Heathrow"},
	})
	if err != nil {
		t.Fatalf("unexpected protocol error: %v", err)
	}
	if !res.IsError {
		t.Fatalf("expected IsError=true on upstream failure")
	}
	text := res.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(text, "failed to search airports") {
		t.Errorf("expected actionable error message, got %q", text)
	}
}

func TestGetAirport_Success(t *testing.T) {
	clientSession, cleanup := setupAviationTest(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/airports/BRU" {
			http.NotFound(w, r)
			return
		}

		resp := `{
			"id": "osm_a123",
			"name_display": "Brussels Airport",
			"class": "Airport",
			"subclass": "international",
			"country_code": "BE",
			"country_name": "Belgium",
			"iata_code": "BRU",
			"icao_code": "EBBR",
			"_geo": {"lat": 50.9010, "lng": 4.4844},
			"popularity_score": 0.95
		}`
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(resp))
	})
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Verify annotations
	toolsList, err := clientSession.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("failed to list tools: %v", err)
	}
	var getTool *mcp.Tool
	for _, tool := range toolsList.Tools {
		if tool.Name == "get_airport" {
			getTool = tool
			break
		}
	}
	if getTool == nil {
		t.Fatalf("tool get_airport not found")
	}
	if getTool.Annotations == nil || !getTool.Annotations.ReadOnlyHint || getTool.Annotations.OpenWorldHint == nil || *getTool.Annotations.OpenWorldHint {
		t.Errorf("expected ReadOnly annotation, got %+v", getTool.Annotations)
	}

	callRes, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      "get_airport",
		Arguments: map[string]any{"code": "bru"},
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
	var out GetAirportOutput
	if err := json.Unmarshal(raw, &out); err != nil {
		t.Fatalf("failed to unmarshal output: %v", err)
	}

	if out.Airport.ID != "osm_a123" || out.Airport.IATACode != "BRU" || out.Airport.ICAOCode != "EBBR" {
		t.Errorf("unexpected airport output: %+v", out.Airport)
	}
}

func TestGetAirport_ValidationErrors(t *testing.T) {
	clientSession, cleanup := setupAviationTest(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{}`))
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
			name:      "empty code",
			args:      map[string]any{"code": ""},
			errSubstr: "Airport code is required",
		},
		{
			name:      "code too short",
			args:      map[string]any{"code": "B"},
			errSubstr: "must be a 3-letter IATA code or 4-letter ICAO code",
		},
		{
			name:      "code too long",
			args:      map[string]any{"code": "TOOLONG"},
			errSubstr: "must be a 3-letter IATA code or 4-letter ICAO code",
		},
		{
			name:      "unsupported language",
			args:      map[string]any{"code": "BRU", "lang": "badlang"},
			errSubstr: "Unsupported language code",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			res, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
				Name:      "get_airport",
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

func TestGetAirport_NotFound(t *testing.T) {
	clientSession, cleanup := setupAviationTest(t, func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error": {"code": "not_found", "message": "Airport not found for code: ZZZ", "status": 404}}`))
	})
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	res, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      "get_airport",
		Arguments: map[string]any{"code": "ZZZ"},
	})
	if err != nil {
		t.Fatalf("unexpected protocol error: %v", err)
	}
	if !res.IsError {
		t.Fatalf("expected IsError=true for non-existent airport")
	}
	text := res.Content[0].(*mcp.TextContent).Text
	if !strings.Contains(text, "Airport not found for code: ZZZ") {
		t.Errorf("expected not found message, got %q", text)
	}
}
