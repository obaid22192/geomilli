package tools

import (
	"errors"
	"fmt"
	"log/slog"

	"github.com/geomilli/geomilli-mcp/internal/apiclient"
	"github.com/geomilli/geomilli-mcp/internal/config"
	"github.com/geomilli/geomilli-mcp/internal/taxonomy"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Deps holds dependencies injected into tool registration functions.
type Deps struct {
	API      *apiclient.Client
	Taxonomy *taxonomy.Cache
	Config   *config.Config
	Logger   *slog.Logger
}

// Annotation presets for standard tool behaviors across Geomilli MCP tools.
var (
	// ReadOnly is for deterministic, side-effect-free queries with closed-world scope
	// (e.g. metadata, taxonomy, country list, distance calculation).
	ReadOnly = &mcp.ToolAnnotations{
		ReadOnlyHint:   true,
		IdempotentHint: true,
		OpenWorldHint:  ptr(false),
	}

	// ReadOnlyOpenWorld is for queries that interact with open-world geospatial data
	// (e.g. geocode, search, nearby, reverse geocode, bounding box, places).
	ReadOnlyOpenWorld = &mcp.ToolAnnotations{
		ReadOnlyHint:   true,
		IdempotentHint: true,
		OpenWorldHint:  ptr(true),
	}

	// DestructiveOps is for mutations that alter or cancel operations (e.g. cancel_import, clear_import_queue).
	DestructiveOps = &mcp.ToolAnnotations{
		ReadOnlyHint:    false,
		DestructiveHint: ptr(true),
		IdempotentHint:  false,
		OpenWorldHint:   ptr(false),
	}

	// NonDestructiveOps is for mutations that are additive (e.g. queue_country_import).
	NonDestructiveOps = &mcp.ToolAnnotations{
		ReadOnlyHint:    false,
		DestructiveHint: ptr(false),
		IdempotentHint:  false,
		OpenWorldHint:   ptr(false),
	}
)

func ptr[T any](v T) *T {
	return &v
}

// ClampInt clamps val between min and max inclusive.
func ClampInt(val, min, max int) int {
	if val < min {
		return min
	}
	if val > max {
		return max
	}
	return val
}

// ClampFloat64 clamps val between min and max inclusive.
func ClampFloat64(val, min, max float64) float64 {
	if val < min {
		return min
	}
	if val > max {
		return max
	}
	return val
}

// ToolError creates an actionable error intended for LLM self-correction.
// When returned by a ToolHandlerFor function, the SDK automatically packages it into
// CallToolResult with IsError: true and the error message in the Content block.
func ToolError(format string, args ...any) error {
	return fmt.Errorf(format, args...)
}

// FormatAPIError unpacks an upstream apiclient.APIError into an actionable tool error description.
func FormatAPIError(action string, err error) error {
	var apiErr *apiclient.APIError
	if errors.As(err, &apiErr) {
		if apiErr.Code != "" {
			return fmt.Errorf("failed to %s: %s (code: %s, status: %d)", action, apiErr.Message, apiErr.Code, apiErr.Status)
		}
		return fmt.Errorf("failed to %s: %s (status: %d)", action, apiErr.Message, apiErr.Status)
	}
	return fmt.Errorf("failed to %s: %w", action, err)
}
