package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/geomilli/geomilli-mcp/internal/apiclient"
	"github.com/geomilli/geomilli-mcp/internal/config"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type mockOpsTransport func(req *http.Request) (*http.Response, error)

func (f mockOpsTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func setupOpsTestClient(t *testing.T, enableOps bool, handler http.HandlerFunc) (*mcp.ClientSession, func()) {
	t.Helper()

	transport := mockOpsTransport(func(req *http.Request) (*http.Response, error) {
		rw := &mockOpsResponseWriter{
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
	deps := Deps{
		API:    api,
		Config: &config.Config{EnableOpsTools: enableOps},
		Logger: slog.New(slog.NewJSONHandler(io.Discard, nil)),
	}

	mcpServer := mcp.NewServer(
		&mcp.Implementation{Name: "test-geomilli-ops", Version: "1.0.0"},
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

type mockOpsResponseWriter struct {
	header http.Header
	buf    *bytes.Buffer
	status int
}

func (w *mockOpsResponseWriter) Header() http.Header { return w.header }
func (w *mockOpsResponseWriter) Write(b []byte) (int, error) {
	return w.buf.Write(b)
}
func (w *mockOpsResponseWriter) WriteHeader(statusCode int) { w.status = statusCode }

func TestOperations_Gating(t *testing.T) {
	// 1. When EnableOpsTools is false, ops tools must not be registered.
	csDisabled, cleanupDisabled := setupOpsTestClient(t, false, func(w http.ResponseWriter, req *http.Request) {})
	defer cleanupDisabled()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	toolsList, err := csDisabled.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("failed to list tools: %v", err)
	}

	opsToolNames := map[string]bool{
		"get_service_status":        false,
		"list_import_queue":         false,
		"list_importable_countries": false,
	}

	for _, tool := range toolsList.Tools {
		if _, ok := opsToolNames[tool.Name]; ok {
			t.Errorf("tool %q should NOT be registered when EnableOpsTools is false", tool.Name)
		}
	}

	// 2. When EnableOpsTools is true, all three ops tools must be registered.
	csEnabled, cleanupEnabled := setupOpsTestClient(t, true, func(w http.ResponseWriter, req *http.Request) {})
	defer cleanupEnabled()

	toolsListEnabled, err := csEnabled.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("failed to list tools when enabled: %v", err)
	}

	for _, tool := range toolsListEnabled.Tools {
		if _, ok := opsToolNames[tool.Name]; ok {
			opsToolNames[tool.Name] = true
		}
	}

	for name, found := range opsToolNames {
		if !found {
			t.Errorf("expected tool %q to be registered when EnableOpsTools is true", name)
		}
	}
}

func TestOperations_GetServiceStatus(t *testing.T) {
	var importerFails atomic.Bool

	mockAPI := func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch req.URL.Path {
		case "/v1/health":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"status": "ok",
				"version": "1.0.0",
				"timestamp": "2026-10-24T22:00:00Z",
				"meilisearch": {"status": "available", "database_size_bytes": 10485760}
			}`))
		case "/v1/importer/status":
			if importerFails.Load() {
				w.WriteHeader(http.StatusServiceUnavailable)
				w.Write([]byte(`{"error": {"code": "daemon_offline", "message": "harvester daemon down", "status": 503}}`))
				return
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"state": "idle",
				"resources": {
					"heap_alloc_mb": 128.5,
					"sys_mem_mb": 256.0,
					"disk_staging_mb": 10.0,
					"disk_cache_mb": 50.0,
					"sqlite_cache_entries": 12000
				}
			}`))
		case "/v1/metrics":
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"timestamp": "2026-10-24T22:00:00Z",
				"total_requests": 50000,
				"active_qps": 42.5,
				"latency": {"p50_ms": 1.2, "p95_ms": 4.8, "p99_ms": 12.0},
				"meilisearch": {"index_docs": 1850000, "status": "available"}
			}`))
		default:
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte(`{"error": "not found"}`))
		}
	}

	cs, cleanup := setupOpsTestClient(t, true, mockAPI)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 1. Healthy cluster test
	t.Run("healthy_cluster", func(t *testing.T) {
		res, err := cs.CallTool(ctx, &mcp.CallToolParams{
			Name:      "get_service_status",
			Arguments: map[string]any{},
		})
		if err != nil {
			t.Fatalf("call tool get_service_status failed: %v", err)
		}
		if res.IsError {
			t.Fatalf("expected successful call, got tool error: %v", res.Content)
		}

		textContent, ok := res.Content[0].(*mcp.TextContent)
		if !ok {
			t.Fatalf("expected *mcp.TextContent")
		}
		if !strings.Contains(textContent.Text, "ok") {
			t.Errorf("expected text to mention ok, got: %s", textContent.Text)
		}
	})

	// 2. Degraded cluster test (importer fails, health and metrics succeed)
	t.Run("degraded_cluster", func(t *testing.T) {
		importerFails.Store(true)
		res, err := cs.CallTool(ctx, &mcp.CallToolParams{
			Name:      "get_service_status",
			Arguments: map[string]any{},
		})
		if err != nil {
			t.Fatalf("call tool get_service_status degraded failed: %v", err)
		}
		if res.IsError {
			t.Fatalf("expected degraded status to return without protocol tool error")
		}

		textContent, ok := res.Content[0].(*mcp.TextContent)
		if !ok {
			t.Fatalf("expected *mcp.TextContent")
		}
		if !strings.Contains(textContent.Text, "degraded") {
			t.Errorf("expected summary to indicate degraded status, got: %s", textContent.Text)
		}
	})
}

func TestOperations_ListImportQueue(t *testing.T) {
	var returnError atomic.Bool

	mockAPI := func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if req.URL.Path == "/v1/importer/queue" {
			if returnError.Load() {
				w.WriteHeader(http.StatusInternalServerError)
				w.Write([]byte(`{"error": {"code": "internal_error", "message": "database locked", "status": 500}}`))
				return
			}

			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"concurrency": 2,
				"active_workers": [
					{
						"id": 1,
						"country_code": "BE",
						"country_name": "Belgium",
						"stage": "indexing",
						"progress_pct": 75,
						"docs_processed": 150000,
						"status": "running"
					}
				],
				"queued_tasks": [
					{
						"id": "task_lux",
						"country_code": "LU",
						"country_name": "Luxembourg",
						"online_enrich": true,
						"mode": "full",
						"status": "pending"
					}
				],
				"country_locks": {"BE": "worker_1"}
			}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}

	cs, cleanup := setupOpsTestClient(t, true, mockAPI)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 1. Success test
	res, err := cs.CallTool(ctx, &mcp.CallToolParams{
		Name:      "list_import_queue",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("call tool list_import_queue failed: %v", err)
	}
	if res.IsError {
		t.Fatalf("unexpected tool error: %v", res.Content)
	}

	textContent := res.Content[0].(*mcp.TextContent)
	if !strings.Contains(textContent.Text, "1 active workers") || !strings.Contains(textContent.Text, "1 queued tasks") {
		t.Errorf("unexpected summary text: %s", textContent.Text)
	}

	// 2. Error handling test
	returnError.Store(true)
	resErr, err := cs.CallTool(ctx, &mcp.CallToolParams{
		Name:      "list_import_queue",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("expected tool call to return normally with IsError set")
	}
	if !resErr.IsError {
		t.Errorf("expected IsError=true on upstream 500")
	}
}

func TestOperations_ListImportableCountries(t *testing.T) {
	mockAPI := func(w http.ResponseWriter, req *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		if req.URL.Path == "/v1/importer/countries" {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{
				"countries": [
					{
						"code": "BE",
						"name": "Belgium",
						"continent": "europe",
						"geofabrik_path": "europe/belgium-latest.osm.pbf",
						"is_indexed": true,
						"doc_count": 150000
					},
					{
						"code": "NL",
						"name": "Netherlands",
						"continent": "europe",
						"geofabrik_path": "europe/netherlands-latest.osm.pbf",
						"is_indexed": false,
						"doc_count": 0
					},
					{
						"code": "JP",
						"name": "Japan",
						"continent": "asia",
						"geofabrik_path": "asia/japan-latest.osm.pbf",
						"is_indexed": true,
						"doc_count": 450000
					}
				]
			}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
	}

	cs, cleanup := setupOpsTestClient(t, true, mockAPI)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// 1. All countries
	t.Run("all_countries", func(t *testing.T) {
		res, err := cs.CallTool(ctx, &mcp.CallToolParams{
			Name:      "list_importable_countries",
			Arguments: map[string]any{},
		})
		if err != nil {
			t.Fatalf("call failed: %v", err)
		}
		if res.IsError {
			t.Fatalf("tool returned error: %v", res.Content)
		}

		textContent := res.Content[0].(*mcp.TextContent)
		if !strings.Contains(textContent.Text, "Found 3 importable countries") {
			t.Errorf("unexpected text: %s", textContent.Text)
		}
	})

	// 2. Filter only_indexed = true
	t.Run("only_indexed", func(t *testing.T) {
		res, err := cs.CallTool(ctx, &mcp.CallToolParams{
			Name: "list_importable_countries",
			Arguments: map[string]any{
				"only_indexed": true,
			},
		})
		if err != nil {
			t.Fatalf("call failed: %v", err)
		}
		textContent := res.Content[0].(*mcp.TextContent)
		if !strings.Contains(textContent.Text, "Found 2 importable countries") {
			t.Errorf("expected 2 indexed countries, got: %s", textContent.Text)
		}
	})

	// 3. Filter continent = "europe"
	t.Run("continent_europe", func(t *testing.T) {
		res, err := cs.CallTool(ctx, &mcp.CallToolParams{
			Name: "list_importable_countries",
			Arguments: map[string]any{
				"continent": "Europe",
			},
		})
		if err != nil {
			t.Fatalf("call failed: %v", err)
		}
		textContent := res.Content[0].(*mcp.TextContent)
		if !strings.Contains(textContent.Text, "Found 2 importable countries") {
			t.Errorf("expected 2 european countries, got: %s", textContent.Text)
		}
	})

	// 4. Filter only_indexed = true AND continent = "asia"
	t.Run("indexed_asia", func(t *testing.T) {
		res, err := cs.CallTool(ctx, &mcp.CallToolParams{
			Name: "list_importable_countries",
			Arguments: map[string]any{
				"only_indexed": true,
				"continent":    "asia",
			},
		})
		if err != nil {
			t.Fatalf("call failed: %v", err)
		}
		textContent := res.Content[0].(*mcp.TextContent)
		if !strings.Contains(textContent.Text, "Found 1 importable countries") {
			t.Errorf("expected 1 country, got: %s", textContent.Text)
		}
	})
}

// Unused variable guard to satisfy imports
var _ = json.Marshal
