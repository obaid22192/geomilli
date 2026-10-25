package contract

import (
	"context"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// expectedOpenWorld defines which tools interact with open-world geospatial environments
// versus closed-world bounded metadata or pure computational algorithms.
var expectedOpenWorld = map[string]bool{
	// Open-World search, discovery & exploration tools
	"geocode":        true,
	"search_places":  true,
	"find_nearby":    true,
	"places_in_area": true,
	"find_airports":  true,

	// Closed-World direct entity lookups, reverse resolution, reference metadata & deterministic calculation
	"reverse_geocode":       false,
	"get_place":             false,
	"get_airport":           false,
	"list_countries":        false,
	"list_regions":          false,
	"list_cities":           false,
	"list_place_categories": false,
	"measure_distance":      false,

	// Tier-2 Read-Only Ops tools (bounded cluster and ingestion state)
	"get_service_status":        false,
	"list_import_queue":         false,
	"list_importable_countries": false,
}

// tier1Tools lists the 13 core public query tools that must always be available.
var tier1Tools = []string{
	"geocode",
	"reverse_geocode",
	"search_places",
	"find_nearby",
	"places_in_area",
	"get_place",
	"find_airports",
	"get_airport",
	"list_countries",
	"list_regions",
	"list_cities",
	"list_place_categories",
	"measure_distance",
}

// tier2OpsTools lists the 3 read-only ops tools gated behind MCP_ENABLE_OPS_TOOLS.
var tier2OpsTools = []string{
	"get_service_status",
	"list_import_queue",
	"list_importable_countries",
}

func TestToolAnnotationsAudit_AllToolsEnabled(t *testing.T) {
	clientSession, cleanup := setupInMemoryServer(t, true)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	toolsList, err := clientSession.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools failed: %v", err)
	}

	toolMap := make(map[string]*mcp.Tool)
	for _, tool := range toolsList.Tools {
		toolMap[tool.Name] = tool
	}

	// 1. Verify all 16 tools are registered
	if len(toolMap) != len(expectedToolNames) {
		t.Errorf("expected %d tools, found %d", len(expectedToolNames), len(toolMap))
	}

	for _, name := range expectedToolNames {
		tool, exists := toolMap[name]
		if !exists {
			t.Errorf("expected tool %q was not found in server registry", name)
			continue
		}

		// 2. Assert annotations are non-nil
		if tool.Annotations == nil {
			t.Errorf("tool %q has nil Annotations", name)
			continue
		}

		ann := tool.Annotations

		// 3. Assert ReadOnlyHint == true for all query and read-only ops tools
		if !ann.ReadOnlyHint {
			t.Errorf("tool %q expected ReadOnlyHint == true, got false", name)
		}

		// 4. Assert IdempotentHint == true for read-only tools
		if !ann.IdempotentHint {
			t.Errorf("tool %q expected IdempotentHint == true, got false", name)
		}

		// 5. Assert DestructiveHint is nil or false
		if ann.DestructiveHint != nil && *ann.DestructiveHint {
			t.Errorf("tool %q expected DestructiveHint == false/nil, got true", name)
		}

		// 6. Assert OpenWorldHint matches specification
		expectedOW, ok := expectedOpenWorld[name]
		if !ok {
			t.Errorf("tool %q has no expected OpenWorldHint specification", name)
			continue
		}

		if ann.OpenWorldHint == nil {
			t.Errorf("tool %q has nil OpenWorldHint, expected %v", name, expectedOW)
		} else if *ann.OpenWorldHint != expectedOW {
			t.Errorf("tool %q has OpenWorldHint == %v, expected %v", name, *ann.OpenWorldHint, expectedOW)
		}

		// 7. Assert tool has non-empty description
		if len(tool.Description) == 0 {
			t.Errorf("tool %q has empty description", name)
		}
	}
}

func TestToolAnnotationsAudit_Tier1DefaultMode(t *testing.T) {
	// Start server with ops tools disabled (default production mode)
	clientSession, cleanup := setupInMemoryServer(t, false)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	toolsList, err := clientSession.ListTools(ctx, nil)
	if err != nil {
		t.Fatalf("ListTools failed: %v", err)
	}

	toolMap := make(map[string]*mcp.Tool)
	for _, tool := range toolsList.Tools {
		toolMap[tool.Name] = tool
	}

	// 1. Assert exactly 13 Tier-1 tools are exposed
	if len(toolMap) != len(tier1Tools) {
		t.Errorf("expected %d Tier-1 tools when ops tools disabled, got %d", len(tier1Tools), len(toolMap))
	}

	// 2. Assert none of the ops tools are exposed
	for _, opsTool := range tier2OpsTools {
		if _, exists := toolMap[opsTool]; exists {
			t.Errorf("ops tool %q was unexpectedly registered when ops tools disabled", opsTool)
		}
	}

	// 3. Assert all Tier-1 tools have valid annotations
	for _, name := range tier1Tools {
		tool, exists := toolMap[name]
		if !exists {
			t.Errorf("tier-1 tool %q missing from registry", name)
			continue
		}

		if tool.Annotations == nil {
			t.Errorf("tier-1 tool %q has nil Annotations", name)
			continue
		}

		if !tool.Annotations.ReadOnlyHint {
			t.Errorf("tier-1 tool %q expected ReadOnlyHint == true", name)
		}

		expectedOW := expectedOpenWorld[name]
		if tool.Annotations.OpenWorldHint == nil || *tool.Annotations.OpenWorldHint != expectedOW {
			t.Errorf("tier-1 tool %q OpenWorldHint expected %v, got %v", name, expectedOW, tool.Annotations.OpenWorldHint)
		}
	}
}
