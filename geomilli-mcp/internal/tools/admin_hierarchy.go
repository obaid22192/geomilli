package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/geomilli/geomilli-mcp/internal/shape"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ListCountriesInput is empty since list_countries requires no parameters.
type ListCountriesInput struct{}

// CountrySummaryItem represents an indexed country with place counts.
type CountrySummaryItem struct {
	CountryCode string `json:"country_code" jsonschema:"ISO 3166-1 alpha-2 country code (e.g. BE, FR)"`
	CountryName string `json:"country_name" jsonschema:"Official or common name of the country"`
	PlaceCount  int    `json:"place_count" jsonschema:"Number of indexed places and destinations"`
}

// ListCountriesOutput models the structured result for list_countries.
type ListCountriesOutput struct {
	TotalCountries int                  `json:"total_countries" jsonschema:"Total count of indexed countries"`
	Countries      []CountrySummaryItem `json:"countries" jsonschema:"Indexed countries sorted by place count descending"`
}

// ListRegionsInput defines input parameters for list_regions.
type ListRegionsInput struct {
	CountryCode string `json:"country_code" jsonschema:"ISO 3166-1 alpha-2 country code or country name (e.g. 'BE', 'Belgium', 'FR', 'France', 'DE', 'Germany')"`
	Lang        string `json:"lang,omitempty" jsonschema:"Preferred language code for localized region names (e.g. 'en', 'fr')"`
}

// ListRegionsOutput models the structured result for list_regions.
type ListRegionsOutput struct {
	CountryCode string               `json:"country_code" jsonschema:"ISO 3166-1 alpha-2 country code"`
	TotalHits   int                  `json:"total_hits" jsonschema:"Total count of administrative regions/provinces found"`
	Regions     []shape.CompactPlace `json:"regions" jsonschema:"List of administrative regions and provinces"`
}

// ListCitiesInput defines input parameters for list_cities.
type ListCitiesInput struct {
	CountryCode string `json:"country_code" jsonschema:"ISO 3166-1 alpha-2 country code or country name (e.g. 'BE', 'Belgium', 'FR', 'France', 'DE', 'Germany')"`
	Limit       int    `json:"limit,omitempty" jsonschema:"Maximum number of cities to return (default: 20, min: 1, max: 100)"`
	Offset      int    `json:"offset,omitempty" jsonschema:"Pagination offset (default: 0)"`
	Lang        string `json:"lang,omitempty" jsonschema:"Preferred language code for localized city names"`
}

// ListCitiesOutput models the structured result for list_cities.
type ListCitiesOutput struct {
	CountryCode string               `json:"country_code" jsonschema:"ISO 3166-1 alpha-2 country code"`
	TotalHits   int                  `json:"total_hits" jsonschema:"Total cities found in country"`
	Limit       int                  `json:"limit" jsonschema:"Pagination limit applied"`
	Offset      int                  `json:"offset" jsonschema:"Pagination offset applied"`
	Cities      []shape.CompactPlace `json:"cities" jsonschema:"List of cities with coordinates and population"`
}

func registerAdminHierarchy(s *mcp.Server, d Deps) {
	// Tool: list_countries
	RegisterTool(s, &mcp.Tool{
		Name:        "list_countries",
		Description: "List all countries currently indexed in the Geomilli database along with total place counts. Use this to determine which countries have available geospatial data.",
		Annotations: ReadOnly,
	}, func(ctx context.Context, req *mcp.CallToolRequest, in ListCountriesInput) (*mcp.CallToolResult, ListCountriesOutput, error) {
		res, err := d.API.ListCountries(ctx)
		if err != nil {
			return nil, ListCountriesOutput{}, FormatAPIError("list countries", err)
		}

		countries := make([]CountrySummaryItem, len(res.Countries))
		var topSummaries []string
		for i, c := range res.Countries {
			countries[i] = CountrySummaryItem{
				CountryCode: c.CountryCode,
				CountryName: c.CountryName,
				PlaceCount:  c.PlaceCount,
			}
			if i < 5 {
				topSummaries = append(topSummaries, fmt.Sprintf("%s (%s: %d places)", c.CountryName, c.CountryCode, c.PlaceCount))
			}
		}

		out := ListCountriesOutput{
			TotalCountries: res.TotalCountries,
			Countries:      countries,
		}

		summary := fmt.Sprintf("Found %d indexed countries in Geomilli database.", res.TotalCountries)
		if len(topSummaries) > 0 {
			summary += " Top countries: " + strings.Join(topSummaries, ", ")
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: summary},
			},
		}, out, nil
	})

	// Tool: list_regions
	RegisterTool(s, &mcp.Tool{
		Name:        "list_regions",
		Description: "List administrative regions, provinces, or states for a specific country by ISO 3166-1 alpha-2 country code (e.g. 'BE', 'FR', 'DE').",
		Annotations: ReadOnly,
	}, func(ctx context.Context, req *mcp.CallToolRequest, in ListRegionsInput) (*mcp.CallToolResult, ListRegionsOutput, error) {
		if strings.TrimSpace(in.CountryCode) == "" {
			return nil, ListRegionsOutput{}, ToolError("Country code is required (e.g. 'BE', 'FR')")
		}
		countryCode, err := validateCountry(in.CountryCode)
		if err != nil {
			return nil, ListRegionsOutput{}, err
		}

		lang := in.Lang
		if lang == "" && d.Config != nil && d.Config.DefaultLang != "" {
			lang = d.Config.DefaultLang
		}
		if lang != "" && !d.Taxonomy.IsValidLanguage(lang) {
			return nil, ListRegionsOutput{}, ToolError(
				"Unsupported language code %q. Supported languages: %s",
				lang, strings.Join(d.Taxonomy.GetLanguageCodes(), ", "),
			)
		}

		res, err := d.API.ListRegions(ctx, countryCode, lang)
		if err != nil {
			return nil, ListRegionsOutput{}, FormatAPIError(fmt.Sprintf("list regions for country %q", countryCode), err)
		}

		compact := shape.ToCompactSlice(res.Regions, false)
		summary := fmt.Sprintf("Found %d region(s) in country %s.", len(compact), countryCode)
		if len(compact) > 0 {
			var names []string
			for i := 0; i < len(compact) && i < 5; i++ {
				names = append(names, compact[i].Name)
			}
			summary += " Regions: " + strings.Join(names, ", ")
			if len(compact) > 5 {
				summary += fmt.Sprintf(" (and %d more)", len(compact)-5)
			}
		}

		out := ListRegionsOutput{
			CountryCode: countryCode,
			TotalHits:   int(res.TotalHits),
			Regions:     compact,
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: summary},
			},
		}, out, nil
	})

	// Tool: list_cities
	RegisterTool(s, &mcp.Tool{
		Name:        "list_cities",
		Description: "List major cities and towns for a specific country by ISO 3166-1 alpha-2 country code, ordered by population and popularity score.",
		Annotations: ReadOnly,
	}, func(ctx context.Context, req *mcp.CallToolRequest, in ListCitiesInput) (*mcp.CallToolResult, ListCitiesOutput, error) {
		if strings.TrimSpace(in.CountryCode) == "" {
			return nil, ListCitiesOutput{}, ToolError("Country code is required (e.g. 'BE', 'FR')")
		}
		countryCode, err := validateCountry(in.CountryCode)
		if err != nil {
			return nil, ListCitiesOutput{}, err
		}

		limit := in.Limit
		if limit <= 0 {
			limit = 20
		}
		limit = ClampInt(limit, 1, 100)

		offset := in.Offset
		if offset < 0 {
			offset = 0
		}

		lang := in.Lang
		if lang == "" && d.Config != nil && d.Config.DefaultLang != "" {
			lang = d.Config.DefaultLang
		}
		if lang != "" && !d.Taxonomy.IsValidLanguage(lang) {
			return nil, ListCitiesOutput{}, ToolError(
				"Unsupported language code %q. Supported languages: %s",
				lang, strings.Join(d.Taxonomy.GetLanguageCodes(), ", "),
			)
		}

		res, err := d.API.ListCities(ctx, countryCode, limit, offset, lang)
		if err != nil {
			return nil, ListCitiesOutput{}, FormatAPIError(fmt.Sprintf("list cities for country %q", countryCode), err)
		}

		compact := shape.ToCompactSlice(res.Cities, false)
		summary := fmt.Sprintf("Found %d cities in country %s (showing %d):", res.TotalHits, countryCode, len(compact))
		if len(compact) > 0 {
			var topCities []string
			for i := 0; i < len(compact) && i < 5; i++ {
				c := compact[i]
				if c.Population > 0 {
					topCities = append(topCities, fmt.Sprintf("%s (pop: %d)", c.Name, c.Population))
				} else {
					topCities = append(topCities, c.Name)
				}
			}
			summary += " " + strings.Join(topCities, ", ")
			if len(compact) > 5 {
				summary += fmt.Sprintf(" (and %d more)", len(compact)-5)
			}
		} else {
			summary = fmt.Sprintf("No cities found for country %s.", countryCode)
		}

		out := ListCitiesOutput{
			CountryCode: countryCode,
			TotalHits:   int(res.TotalHits),
			Limit:       limit,
			Offset:      offset,
			Cities:      compact,
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: summary},
			},
		}, out, nil
	})
}
