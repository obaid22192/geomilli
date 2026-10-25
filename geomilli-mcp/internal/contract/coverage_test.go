package contract

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// RouteKey identifies an HTTP method and path in the REST API.
type RouteKey struct {
	Method string
	Path   string
}

func (r RouteKey) String() string {
	return r.Method + " " + r.Path
}

// RouteMapping documents how an OpenAPI route is exposed via MCP tools and resources.
type RouteMapping struct {
	Tools     []string
	Resources []string
	Notes     string
}

// ExcludedRoute documents why an OpenAPI route is deliberately excluded from MCP tool exposure.
type ExcludedRoute struct {
	Reason string
}

// mappedRoutes lists every OpenAPI endpoint exposed as one or more MCP tools or resources.
var mappedRoutes = map[RouteKey]RouteMapping{
	{"GET", "/v1/autocomplete"}: {
		Tools: []string{"geocode"},
		Notes: "Forward geocoding: maps queries to places with coordinates and candidate places",
	},
	{"GET", "/v1/reverse"}: {
		Tools: []string{"reverse_geocode"},
		Notes: "Reverse geocoding: coordinates to structured address and admin hierarchy",
	},
	{"GET", "/v1/search"}: {
		Tools: []string{"search_places"},
		Notes: "Full-text search with structured taxonomy and country filters",
	},
	{"GET", "/v1/nearby"}: {
		Tools: []string{"find_nearby"},
		Notes: "Radial POI discovery around coordinate centers",
	},
	{"GET", "/v1/bbox"}: {
		Tools: []string{"places_in_area"},
		Notes: "Bounding-box spatial query within viewport limits",
	},
	{"GET", "/v1/places/{id}"}: {
		Tools: []string{"get_place"},
		Notes: "Direct place lookup by internal Geomilli/OSM ID",
	},
	{"GET", "/v1/places/wikidata/{qid}"}: {
		Tools: []string{"get_place"},
		Notes: "Direct place lookup by Wikidata QID entity identifier",
	},
	{"GET", "/v1/airports"}: {
		Tools: []string{"find_airports"},
		Notes: "Airport search by code, name, country, or proximity",
	},
	{"GET", "/v1/airports/{code}"}: {
		Tools: []string{"get_airport"},
		Notes: "Direct airport lookup by 3-letter IATA or 4-letter ICAO code",
	},
	{"GET", "/v1/countries"}: {
		Tools:     []string{"list_countries"},
		Resources: []string{"geomilli://countries"},
		Notes:     "Enumerates all indexed countries and place counts",
	},
	{"GET", "/v1/countries/{code}/regions"}: {
		Tools: []string{"list_regions"},
		Notes: "Enumerates administrative subdivisions for a country",
	},
	{"GET", "/v1/countries/{code}/cities"}: {
		Tools: []string{"list_cities"},
		Notes: "Lists major cities for a country sorted by population",
	},
	{"GET", "/v1/taxonomy"}: {
		Tools:     []string{"list_place_categories"},
		Resources: []string{"geomilli://taxonomy"},
		Notes:     "Exposes 19 canonical place categories and taxonomy metadata",
	},
	{"GET", "/v1/languages"}: {
		Resources: []string{"geomilli://languages"},
		Notes:     "Exposes supported language whitelist via resource and taxonomy cache",
	},
	{"GET", "/v1/health"}: {
		Tools: []string{"get_service_status"},
		Notes: "Aggregated in get_service_status ops tool for cluster health",
	},
	{"GET", "/v1/metrics"}: {
		Tools: []string{"get_service_status"},
		Notes: "Aggregated in get_service_status ops tool for query throughput and latency metrics",
	},
	{"GET", "/v1/importer/status"}: {
		Tools: []string{"get_service_status"},
		Notes: "Aggregated in get_service_status ops tool for harvester ingestion status",
	},
	{"GET", "/v1/importer/queue"}: {
		Tools: []string{"list_import_queue"},
		Notes: "Read-only inspection of active and queued PBF import jobs",
	},
	{"GET", "/v1/importer/countries"}: {
		Tools: []string{"list_importable_countries"},
		Notes: "Lists importable country boundaries and sync schedules",
	},
}

// excludedRoutes lists every OpenAPI endpoint deliberately excluded from MCP tool exposure with justifications.
var excludedRoutes = map[RouteKey]ExcludedRoute{
	{"GET", "/"}: {
		Reason: "Root service info/discovery endpoint; used for HTTP ping and docs redirect, not an agent tool",
	},
	{"GET", "/v1/openapi.json"}: {
		Reason: "Dynamic OpenAPI 3.1 JSON specification; used by API documentation and schema tools, not an agent tool",
	},
	{"PUT", "/v1/languages"}: {
		Reason: "Mutates index-wide language configuration in Meilisearch; belongs exclusively in Studio UI to prevent accidental re-indexing",
	},
	{"POST", "/v1/importer/jobs"}: {
		Reason: "Mutating pipeline trigger downloading PBF files and running heavy ETL; gated ops tool excluded in v1 (reserved for future admin ops)",
	},
	{"POST", "/v1/importer/cancel"}: {
		Reason: "Destructive pipeline operation cancelling in-flight ingestion; gated ops tool excluded in v1 (reserved for future admin ops)",
	},
	{"DELETE", "/v1/importer/queue"}: {
		Reason: "Destructive pipeline operation clearing all queued ingestion jobs; gated ops tool excluded in v1 (reserved for future admin ops)",
	},
	{"POST", "/v1/importer/queue/dequeue"}: {
		Reason: "Mutating queue management removing specific jobs; redundant with job cancellation and excluded from LLM tools",
	},
	{"GET", "/v1/importer/events"}: {
		Reason: "Server-Sent Events (SSE) streaming endpoint for live logs; incompatible with request/response MCP tool invocation pattern",
	},
}

var httpMethods = map[string]bool{
	"get":     true,
	"post":    true,
	"put":     true,
	"delete":  true,
	"patch":   true,
	"options": true,
	"head":    true,
	"trace":   true,
}

// findOpenAPISpecPath locates the canonical openapi.yaml spec.
func findOpenAPISpecPath() (string, error) {
	candidates := []string{
		filepath.Join("..", "..", "..", "geomilli-api", "api", "openapi.yaml"),
		filepath.Join("..", "geomilli-api", "api", "openapi.yaml"),
		filepath.Join("geomilli-api", "api", "openapi.yaml"),
	}

	for _, c := range candidates {
		if fi, err := os.Stat(c); err == nil && !fi.IsDir() {
			return c, nil
		}
	}

	// Search upwards from cwd
	dir, err := os.Getwd()
	if err == nil {
		for i := 0; i < 6; i++ {
			p := filepath.Join(dir, "geomilli-api", "api", "openapi.yaml")
			if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
				return p, nil
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}

	return "", fmt.Errorf("canonical openapi.yaml not found in candidate paths")
}

// TestOpenAPIRoutesCoverage asserts that every route documented in canonical
// geomilli-api/api/openapi.yaml is accounted for in either mappedRoutes or excludedRoutes.
func TestOpenAPIRoutesCoverage(t *testing.T) {
	specPath, err := findOpenAPISpecPath()
	if err != nil {
		t.Fatalf("failed to locate canonical openapi.yaml: %v", err)
	}

	data, err := os.ReadFile(specPath)
	if err != nil {
		t.Fatalf("failed to read openapi.yaml at %s: %v", specPath, err)
	}

	var openapiSpec struct {
		Paths map[string]map[string]any `yaml:"paths"`
	}

	if err := yaml.Unmarshal(data, &openapiSpec); err != nil {
		t.Fatalf("failed to parse openapi.yaml: %v", err)
	}

	if len(openapiSpec.Paths) == 0 {
		t.Fatalf("no paths found in openapi.yaml")
	}

	openAPIRoutes := make(map[RouteKey]bool)
	for path, item := range openapiSpec.Paths {
		for method := range item {
			methodUpper := strings.ToUpper(method)
			if httpMethods[strings.ToLower(method)] {
				openAPIRoutes[RouteKey{
					Method: methodUpper,
					Path:   path,
				}] = true
			}
		}
	}

	const expectedTotalRoutes = 27
	if len(openAPIRoutes) != expectedTotalRoutes {
		t.Errorf("expected exactly %d routes in openapi.yaml, found %d", expectedTotalRoutes, len(openAPIRoutes))
	}

	// 1. Assert that every OpenAPI route is mapped or excluded
	var unaccounted []string
	for rKey := range openAPIRoutes {
		_, isMapped := mappedRoutes[rKey]
		_, isExcluded := excludedRoutes[rKey]

		if !isMapped && !isExcluded {
			unaccounted = append(unaccounted, rKey.String())
		}
		if isMapped && isExcluded {
			t.Errorf("route %s is present in both mappedRoutes and excludedRoutes", rKey)
		}
	}

	if len(unaccounted) > 0 {
		t.Errorf("found %d OpenAPI route(s) with NO MCP mapping or exclusion decision:\n  - %s\n"+
			"Every API endpoint must be either mapped to an MCP tool/resource or documented in excludedRoutes.",
			len(unaccounted), strings.Join(unaccounted, "\n  - "))
	}

	// 2. Assert that no stale routes exist in mappedRoutes
	for rKey := range mappedRoutes {
		if !openAPIRoutes[rKey] {
			t.Errorf("mapped route %s does not exist in openapi.yaml (stale contract)", rKey)
		}
	}

	// 3. Assert that no stale routes exist in excludedRoutes
	for rKey := range excludedRoutes {
		if !openAPIRoutes[rKey] {
			t.Errorf("excluded route %s does not exist in openapi.yaml (stale exclusion)", rKey)
		}
	}

	// 4. Assert that every excluded route has a substantive justification
	for rKey, excl := range excludedRoutes {
		if strings.TrimSpace(excl.Reason) == "" {
			t.Errorf("excluded route %s has empty justification", rKey)
		} else if len(excl.Reason) < 20 {
			t.Errorf("excluded route %s justification too brief: %q", rKey, excl.Reason)
		}
	}

	// 5. Assert that every referenced tool exists in the expectedToolNames list
	registeredToolSet := make(map[string]bool)
	for _, name := range expectedToolNames {
		registeredToolSet[name] = true
	}

	for rKey, mapping := range mappedRoutes {
		for _, toolName := range mapping.Tools {
			if !registeredToolSet[toolName] {
				t.Errorf("route %s references unknown MCP tool %q", rKey, toolName)
			}
		}
	}

	t.Logf("OpenAPI contract coverage verified: %d total routes (%d mapped, %d explicitly excluded)",
		len(openAPIRoutes), len(mappedRoutes), len(excludedRoutes))
}
