package contract

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"testing"
	"time"

	"github.com/geomilli/geomilli-mcp/internal/apiclient"
	"github.com/geomilli/geomilli-mcp/internal/config"
	"github.com/geomilli/geomilli-mcp/internal/server"
	"github.com/geomilli/geomilli-mcp/internal/taxonomy"
	"github.com/geomilli/geomilli-mcp/internal/tools"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

var updateGolden = flag.Bool("update", false, "update golden snapshot files")

// ToolSnapshot captures the full schema and behavioral contract of an MCP tool.
type ToolSnapshot struct {
	Name         string               `json:"name"`
	Description  string               `json:"description"`
	InputSchema  any                  `json:"inputSchema"`
	OutputSchema any                  `json:"outputSchema,omitempty"`
	Annotations  *mcp.ToolAnnotations `json:"annotations,omitempty"`
}

// Expected 16 tools when all tool groups (including ops tools) are enabled.
var expectedToolNames = []string{
	"find_airports",
	"find_nearby",
	"geocode",
	"get_airport",
	"get_place",
	"get_service_status",
	"list_cities",
	"list_countries",
	"list_import_queue",
	"list_importable_countries",
	"list_place_categories",
	"list_regions",
	"measure_distance",
	"places_in_area",
	"reverse_geocode",
	"search_places",
}

// setupInMemoryServer initializes an in-memory MCP client/server session with all capabilities.
func setupInMemoryServer(t *testing.T, enableOpsTools bool) (*mcp.ClientSession, func()) {
	t.Helper()

	api := apiclient.NewClient("http://mock-geomilli", "", 5*time.Second, &http.Client{
		Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(bytes.NewBufferString("{}")),
			}, nil
		}),
	})
	tax := taxonomy.NewCache(api, 10*time.Minute)

	deps := tools.Deps{
		API:      api,
		Taxonomy: tax,
		Config: &config.Config{
			EnableOpsTools: enableOpsTools,
		},
		Logger: slog.New(slog.NewJSONHandler(io.Discard, nil)),
	}

	mcpServer := server.NewServer(deps)
	serverTransport, clientTransport := mcp.NewInMemoryTransports()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

	serverSession, err := mcpServer.Connect(ctx, serverTransport, nil)
	if err != nil {
		cancel()
		t.Fatalf("failed to connect server session: %v", err)
	}

	client := mcp.NewClient(&mcp.Implementation{
		Name:    "contract-test-client",
		Version: "1.0.0",
	}, nil)

	clientSession, err := client.Connect(ctx, clientTransport, nil)
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

type roundTripFunc func(req *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

// findTestDataPath locates the testdata directory relative to the test runner.
func findTestDataPath(elem ...string) (string, error) {
	candidates := []string{
		filepath.Join(append([]string{"..", "..", "testdata"}, elem...)...),
		filepath.Join(append([]string{"testdata"}, elem...)...),
	}
	for _, c := range candidates {
		dir := filepath.Dir(c)
		if fi, err := os.Stat(dir); err == nil && fi.IsDir() {
			return c, nil
		}
	}
	// Default to relative to package
	return filepath.Join(append([]string{"..", "..", "testdata"}, elem...)...), nil
}

func TestToolsGoldenSnapshot(t *testing.T) {
	clientSession, cleanup := setupInMemoryServer(t, true)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	toolsList, err := clientSession.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools failed: %v", err)
	}

	if len(toolsList.Tools) != len(expectedToolNames) {
		t.Fatalf("expected %d tools, got %d", len(expectedToolNames), len(toolsList.Tools))
	}

	snapshots := make([]ToolSnapshot, 0, len(toolsList.Tools))
	names := make([]string, 0, len(toolsList.Tools))

	for _, tool := range toolsList.Tools {
		names = append(names, tool.Name)
		snapshots = append(snapshots, ToolSnapshot{
			Name:         tool.Name,
			Description:  tool.Description,
			InputSchema:  tool.InputSchema,
			OutputSchema: tool.OutputSchema,
			Annotations:  tool.Annotations,
		})
	}

	// Sort snapshots alphabetically by tool name for deterministic output
	sort.Slice(snapshots, func(i, j int) bool {
		return snapshots[i].Name < snapshots[j].Name
	})
	sort.Strings(names)

	// Verify all expected tools exist
	for i, expected := range expectedToolNames {
		if names[i] != expected {
			t.Errorf("tool mismatch at index %d: expected %q, got %q", i, expected, names[i])
		}
	}

	actualJSON, err := json.MarshalIndent(snapshots, "", "  ")
	if err != nil {
		t.Fatalf("failed to marshal tool snapshots: %v", err)
	}
	actualJSON = append(actualJSON, '\n')

	goldenFile, err := findTestDataPath("golden", "tools.json")
	if err != nil {
		t.Fatalf("failed to resolve golden file path: %v", err)
	}

	if *updateGolden {
		if err := os.MkdirAll(filepath.Dir(goldenFile), 0755); err != nil {
			t.Fatalf("failed to create golden directory: %v", err)
		}
		if err := os.WriteFile(goldenFile, actualJSON, 0644); err != nil {
			t.Fatalf("failed to write golden file %s: %v", goldenFile, err)
		}
		t.Logf("Updated golden file: %s", goldenFile)
		return
	}

	expectedJSON, err := os.ReadFile(goldenFile)
	if err != nil {
		if os.IsNotExist(err) {
			t.Fatalf("golden file %s does not exist. Run with -update to generate it: go test ./internal/contract -update", goldenFile)
		}
		t.Fatalf("failed to read golden file %s: %v", goldenFile, err)
	}

	if !bytes.Equal(actualJSON, expectedJSON) {
		t.Errorf("tools schema does not match golden snapshot %s.\nRun 'go test ./internal/contract -update' if this change was intended.\nDiff preview:\n%s",
			goldenFile, diffPreview(expectedJSON, actualJSON))
	}
}

// diffPreview provides a human-readable comparison when snapshots differ.
func diffPreview(expected, actual []byte) string {
	expLines := bytes.Split(expected, []byte("\n"))
	actLines := bytes.Split(actual, []byte("\n"))

	var buf bytes.Buffer
	max := len(expLines)
	if len(actLines) > max {
		max = len(actLines)
	}

	diffCount := 0
	for i := 0; i < max && diffCount < 20; i++ {
		var expLine, actLine string
		if i < len(expLines) {
			expLine = string(expLines[i])
		}
		if i < len(actLines) {
			actLine = string(actLines[i])
		}
		if expLine != actLine {
			diffCount++
			fmt.Fprintf(&buf, "Line %d:\n  - %s\n  + %s\n", i+1, expLine, actLine)
		}
	}
	if diffCount >= 20 {
		buf.WriteString("... (further diffs truncated)\n")
	}
	return buf.String()
}
