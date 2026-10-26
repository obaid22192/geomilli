package tools

import (
	"errors"
	"fmt"
	"log/slog"
	"reflect"

	"github.com/geomilli/geomilli-mcp/internal/apiclient"
	"github.com/geomilli/geomilli-mcp/internal/config"
	"github.com/geomilli/geomilli-mcp/internal/taxonomy"
	"github.com/google/jsonschema-go/jsonschema"
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

// RegisterTool registers a typed tool on the MCP server with clean, portable single-type schemas.
// By default, reflection over Go pointers and slices produces `type: ["null", "X"]` arrays
// in JSON Schema 2020-12. While legal in JSON Schema, this triggers portability warnings in MCP
// Inspector and is rejected by single-type dialects such as Gemini Function Declarations and OpenAPI 3.0.
// RegisterTool pre-infers the schemas and normalizes nullable types to portable single types.
func RegisterTool[In, Out any](s *mcp.Server, t *mcp.Tool, h mcp.ToolHandlerFor[In, Out]) {
	if t.InputSchema == nil && reflect.TypeFor[In]() != reflect.TypeFor[any]() {
		schema, err := jsonschema.ForType(reflect.TypeFor[In](), nil)
		if err == nil && schema != nil {
			normalizeSchema(schema)
			t.InputSchema = schema
		}
	}
	if t.OutputSchema == nil && reflect.TypeFor[Out]() != reflect.TypeFor[any]() {
		schema, err := jsonschema.ForType(reflect.TypeFor[Out](), nil)
		if err == nil && schema != nil {
			normalizeSchema(schema)
			t.OutputSchema = schema
		}
	}
	mcp.AddTool(s, t, h)
}

// normalizeSchema recursively converts multi-type arrays (e.g. `type: ["null", "number"]`
// or `type: ["null", "array"]`) to single-type declarations (`type: "number"`, `type: "array"`).
func normalizeSchema(s *jsonschema.Schema) {
	if s == nil {
		return
	}
	if len(s.Types) > 0 {
		var nonNull []string
		for _, t := range s.Types {
			if t != "null" {
				nonNull = append(nonNull, t)
			}
		}
		if len(nonNull) == 1 {
			s.Type = nonNull[0]
			s.Types = nil
		} else if len(nonNull) == 0 {
			s.Type = "string"
			s.Types = nil
		} else {
			s.Types = nonNull
		}
	}
	for _, prop := range s.Properties {
		normalizeSchema(prop)
	}
	for _, prop := range s.PatternProperties {
		normalizeSchema(prop)
	}
	for _, item := range s.PrefixItems {
		normalizeSchema(item)
	}
	normalizeSchema(s.Items)
	for _, item := range s.ItemsArray {
		normalizeSchema(item)
	}
	normalizeSchema(s.AdditionalProperties)
	normalizeSchema(s.Contains)
	for _, def := range s.Defs {
		normalizeSchema(def)
	}
	for _, def := range s.Definitions {
		normalizeSchema(def)
	}
	for _, sub := range s.AllOf {
		normalizeSchema(sub)
	}
	for _, sub := range s.AnyOf {
		normalizeSchema(sub)
	}
	for _, sub := range s.OneOf {
		normalizeSchema(sub)
	}
}
