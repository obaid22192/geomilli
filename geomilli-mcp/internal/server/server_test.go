package server

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
	"github.com/geomilli/geomilli-mcp/internal/taxonomy"
	"github.com/geomilli/geomilli-mcp/internal/tools"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type mockTransport func(req *http.Request) (*http.Response, error)

func (f mockTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return f(req)
}

func TestServer_InMemoryClientServer_ListCountries(t *testing.T) {
	var returnError atomic.Bool

	transport := mockTransport(func(req *http.Request) (*http.Response, error) {
		if req.URL.Path == "/v1/countries" {
			if returnError.Load() {
				body := `{"error": {"code": "upstream_unavailable", "message": "database offline", "status": 503}}`
				return &http.Response{
					StatusCode: http.StatusServiceUnavailable,
					Header:     http.Header{"Content-Type": []string{"application/json"}},
					Body:       io.NopCloser(bytes.NewBufferString(body)),
				}, nil
			}

			body := `{
				"total_countries": 3,
				"countries": [
					{"country_code": "BE", "country_name": "Belgium", "place_count": 150},
					{"country_code": "FR", "country_name": "France", "place_count": 80},
					{"country_code": "DE", "country_name": "Germany", "place_count": 40}
				]
			}`
			return &http.Response{
				StatusCode: http.StatusOK,
				Header:     http.Header{"Content-Type": []string{"application/json"}},
				Body:       io.NopCloser(bytes.NewBufferString(body)),
			}, nil
		}

		return &http.Response{
			StatusCode: http.StatusNotFound,
			Body:       io.NopCloser(bytes.NewBufferString(`{"error": "not found"}`)),
		}, nil
	})

	api := apiclient.NewClient("http://mock-geomilli", "", 5*time.Second, &http.Client{Transport: transport})
	tax := taxonomy.NewCache(api, 10*time.Minute)

	deps := tools.Deps{
		API:      api,
		Taxonomy: tax,
		Config:   &config.Config{},
		Logger:   slog.New(slog.NewJSONHandler(io.Discard, nil)),
	}

	mcpServer := NewServer(deps)

	serverTransport, clientTransport := mcp.NewInMemoryTransports()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// 1. Connect server session
	serverSession, err := mcpServer.Connect(ctx, serverTransport, nil)
	if err != nil {
		t.Fatalf("failed to connect server session: %v", err)
	}
	defer serverSession.Close()

	// 2. Connect client session
	client := mcp.NewClient(&mcp.Implementation{
		Name:    "test-client",
		Version: "1.0.0",
	}, nil)

	clientSession, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		t.Fatalf("failed to connect client session: %v", err)
	}
	defer clientSession.Close()

	// 3. Test ListTools
	toolsList, err := clientSession.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("failed to list tools: %v", err)
	}

	var foundListCountries bool
	var listCountriesTool *mcp.Tool
	for _, tool := range toolsList.Tools {
		if tool.Name == "list_countries" {
			foundListCountries = true
			listCountriesTool = tool
			break
		}
	}

	if !foundListCountries {
		t.Fatalf("tool 'list_countries' not found in tools list")
	}

	// Verify annotations
	if listCountriesTool.Annotations == nil {
		t.Fatalf("expected tool annotations on list_countries, got nil")
	}
	if !listCountriesTool.Annotations.ReadOnlyHint {
		t.Errorf("expected ReadOnlyHint=true")
	}
	if !listCountriesTool.Annotations.IdempotentHint {
		t.Errorf("expected IdempotentHint=true")
	}
	if listCountriesTool.Annotations.OpenWorldHint == nil || *listCountriesTool.Annotations.OpenWorldHint != false {
		t.Errorf("expected OpenWorldHint=false")
	}

	// 4. Test CallTool success path
	callRes, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      "list_countries",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("CallTool returned protocol error: %v", err)
	}
	if callRes.IsError {
		t.Fatalf("CallTool returned tool error unexpectedly: %+v", callRes)
	}

	// Check text summary
	if len(callRes.Content) == 0 {
		t.Fatalf("expected text content in tool result, got none")
	}
	textContent, ok := callRes.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", callRes.Content[0])
	}
	if !strings.Contains(textContent.Text, "Found 3 indexed countries") {
		t.Errorf("unexpected summary text: %q", textContent.Text)
	}

	// Check structured output
	structuredBytes, err := json.Marshal(callRes.StructuredContent)
	if err != nil {
		t.Fatalf("failed to marshal StructuredContent: %v", err)
	}
	var output tools.ListCountriesOutput
	if err := json.Unmarshal(structuredBytes, &output); err != nil {
		t.Fatalf("failed to unmarshal structured output: %v", err)
	}
	if output.TotalCountries != 3 || len(output.Countries) != 3 {
		t.Errorf("unexpected structured output: %+v", output)
	}
	if output.Countries[0].CountryCode != "BE" || output.Countries[0].PlaceCount != 150 {
		t.Errorf("unexpected first country: %+v", output.Countries[0])
	}

	// 5. Test CallTool error path
	returnError.Store(true)

	errRes, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
		Name:      "list_countries",
		Arguments: map[string]any{},
	})
	if err != nil {
		t.Fatalf("CallTool returned protocol error on upstream error: %v", err)
	}
	if !errRes.IsError {
		t.Errorf("expected IsError=true on upstream failure, got false")
	}
	if len(errRes.Content) == 0 {
		t.Fatalf("expected error text in content, got none")
	}
	errText, ok := errRes.Content[0].(*mcp.TextContent)
	if !ok || !strings.Contains(errText.Text, "failed to list countries") {
		t.Errorf("expected actionable error message in content, got %v", errRes.Content[0])
	}
}
