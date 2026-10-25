package resources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/geomilli/geomilli-mcp/internal/apiclient"
	"github.com/geomilli/geomilli-mcp/internal/tools"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const placeSchemaMarkdown = `# Geomilli Place Document Schema

The Place document is the canonical geospatial data model returned by Geomilli for destinations, administrative units, transit hubs, and points of interest (POIs).

## Schema Fields

| Field | Type | Description |
| :--- | :--- | :--- |
| id | string | Unique primary identifier formatted as osm_{type}{id} (e.g. osm_n12345 for node, osm_w67890 for way, osm_r112233 for relation). |
| entity_id | string | Raw OpenStreetMap entity identifier (e.g. node/12345, way/67890, relation/112233). |
| class | string | Canonical taxonomy category. Strictly one of the 19 canonical classes defined in config/taxonomy.yaml (e.g. City, Airport, Hotel, Restaurant, Attraction, Beach, Stadium). |
| subclass | string | Secondary category or OSM tag value providing fine-grained classification (e.g. monument, museum, international, italian, chalet). |
| name_display | string | Preferred primary display name, localized based on requested language if available, falling back to international or default name. |
| names_by_language | map[string]string | Map of ISO language codes to localized names (e.g. {"en": "Brussels", "fr": "Bruxelles", "nl": "Brussel"}). |
| name_alt_active | []string | Active alternative names, aliases, and transliterations used for search indexing. |
| name_alt_historical | []string | Historical names (useful for cultural heritage research). |
| country_code | string | ISO 3166-1 alpha-2 two-letter country code (e.g. BE, FR, DE, US). |
| country_name | string | English display name of the country. |
| admin_hierarchy | []string | Enclosing administrative division path ordered from local neighborhood to country (e.g. ["Laeken", "Brussels", "Brussels-Capital Region", "Belgium"]). |
| parent_entity_id | string | Identifier of parent administrative division or containing complex. |
| _geo | object | Geographic coordinate point: {"lat": float64, "lng": float64}. Latitude [-90.0, 90.0], Longitude [-180.0, 180.0]. |
| bbox | []float64 | Geographic bounding box [min_lat, min_lng, max_lat, max_lng] for areas, cities, and regions. |
| iata_code | string | 3-letter IATA airport code (e.g. BRU, CRL, CDG). Present for airports. |
| icao_code | string | 4-letter ICAO airport code (e.g. EBBR, EBCI, LFPG). Present for airports. |
| star_rating | integer | Hotel or lodging star rating (1 to 5). |
| cuisine | string | Cuisine classification for dining places (e.g. italian, french, japanese). |
| accommodation_type | string | Specific lodging type (e.g. hotel, hostel, motel, guest_house). |
| admin_level | integer | OSM administrative boundary level (2 = Country, 4 = Region/State, 6 = County/Province, 8 = Municipality/City). |
| population | integer | Resident population count for settlements and administrative divisions. |
| wikidata_id | string | Wikidata entity QID (e.g. Q207 for Atomium, Q239 for Brussels). |
| popularity_score | float64 | Normalized composite popularity score between 0.0 and 1.0. |
| distance_meters | float64 | Calculated distance in meters from query coordinates (populated on find_nearby and reverse_geocode). |
| source | string | Data origin indicator (e.g. osm, wikidata). |
| updated_at | string | ISO 8601 timestamp of index creation or update. |

## Scoring & Popularity Ranking

The popularity_score (0.0 to 1.0) is a composite metric calculated during data harvesting:
1. Taxonomy Base Score: Assigned baseline weight from config/taxonomy.yaml (e.g. City: 0.90, Airport: 0.85, Attraction: 0.80, Restaurant: 0.50).
2. Wikipedia Pageviews: Log-normalized 30-day global pageview traffic across all language editions of Wikipedia.
3. Wikidata Sitelinks: Count of cross-language Wikimedia projects referencing the entity.
4. Administrative Hierarchy Boost: Capital cities and primary regional seats receive hierarchy-based weight boosts.

## Coordinate Representations

- All coordinates use WGS 84 (EPSG:4326) decimal degrees.
- _geo.lat: Latitude [-90.0, 90.0].
- _geo.lng: Longitude [-180.0, 180.0].
- bbox: Bounding box array formatted as [min_lat, min_lng, max_lat, max_lng].
`

// Register registers all Geomilli resources on the provided MCP server.
func Register(s *mcp.Server, d tools.Deps) {
	// 1. geomilli://taxonomy
	s.AddResource(&mcp.Resource{
		URI:         "geomilli://taxonomy",
		Name:        "taxonomy",
		Title:       "Taxonomy Configuration",
		Description: "Full taxonomy configuration including the 19 canonical place classes, match rules, layers, and base scores.",
		MIMEType:    "application/json",
	}, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		tax, err := d.API.GetTaxonomy(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch taxonomy: %w", err)
		}
		b, err := json.MarshalIndent(tax, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to encode taxonomy JSON: %w", err)
		}
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{
				{
					URI:      "geomilli://taxonomy",
					MIMEType: "application/json",
					Text:     string(b),
				},
			},
		}, nil
	})

	// 2. geomilli://languages
	s.AddResource(&mcp.Resource{
		URI:         "geomilli://languages",
		Name:        "languages",
		Title:       "Supported Languages",
		Description: "Whitelisted language codes and localized display names supported by Geomilli.",
		MIMEType:    "application/json",
	}, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		langs, err := d.API.GetLanguages(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch languages: %w", err)
		}
		b, err := json.MarshalIndent(langs, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to encode languages JSON: %w", err)
		}
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{
				{
					URI:      "geomilli://languages",
					MIMEType: "application/json",
					Text:     string(b),
				},
			},
		}, nil
	})

	// 3. geomilli://countries
	s.AddResource(&mcp.Resource{
		URI:         "geomilli://countries",
		Name:        "countries",
		Title:       "Indexed Countries",
		Description: "List of countries currently indexed in the Geomilli database with place counts.",
		MIMEType:    "application/json",
	}, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		countries, err := d.API.ListCountries(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch countries: %w", err)
		}
		b, err := json.MarshalIndent(countries, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to encode countries JSON: %w", err)
		}
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{
				{
					URI:      "geomilli://countries",
					MIMEType: "application/json",
					Text:     string(b),
				},
			},
		}, nil
	})

	// 4. geomilli://schema/place
	s.AddResource(&mcp.Resource{
		URI:         "geomilli://schema/place",
		Name:        "place_schema",
		Title:       "Place Schema Documentation",
		Description: "Markdown documentation describing the Geomilli Place document schema, fields, coordinate representation, and popularity scoring.",
		MIMEType:    "text/markdown",
	}, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{
				{
					URI:      "geomilli://schema/place",
					MIMEType: "text/markdown",
					Text:     placeSchemaMarkdown,
				},
			},
		}, nil
	})

	// 5. geomilli://places/{id} (Resource Template)
	s.AddResourceTemplate(&mcp.ResourceTemplate{
		URITemplate: "geomilli://places/{id}",
		Name:        "place_by_id",
		Title:       "Place Document by ID",
		Description: "Retrieve full place document by ID (e.g. geomilli://places/osm_n123).",
		MIMEType:    "application/json",
	}, func(ctx context.Context, req *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
		uri := req.Params.URI
		const prefix = "geomilli://places/"
		if !strings.HasPrefix(uri, prefix) {
			return nil, mcp.ResourceNotFoundError(uri)
		}
		id := strings.TrimPrefix(uri, prefix)
		if id == "" {
			return nil, mcp.ResourceNotFoundError(uri)
		}

		defaultLang := ""
		if d.Config != nil {
			defaultLang = d.Config.DefaultLang
		}

		place, err := d.API.GetPlaceByID(ctx, id, defaultLang, false)
		if err != nil {
			var apiErr *apiclient.APIError
			if errors.As(err, &apiErr) && apiErr.Status == 404 {
				return nil, mcp.ResourceNotFoundError(uri)
			}
			return nil, fmt.Errorf("failed to fetch place %s: %w", id, err)
		}

		b, err := json.MarshalIndent(place, "", "  ")
		if err != nil {
			return nil, fmt.Errorf("failed to encode place JSON: %w", err)
		}

		return &mcp.ReadResourceResult{
			Contents: []*mcp.ResourceContents{
				{
					URI:      uri,
					MIMEType: "application/json",
					Text:     string(b),
				},
			},
		}, nil
	})
}
