package tools

import (
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

/*
================================================================================
HOW TO ADD A TOOL (Pattern for Wave 2 Agents)
================================================================================

All tools in Geomilli MCP follow the official MCP Go SDK (`mcp.AddTool`) pattern:

1. Locate the appropriate group file:
   - `geocoding.go`: Forward/reverse geocoding (`geocode`, `reverse_geocode`)
   - `discovery.go`: Search and spatial exploration (`search_places`, `find_nearby`, `places_in_area`, `get_place`)
   - `aviation.go`: Airports lookup (`find_airports`, `get_airport`)
   - `admin_hierarchy.go`: Country/region/city hierarchy (`list_countries`, `list_regions`, `list_cities`)
   - `reference.go`: Reference metadata and helpers (`list_place_categories`, `measure_distance`)
   - `operations.go`: Ops & monitoring tools (`get_service_status`, `list_import_queue`, `list_importable_countries`)

2. Define typed Input and Output structs:
   - Every field must have a `json:"..."` tag.
   - Use `jsonschema:"..."` tags to describe parameters for LLM comprehension.
   - For optional fields, include `omitempty` in the json tag.
   - If the tool takes no parameters, define an empty struct (e.g. `type MyToolInput struct{}`).

   Example:
   ```go
   type FindNearbyInput struct {
       Lat          float64 `json:"lat" jsonschema:"Latitude coordinate (-90 to 90)"`
       Lng          float64 `json:"lng" jsonschema:"Longitude coordinate (-180 to 180)"`
       RadiusMeters int     `json:"radius_meters,omitempty" jsonschema:"Search radius in meters (1 to 50000, default: 5000)"`
       Class        string  `json:"class,omitempty" jsonschema:"Filter by taxonomy class (e.g. Restaurant, Hotel, Airport)"`
       Limit        int     `json:"limit,omitempty" jsonschema:"Maximum places to return (default: 10, max: 25)"`
   }

   type FindNearbyOutput struct {
       TotalHits int                  `json:"total_hits" jsonschema:"Total matching places found"`
       Places    []shape.CompactPlace `json:"places" jsonschema:"List of nearby places sorted by distance"`
   }
   ```

3. Register with `mcp.AddTool`:
   Inside `registerXxx(s *mcp.Server, d Deps)`:
   ```go
   mcp.AddTool(s, &mcp.Tool{
       Name:        "find_nearby",
       Description: "Find points of interest (POIs) and destinations near coordinates within a radius.",
       Annotations: ReadOnlyOpenWorld, // Use preset: ReadOnly, ReadOnlyOpenWorld, DestructiveOps, etc.
   }, func(ctx context.Context, req *mcp.CallToolRequest, in FindNearbyInput) (*mcp.CallToolResult, FindNearbyOutput, error) {
       // A. Validate & clamp inputs
       limit := ClampInt(in.Limit, 1, 25)
       radius := ClampInt(in.RadiusMeters, 1, 50000)

       // B. Validate taxonomy class if provided
       if in.Class != "" {
           canonical, ok := d.Taxonomy.IsValidClass(in.Class)
           if !ok {
               return nil, FindNearbyOutput{}, ToolError(
                   "Unknown class %q. Valid classes: %s. Call list_place_categories for details.",
                   in.Class, strings.Join(d.Taxonomy.GetClassNames(), ", "),
               )
           }
           in.Class = canonical
       }

       // C. Call API client
       resp, err := d.API.Nearby(ctx, apiclient.NearbyParams{
           Lat:          in.Lat,
           Lng:          in.Lng,
           RadiusMeters: radius,
           Class:        in.Class,
           Limit:        limit,
       })
       if err != nil {
           return nil, FindNearbyOutput{}, FormatAPIError("find nearby places", err)
       }

       // D. Shape response (compact view) & build short text summary
       compact := shape.ToCompactSlice(resp.Results, false)
       summary := shape.SummarizePlaces(compact, 5)

       out := FindNearbyOutput{
           TotalHits: int(resp.TotalHits),
           Places:    compact,
       }

       // E. Return CallToolResult with text summary and structured output
       return &mcp.CallToolResult{
           Content: []mcp.Content{
               &mcp.TextContent{Text: summary},
           },
       }, out, nil
   })
   ```

4. Key Rules:
   - NEVER expose raw Meilisearch filter syntax. Use structured parameters only.
   - Keep outputs token-efficient via `shape.ToCompact` / `shape.ToCompactSlice`.
   - Always return an actionable message with `ToolError(...)` on user-recoverable errors.
   - Set accurate annotations on every tool (`ReadOnly`, `ReadOnlyOpenWorld`, `DestructiveOps`, etc.).
================================================================================
*/

// Register registers all tool groups on the provided MCP server.
func Register(s *mcp.Server, d Deps) {
	registerGeocoding(s, d)
	registerDiscovery(s, d)
	registerAviation(s, d)
	registerAdminHierarchy(s, d)
	registerReference(s, d)

	if d.Config.EnableOpsTools {
		registerOperations(s, d)
	}
}
