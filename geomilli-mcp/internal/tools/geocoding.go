package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/geomilli/geomilli-mcp/internal/apiclient"
	"github.com/geomilli/geomilli-mcp/internal/countries"
	"github.com/geomilli/geomilli-mcp/internal/shape"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// GeocodeInput defines input parameters for forward geocoding.
type GeocodeInput struct {
	Query   string   `json:"query" jsonschema:"Search query or place name to geocode (e.g. 'Atomium', 'Brussels', 'Grand Place')"`
	Country string   `json:"country,omitempty" jsonschema:"Optional ISO 3166-1 alpha-2 country code or country name to restrict results (e.g. 'BE', 'Belgium', 'FR', 'France')"`
	Class   string   `json:"class,omitempty" jsonschema:"Optional taxonomy class filter (e.g. 'City', 'Attraction', 'Airport', 'Hotel')"`
	Lat     *float64 `json:"lat,omitempty" jsonschema:"Optional latitude for proximity-biased results (-90 to 90)"`
	Lng     *float64 `json:"lng,omitempty" jsonschema:"Optional longitude for proximity-biased results (-180 to 180)"`
	Lang    string   `json:"lang,omitempty" jsonschema:"Optional preferred language code for place names (e.g. 'en', 'fr')"`
	Limit   int      `json:"limit,omitempty" jsonschema:"Maximum candidate places to return (default: 10, max: 10)"`
}

// GeocodeOutput models the structured result for forward geocoding.
type GeocodeOutput struct {
	Query     string               `json:"query" jsonschema:"Search query that was geocoded"`
	TotalHits int                  `json:"total_hits" jsonschema:"Total matching candidate places"`
	Places    []shape.CompactPlace `json:"places" jsonschema:"List of candidate places with coordinates and details"`
}

// ReverseGeocodeInput defines input parameters for reverse geocoding.
type ReverseGeocodeInput struct {
	Lat          float64 `json:"lat" jsonschema:"Latitude coordinate (-90 to 90)"`
	Lng          float64 `json:"lng" jsonschema:"Longitude coordinate (-180 to 180)"`
	RadiusMeters int     `json:"radius_meters,omitempty" jsonschema:"Search radius in meters (1 to 100000, default: 50000)"`
	Lang         string  `json:"lang,omitempty" jsonschema:"Optional language code for place names (e.g. 'en', 'fr')"`
}

// ReverseGeocodeOutput models the structured result for reverse geocoding.
type ReverseGeocodeOutput struct {
	Lat            float64                  `json:"lat" jsonschema:"Latitude of the query"`
	Lng            float64                  `json:"lng" jsonschema:"Longitude of the query"`
	RadiusMeters   int                      `json:"radius_meters,omitempty" jsonschema:"Effective search radius in meters"`
	Address        apiclient.ReverseAddress `json:"address" jsonschema:"Structured address components"`
	AdminHierarchy []string                 `json:"admin_hierarchy" jsonschema:"Administrative hierarchy from city to country"`
	ClosestPlace   *shape.CompactPlace      `json:"closest_place,omitempty" jsonschema:"Closest place to the given coordinates"`
}

func registerGeocoding(s *mcp.Server, d Deps) {
	// Tool 1: geocode
	RegisterTool(s, &mcp.Tool{
		Name:        "geocode",
		Description: "Forward geocoding: resolve place names, addresses, or landmarks into coordinates and candidate places.",
		Annotations: ReadOnlyOpenWorld,
	}, func(ctx context.Context, req *mcp.CallToolRequest, in GeocodeInput) (*mcp.CallToolResult, GeocodeOutput, error) {
		query := strings.TrimSpace(in.Query)
		if query == "" {
			return nil, GeocodeOutput{}, ToolError("query parameter is required")
		}

		limit := in.Limit
		if limit <= 0 {
			limit = 10
		} else {
			limit = ClampInt(limit, 1, 10)
		}

		country, err := validateCountry(in.Country)
		if err != nil {
			return nil, GeocodeOutput{}, err
		}

		class, err := validateClass(in.Class, d)
		if err != nil {
			return nil, GeocodeOutput{}, err
		}

		lang, err := resolveLanguage(in.Lang, d)
		if err != nil {
			return nil, GeocodeOutput{}, err
		}

		if (in.Lat != nil && in.Lng == nil) || (in.Lat == nil && in.Lng != nil) {
			return nil, GeocodeOutput{}, ToolError("both lat and lng must be provided together for proximity biasing")
		}

		var latPtr, lngPtr *float64
		if in.Lat != nil && in.Lng != nil {
			if err := validateLatLng(*in.Lat, *in.Lng); err != nil {
				return nil, GeocodeOutput{}, err
			}
			latPtr = in.Lat
			lngPtr = in.Lng
		}

		resp, err := d.API.Autocomplete(ctx, apiclient.AutocompleteParams{
			Query:   query,
			Country: country,
			Class:   class,
			Lat:     latPtr,
			Lng:     lngPtr,
			Lang:    lang,
			Limit:   limit,
		})
		if err != nil {
			return nil, GeocodeOutput{}, FormatAPIError("geocode", err)
		}

		compact := shape.ToCompactSlice(resp.Results, false)
		summary := shape.SummarizePlaces(compact, 5)

		out := GeocodeOutput{
			Query:     resp.Query,
			TotalHits: int(resp.TotalHits),
			Places:    compact,
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: summary},
			},
		}, out, nil
	})

	// Tool 2: reverse_geocode
	RegisterTool(s, &mcp.Tool{
		Name:        "reverse_geocode",
		Description: "Reverse geocoding: resolve latitude and longitude coordinates into a structured address, administrative hierarchy, and closest place.",
		Annotations: ReadOnly,
	}, func(ctx context.Context, req *mcp.CallToolRequest, in ReverseGeocodeInput) (*mcp.CallToolResult, ReverseGeocodeOutput, error) {
		if err := validateLatLng(in.Lat, in.Lng); err != nil {
			return nil, ReverseGeocodeOutput{}, err
		}

		radius := in.RadiusMeters
		if radius <= 0 {
			radius = 50000
		} else {
			radius = ClampInt(radius, 1, 100000)
		}

		lang, err := resolveLanguage(in.Lang, d)
		if err != nil {
			return nil, ReverseGeocodeOutput{}, err
		}

		resp, err := d.API.Reverse(ctx, apiclient.ReverseParams{
			Lat:          in.Lat,
			Lng:          in.Lng,
			RadiusMeters: &radius,
			Lang:         lang,
		})
		if err != nil {
			return nil, ReverseGeocodeOutput{}, FormatAPIError("reverse geocode", err)
		}

		var closestCompact *shape.CompactPlace
		if resp.ClosestPlace != nil {
			cp := shape.ToCompact(resp.ClosestPlace, false)
			closestCompact = &cp
		}

		out := ReverseGeocodeOutput{
			Lat:            resp.Lat,
			Lng:            resp.Lng,
			RadiusMeters:   resp.RadiusMeters,
			Address:        resp.Address,
			AdminHierarchy: resp.AdminHierarchy,
			ClosestPlace:   closestCompact,
		}

		summary := summarizeReverseGeocode(out)

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: summary},
			},
		}, out, nil
	})
}

// resolveLanguage validates and resolves the language code against the configured whitelist.
// If lang is empty, it falls back to d.Config.DefaultLang if configured.
func resolveLanguage(lang string, d Deps) (string, error) {
	lang = strings.ToLower(strings.TrimSpace(lang))
	if lang == "" {
		if d.Config != nil && d.Config.DefaultLang != "" {
			lang = strings.ToLower(strings.TrimSpace(d.Config.DefaultLang))
		}
	}
	if lang != "" && d.Taxonomy != nil {
		if !d.Taxonomy.IsValidLanguage(lang) {
			return "", ToolError("unsupported language code %q. Supported languages: %s", lang, strings.Join(d.Taxonomy.GetLanguageCodes(), ", "))
		}
	}
	return lang, nil
}

// validateClass validates and canonicalizes a taxonomy class against the taxonomy cache.
func validateClass(class string, d Deps) (string, error) {
	class = strings.TrimSpace(class)
	if class == "" {
		return "", nil
	}
	if d.Taxonomy != nil {
		canonical, ok := d.Taxonomy.IsValidClass(class)
		if !ok {
			return "", ToolError("unknown class %q. Valid classes: %s. Call list_place_categories for details.", class, strings.Join(d.Taxonomy.GetClassNames(), ", "))
		}
		return canonical, nil
	}
	return class, nil
}

// validateCountry normalizes a country code or country name to a canonical 2-letter ISO 3166-1 alpha-2 code.
func validateCountry(country string) (string, error) {
	country = strings.TrimSpace(country)
	if country == "" {
		return "", nil
	}
	if code, ok := countries.NormalizeCode(country); ok {
		return code, nil
	}
	if len(country) == 2 {
		return strings.ToUpper(country), nil
	}
	return "", ToolError("Unrecognized country %q. Please provide a 2-letter ISO country code (e.g. 'BE') or common country name (e.g. 'Belgium').", country)
}

// validateLatLng validates that lat and lng are within valid geographic bounds.
func validateLatLng(lat, lng float64) error {
	if lat < -90 || lat > 90 {
		return ToolError("lat must be between -90 and 90, got %f", lat)
	}
	if lng < -180 || lng > 180 {
		return ToolError("lng must be between -180 and 180, got %f", lng)
	}
	return nil
}

func summarizeReverseGeocode(out ReverseGeocodeOutput) string {
	var parts []string
	addr := out.Address
	var addrParts []string
	if addr.Landmark != "" {
		addrParts = append(addrParts, addr.Landmark)
	}
	if addr.Neighborhood != "" {
		addrParts = append(addrParts, addr.Neighborhood)
	}
	if addr.City != "" {
		addrParts = append(addrParts, addr.City)
	}
	if addr.Region != "" {
		addrParts = append(addrParts, addr.Region)
	}
	if addr.Country != "" {
		addrParts = append(addrParts, addr.Country)
	}
	if len(addrParts) > 0 {
		parts = append(parts, strings.Join(addrParts, ", "))
	} else if len(out.AdminHierarchy) > 0 {
		parts = append(parts, strings.Join(out.AdminHierarchy, ", "))
	}

	if out.ClosestPlace != nil {
		parts = append(parts, fmt.Sprintf("Closest place: %s", shape.SummarizePlace(*out.ClosestPlace)))
	}

	if len(parts) == 0 {
		return fmt.Sprintf("Coordinates [%.4f, %.4f]: no address or place found within %d meters.", out.Lat, out.Lng, out.RadiusMeters)
	}

	return fmt.Sprintf("Resolved [%.4f, %.4f]: %s", out.Lat, out.Lng, strings.Join(parts, " | "))
}
