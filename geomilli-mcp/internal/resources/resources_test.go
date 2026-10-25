package resources

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/geomilli/geomilli-mcp/internal/apiclient"
	"github.com/geomilli/geomilli-mcp/internal/config"
	"github.com/geomilli/geomilli-mcp/internal/tools"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type mockTransport func(req *http.Request) (*http.Response, error)

func (f mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func setupTestServerAndClient(t *testing.T, handler http.HandlerFunc) (*mcp.ClientSession, func()) {
	t.Helper()

	transport := mockTransport(func(req *http.Request) (*http.Response, error) {
		rw := &mockResponseWriter{
			header: make(http.Header),
			buf:    &bytes.Buffer{},
			status: http.StatusOK,
		}
		handler(rw, req)
		return &http.Response{
			StatusCode: rw.status,
			Header:     rw.header,
			Body:       io.NopCloser(rw.buf),
		}, nil
	})

	api := apiclient.NewClient("http://mock-geomilli", "", 5*time.Second, &http.Client{Transport: transport})
	deps := tools.Deps{
		API:    api,
		Config: &config.Config{DefaultLang: "en"},
		Logger: slog.New(slog.NewJSONHandler(io.Discard, nil)),
	}

	mcpServer := mcp.NewServer(
		&mcp.Implementation{Name: "test-geomilli-resources", Version: "1.0.0"},
		nil,
	)
	Register(mcpServer, deps)

	st, ct := mcp.NewInMemoryTransports()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

	serverSession, err := mcpServer.Connect(ctx, st, nil)
	if err != nil {
		cancel()
		t.Fatalf("failed to connect server session: %v", err)
	}

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "1.0.0"}, nil)
	clientSession, err := client.Connect(ctx, ct, nil)
	if err != nil {
		serverSession.Close()
		cancel()
		t.Fatalf("failed to connect client session: %v", err)
	}

	cleanup := func() {
		clientSession.Close()
		serverSession.Close()
		cancel()
	}

	return clientSession, cleanup
}

type mockResponseWriter struct {
	header http.Header
	buf    *bytes.Buffer
	status int
}

func (w *mockResponseWriter) Header() http.Header { return w.header }
func (w *mockResponseWriter) Write(b []byte) (int, error) {
	return w.buf.Write(b)
}
func (w *mockResponseWriter) WriteHeader(statusCode int) { w.status = statusCode }

func TestResources_ListAndRead(t *testing.T) {
	var return404ForPlace atomic.Bool

	mockAPI := func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch req.URL.Path {
		case "/v1/taxonomy":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"version": 1,
				"classes": [
					{"name": "Attraction", "base_score": 0.80, "layer": "poi", "match": []},
					{"name": "City", "base_score": 0.90, "layer": "admin", "match": []}
				]
			}`))
		case "/v1/languages":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"version": 1,
				"languages": [
					{"code": "en", "name": "English", "enabled": true},
					{"code": "fr", "name": "French", "enabled": true}
				]
			}`))
		case "/v1/countries":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"total_countries": 2,
				"countries": [
					{"country_code": "BE", "country_name": "Belgium", "place_count": 150},
					{"country_code": "FR", "country_name": "France", "place_count": 80}
				]
			}`))
		case "/v1/places/osm_n12345":
			if return404ForPlace.Load() {
				w.WriteHeader(http.StatusNotFound)
				w.Write([]byte(`{"error": {"code": "not_found", "message": "place not found", "status": 404}}`))
				return
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"id": "osm_n12345",
				"name_display": "Atomium",
				"class": "Attraction",
				"subclass": "monument",
				"_geo": {"lat": 50.8949, "lng": 4.3415},
				"country_code": "BE",
				"popularity_score": 0.95
			}`))
		default:
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error": "not found"}`))
		}
	}

	cs, cleanup := setupTestServerAndClient(t, mockAPI)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 1. Verify static resources listing
	expectedStatic := map[string]bool{
		"geomilli://taxonomy":     false,
		"geomilli://languages":    false,
		"geomilli://countries":    false,
		"geomilli://schema/place": false,
	}

	for res, err := range cs.Resources(ctx, nil) {
		if err != nil {
			t.Fatalf("unexpected error iterating resources: %v", err)
		}
		if _, ok := expectedStatic[res.URI]; ok {
			expectedStatic[res.URI] = true
		}
	}

	for uri, found := range expectedStatic {
		if !found {
			t.Errorf("expected resource %s to be registered", uri)
		}
	}

	// 2. Verify resource templates listing
	foundTemplate := false
	for tmpl, err := range cs.ResourceTemplates(ctx, nil) {
		if err != nil {
			t.Fatalf("unexpected error iterating templates: %v", err)
		}
		if tmpl.URITemplate == "geomilli://places/{id}" {
			foundTemplate = true
		}
	}
	if !foundTemplate {
		t.Errorf("expected resource template geomilli://places/{id} to be registered")
	}

	// 3. Read geomilli://taxonomy
	taxRes, err := cs.ReadResource(ctx, &mcp.ReadResourceParams{URI: "geomilli://taxonomy"})
	if err != nil {
		t.Fatalf("failed to read geomilli://taxonomy: %v", err)
	}
	if len(taxRes.Contents) == 0 {
		t.Fatalf("expected content for geomilli://taxonomy")
	}
	if taxRes.Contents[0].MIMEType != "application/json" {
		t.Errorf("expected MIMEType application/json, got %s", taxRes.Contents[0].MIMEType)
	}
	if !strings.Contains(taxRes.Contents[0].Text, "Attraction") {
		t.Errorf("expected taxonomy content to contain Attraction, got: %s", taxRes.Contents[0].Text)
	}

	// 4. Read geomilli://languages
	langRes, err := cs.ReadResource(ctx, &mcp.ReadResourceParams{URI: "geomilli://languages"})
	if err != nil {
		t.Fatalf("failed to read geomilli://languages: %v", err)
	}
	if len(langRes.Contents) == 0 || !strings.Contains(langRes.Contents[0].Text, "English") {
		t.Errorf("expected languages content to contain English")
	}

	// 5. Read geomilli://countries
	ctryRes, err := cs.ReadResource(ctx, &mcp.ReadResourceParams{URI: "geomilli://countries"})
	if err != nil {
		t.Fatalf("failed to read geomilli://countries: %v", err)
	}
	if len(ctryRes.Contents) == 0 || !strings.Contains(ctryRes.Contents[0].Text, "Belgium") {
		t.Errorf("expected countries content to contain Belgium")
	}

	// 6. Read geomilli://schema/place
	schemaRes, err := cs.ReadResource(ctx, &mcp.ReadResourceParams{URI: "geomilli://schema/place"})
	if err != nil {
		t.Fatalf("failed to read geomilli://schema/place: %v", err)
	}
	if len(schemaRes.Contents) == 0 || schemaRes.Contents[0].MIMEType != "text/markdown" {
		t.Errorf("expected markdown schema content")
	}
	if !strings.Contains(schemaRes.Contents[0].Text, "Geomilli Place Document Schema") {
		t.Errorf("expected markdown title in schema content")
	}

	// 7. Read geomilli://places/osm_n12345
	placeRes, err := cs.ReadResource(ctx, &mcp.ReadResourceParams{URI: "geomilli://places/osm_n12345"})
	if err != nil {
		t.Fatalf("failed to read place resource: %v", err)
	}
	if len(placeRes.Contents) == 0 || !strings.Contains(placeRes.Contents[0].Text, "Atomium") {
		t.Errorf("expected place content to contain Atomium")
	}

	// 8. Read non-existent place (404)
	return404ForPlace.Store(true)
	_, err = cs.ReadResource(ctx, &mcp.ReadResourceParams{URI: "geomilli://places/osm_n12345"})
	if err == nil {
		t.Errorf("expected error for non-existent place resource, got nil")
	}
}
