package server

import (
	"github.com/geomilli/geomilli-mcp/internal/prompts"
	"github.com/geomilli/geomilli-mcp/internal/resources"
	"github.com/geomilli/geomilli-mcp/internal/tools"
	"github.com/geomilli/geomilli-mcp/internal/version"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ServerInstructions describes best practices for AI agents interacting with Geomilli tools.
const ServerInstructions = `Geomilli provides high-performance geospatial search and intelligence.
Best practices for using Geomilli tools:
- Use 'geocode' first to resolve place names, addresses, or queries into coordinate points and place IDs.
- Use 'find_nearby' to search for POIs around coordinates within a specified radius (in meters).
- Use 'search_places' for keyword search with structured category or country filters.
- Use 'reverse_geocode' to resolve coordinates into administrative hierarchies and addresses.
- Use 'get_place' to fetch full details by place ID or Wikidata QID.
- Use 'list_countries' to see which countries have indexed data in the database.`

// NewServer creates and initializes the Geomilli MCP server with all tools, resources, and prompts.
func NewServer(deps tools.Deps) *mcp.Server {
	s := mcp.NewServer(
		&mcp.Implementation{
			Name:    "geomilli",
			Version: version.Version,
		},
		&mcp.ServerOptions{
			Instructions: ServerInstructions,
			Logger:       deps.Logger,
		},
	)

	// Register all capabilities
	tools.Register(s, deps)
	resources.Register(s, deps)
	prompts.Register(s, deps)

	return s
}
