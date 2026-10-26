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

// FindAirportsInput defines parameters for the find_airports tool.
type FindAirportsInput struct {
	Query   string   `json:"query,omitempty" jsonschema:"Free-text search query (e.g. 'Brussels', 'Heathrow', 'Charles de Gaulle')"`
	Country string   `json:"country,omitempty" jsonschema:"ISO 3166-1 alpha-2 country code or country name (e.g. 'BE', 'Belgium', 'FR', 'France', 'US')"`
	IATA    string   `json:"iata,omitempty" jsonschema:"3-letter IATA airport code filter (e.g. 'BRU', 'LHR')"`
	ICAO    string   `json:"icao,omitempty" jsonschema:"4-letter ICAO airport code filter (e.g. 'EBBR', 'EGLL')"`
	Lat     *float64 `json:"lat,omitempty" jsonschema:"Latitude coordinate (-90 to 90) for proximity-based ranking"`
	Lng     *float64 `json:"lng,omitempty" jsonschema:"Longitude coordinate (-180 to 180) for proximity-based ranking"`
	Limit   int      `json:"limit,omitempty" jsonschema:"Maximum number of airports to return (default: 10, min: 1, max: 25)"`
	Lang    string   `json:"lang,omitempty" jsonschema:"Preferred language code for localized airport and place names (e.g. 'en', 'fr')"`
}

// FindAirportsOutput models the structured result for find_airports.
type FindAirportsOutput struct {
	TotalHits int                  `json:"total_hits" jsonschema:"Total matching airports found"`
	Airports  []shape.CompactPlace `json:"airports" jsonschema:"List of matching airports"`
}

// GetAirportInput defines parameters for the get_airport tool.
type GetAirportInput struct {
	Code string `json:"code" jsonschema:"3-letter IATA (e.g. 'BRU', 'JFK') or 4-letter ICAO (e.g. 'EBBR', 'KJFK') airport code"`
	Lang string `json:"lang,omitempty" jsonschema:"Preferred language code for localized airport name (e.g. 'en', 'fr')"`
}

// GetAirportOutput models the structured result for get_airport.
type GetAirportOutput struct {
	Airport shape.CompactPlace `json:"airport" jsonschema:"Airport details"`
}

func registerAviation(s *mcp.Server, d Deps) {
	// Tool 1: find_airports
	RegisterTool(s, &mcp.Tool{
		Name:        "find_airports",
		Description: "Search for airports by name, IATA code, ICAO code, country, or proximity to geographic coordinates. Returns matching airports with coordinates, codes, and location details.",
		Annotations: ReadOnlyOpenWorld,
	}, func(ctx context.Context, req *mcp.CallToolRequest, in FindAirportsInput) (*mcp.CallToolResult, FindAirportsOutput, error) {
		limit := in.Limit
		if limit <= 0 {
			limit = 10
		}
		limit = ClampInt(limit, 1, 25)

		country, err := validateCountry(in.Country)
		if err != nil {
			return nil, FindAirportsOutput{}, err
		}

		query := strings.TrimSpace(in.Query)
		if country == "" && query != "" {
			if code, ok := countries.NormalizeCode(query); ok {
				country = code
				query = ""
			} else if cleaned, code, found := countries.ExtractFromQuery(query); found {
				country = code
				if strings.EqualFold(cleaned, "airports") || strings.EqualFold(cleaned, "airport") {
					query = ""
				} else {
					query = cleaned
				}
			}
		}

		iata := strings.ToUpper(strings.TrimSpace(in.IATA))
		if iata != "" && len(iata) != 3 {
			return nil, FindAirportsOutput{}, ToolError("IATA airport code must be a 3-letter code, got %q", in.IATA)
		}

		icao := strings.ToUpper(strings.TrimSpace(in.ICAO))
		if icao != "" && len(icao) != 4 {
			return nil, FindAirportsOutput{}, ToolError("ICAO airport code must be a 4-letter code, got %q", in.ICAO)
		}

		if (in.Lat != nil && in.Lng == nil) || (in.Lat == nil && in.Lng != nil) {
			return nil, FindAirportsOutput{}, ToolError("Both lat and lng coordinates must be provided together for proximity search")
		}

		if in.Lat != nil && in.Lng != nil {
			if *in.Lat < -90 || *in.Lat > 90 {
				return nil, FindAirportsOutput{}, ToolError("Latitude must be between -90 and 90, got %f", *in.Lat)
			}
			if *in.Lng < -180 || *in.Lng > 180 {
				return nil, FindAirportsOutput{}, ToolError("Longitude must be between -180 and 180, got %f", *in.Lng)
			}
		}

		lang := in.Lang
		if lang == "" && d.Config != nil && d.Config.DefaultLang != "" {
			lang = d.Config.DefaultLang
		}
		if lang != "" && !d.Taxonomy.IsValidLanguage(lang) {
			return nil, FindAirportsOutput{}, ToolError(
				"Unsupported language code %q. Supported languages: %s",
				lang, strings.Join(d.Taxonomy.GetLanguageCodes(), ", "),
			)
		}

		res, err := d.API.SearchAirports(ctx, apiclient.SearchAirportsParams{
			Query:   query,
			Country: country,
			IATA:    iata,
			ICAO:    icao,
			Lat:     in.Lat,
			Lng:     in.Lng,
			Limit:   limit,
			Lang:    lang,
		})
		if err != nil {
			return nil, FindAirportsOutput{}, FormatAPIError("search airports", err)
		}

		compact := shape.ToCompactSlice(res.Results, false)
		summary := shape.SummarizePlaces(compact, 5)

		out := FindAirportsOutput{
			TotalHits: int(res.TotalHits),
			Airports:  compact,
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: summary},
			},
		}, out, nil
	})

	// Tool 2: get_airport
	RegisterTool(s, &mcp.Tool{
		Name:        "get_airport",
		Description: "Retrieve detailed information for a specific airport by its 3-letter IATA code (e.g. 'BRU', 'JFK') or 4-letter ICAO code (e.g. 'EBBR', 'KJFK').",
		Annotations: ReadOnly,
	}, func(ctx context.Context, req *mcp.CallToolRequest, in GetAirportInput) (*mcp.CallToolResult, GetAirportOutput, error) {
		code := strings.ToUpper(strings.TrimSpace(in.Code))
		if code == "" {
			return nil, GetAirportOutput{}, ToolError("Airport code is required (e.g. 'BRU', 'EBBR')")
		}
		if len(code) != 3 && len(code) != 4 {
			return nil, GetAirportOutput{}, ToolError("Airport code must be a 3-letter IATA code or 4-letter ICAO code, got %q", in.Code)
		}

		lang := in.Lang
		if lang == "" && d.Config != nil && d.Config.DefaultLang != "" {
			lang = d.Config.DefaultLang
		}
		if lang != "" && !d.Taxonomy.IsValidLanguage(lang) {
			return nil, GetAirportOutput{}, ToolError(
				"Unsupported language code %q. Supported languages: %s",
				lang, strings.Join(d.Taxonomy.GetLanguageCodes(), ", "),
			)
		}

		place, err := d.API.GetAirportByCode(ctx, code, lang)
		if err != nil {
			return nil, GetAirportOutput{}, FormatAPIError(fmt.Sprintf("get airport for code %q", code), err)
		}

		compact := shape.ToCompact(place, false)
		summary := shape.SummarizePlace(compact)

		out := GetAirportOutput{
			Airport: compact,
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: summary},
			},
		}, out, nil
	})
}
