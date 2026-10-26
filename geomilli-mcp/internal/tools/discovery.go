package tools

import (
	"context"
	"fmt"
	"strings"

	"github.com/geomilli/geomilli-mcp/internal/apiclient"
	"github.com/geomilli/geomilli-mcp/internal/shape"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// SearchPlacesInput defines input parameters for full-text search with structured filters.
type SearchPlacesInput struct {
	Query      string `json:"query,omitempty" jsonschema:"Full-text search keywords or place name"`
	Class      string `json:"class,omitempty" jsonschema:"Filter by taxonomy class (e.g. 'Restaurant', 'Hotel', 'Attraction')"`
	Subclass   string `json:"subclass,omitempty" jsonschema:"Filter by taxonomy subclass (e.g. 'museum', 'monument', 'cafe')"`
	Country    string `json:"country,omitempty" jsonschema:"Filter by 2-letter ISO country code (e.g. 'BE', 'FR')"`
	Cuisine    string `json:"cuisine,omitempty" jsonschema:"Filter by cuisine type (e.g. 'italian', 'french', 'japanese')"`
	StarRating int    `json:"star_rating,omitempty" jsonschema:"Filter by minimum star rating for hotels (1 to 5)"`
	Lang       string `json:"lang,omitempty" jsonschema:"Preferred language code for place names (e.g. 'en', 'fr')"`
	Limit      int    `json:"limit,omitempty" jsonschema:"Maximum places to return (default: 10, max: 25)"`
	Offset     int    `json:"offset,omitempty" jsonschema:"Pagination offset (default: 0)"`
}

// SearchPlacesOutput models the structured result for search_places.
type SearchPlacesOutput struct {
	Query     string               `json:"query,omitempty" jsonschema:"Query string searched"`
	TotalHits int                  `json:"total_hits" jsonschema:"Total matching places found"`
	Limit     int                  `json:"limit" jsonschema:"Effective limit of places returned"`
	Offset    int                  `json:"offset" jsonschema:"Offset applied to results"`
	Places    []shape.CompactPlace `json:"places" jsonschema:"List of matching places"`
}

// FindNearbyInput defines input parameters for finding places near coordinates.
type FindNearbyInput struct {
	Lat          float64 `json:"lat" jsonschema:"Latitude coordinate (-90 to 90)"`
	Lng          float64 `json:"lng" jsonschema:"Longitude coordinate (-180 to 180)"`
	RadiusMeters int     `json:"radius_meters,omitempty" jsonschema:"Search radius in meters (1 to 50000, default: 5000)"`
	Class        string  `json:"class,omitempty" jsonschema:"Filter by taxonomy class (e.g. 'Restaurant', 'Hotel', 'Attraction')"`
	Subclass     string  `json:"subclass,omitempty" jsonschema:"Filter by taxonomy subclass (e.g. 'museum', 'cafe')"`
	Cuisine      string  `json:"cuisine,omitempty" jsonschema:"Filter by cuisine type (e.g. 'italian', 'french')"`
	MinStars     int     `json:"min_stars,omitempty" jsonschema:"Minimum star rating for accommodation (1 to 5)"`
	Lang         string  `json:"lang,omitempty" jsonschema:"Optional language code for place names (e.g. 'en', 'fr')"`
	Limit        int     `json:"limit,omitempty" jsonschema:"Maximum places to return (default: 10, max: 25)"`
}

// FindNearbyOutput models the structured result for find_nearby.
type FindNearbyOutput struct {
	Lat          float64              `json:"lat" jsonschema:"Center latitude"`
	Lng          float64              `json:"lng" jsonschema:"Center longitude"`
	RadiusMeters int                  `json:"radius_meters" jsonschema:"Effective search radius in meters"`
	TotalHits    int                  `json:"total_hits" jsonschema:"Total matching places found"`
	Places       []shape.CompactPlace `json:"places" jsonschema:"List of nearby places sorted by distance"`
}

// PlacesInAreaInput defines input parameters for bounding box queries.
type PlacesInAreaInput struct {
	MinLat float64 `json:"min_lat" jsonschema:"Southern boundary latitude (-90 to 90)"`
	MinLng float64 `json:"min_lng" jsonschema:"Western boundary longitude (-180 to 180)"`
	MaxLat float64 `json:"max_lat" jsonschema:"Northern boundary latitude (-90 to 90)"`
	MaxLng float64 `json:"max_lng" jsonschema:"Eastern boundary longitude (-180 to 180)"`
	Class  string  `json:"class,omitempty" jsonschema:"Optional filter by taxonomy class (e.g. 'Restaurant', 'Hotel')"`
	Lang   string  `json:"lang,omitempty" jsonschema:"Optional language code for place names (e.g. 'en', 'fr')"`
	Limit  int     `json:"limit,omitempty" jsonschema:"Maximum places to return (default: 20, max: 25)"`
}

// PlacesInAreaOutput models the structured result for places_in_area.
type PlacesInAreaOutput struct {
	MinLat    float64              `json:"min_lat" jsonschema:"Southern boundary latitude"`
	MinLng    float64              `json:"min_lng" jsonschema:"Western boundary longitude"`
	MaxLat    float64              `json:"max_lat" jsonschema:"Northern boundary latitude"`
	MaxLng    float64              `json:"max_lng" jsonschema:"Eastern boundary longitude"`
	TotalHits int                  `json:"total_hits" jsonschema:"Total matching places found in bounding box"`
	Places    []shape.CompactPlace `json:"places" jsonschema:"Places within the bounding box"`
}

// GetPlaceInput defines input parameters for retrieving single place details.
type GetPlaceInput struct {
	ID            string `json:"id,omitempty" jsonschema:"Geomilli place ID (e.g. 'osm_n123', 'place_456')"`
	WikidataID    string `json:"wikidata_id,omitempty" jsonschema:"Wikidata QID identifier (e.g. 'Q207')"`
	IncludeNearby bool   `json:"include_nearby,omitempty" jsonschema:"Whether to include nearby points of interest"`
	Lang          string `json:"lang,omitempty" jsonschema:"Optional language code for localized place names (e.g. 'en', 'fr')"`
	Verbose       bool   `json:"verbose,omitempty" jsonschema:"If true, returns full entity with all aliases, localized names, and metadata; if false, returns token-efficient compact place"`
}

// GetPlaceOutput models the structured result for get_place.
type GetPlaceOutput struct {
	Place        shape.CompactPlace   `json:"place" jsonschema:"Place details (compact view when verbose=false; includes all alternate names, languages, and source when verbose=true)"`
	FullEntity   *apiclient.Place     `json:"full_entity,omitempty" jsonschema:"Full place entity when verbose=true"`
	NearbyPlaces []shape.CompactPlace `json:"nearby_places,omitempty" jsonschema:"Nearby places if include_nearby was requested"`
}

func registerDiscovery(s *mcp.Server, d Deps) {
	// Tool 3: search_places
	RegisterTool(s, &mcp.Tool{
		Name:        "search_places",
		Description: "Search places, destinations, and points of interest using full-text search and structured filters (class, subclass, country, cuisine, star rating).",
		Annotations: ReadOnlyOpenWorld,
	}, func(ctx context.Context, req *mcp.CallToolRequest, in SearchPlacesInput) (*mcp.CallToolResult, SearchPlacesOutput, error) {
		limit := in.Limit
		if limit <= 0 {
			limit = 10
		} else {
			limit = ClampInt(limit, 1, 25)
		}

		offset := in.Offset
		if offset < 0 {
			offset = 0
		}

		if in.StarRating < 0 || in.StarRating > 5 {
			return nil, SearchPlacesOutput{}, ToolError("star_rating must be between 1 and 5, got %d", in.StarRating)
		}

		country, err := validateCountry(in.Country)
		if err != nil {
			return nil, SearchPlacesOutput{}, err
		}

		class, err := validateClass(in.Class, d)
		if err != nil {
			return nil, SearchPlacesOutput{}, err
		}

		lang, err := resolveLanguage(in.Lang, d)
		if err != nil {
			return nil, SearchPlacesOutput{}, err
		}

		filterExpr := buildSearchFilter(class, strings.TrimSpace(in.Subclass), country, strings.TrimSpace(in.Cuisine), in.StarRating)

		resp, err := d.API.Search(ctx, apiclient.SearchParams{
			Query:  strings.TrimSpace(in.Query),
			Filter: filterExpr,
			Lang:   lang,
			Limit:  limit,
			Offset: offset,
		})
		if err != nil {
			return nil, SearchPlacesOutput{}, FormatAPIError("search places", err)
		}

		compact := shape.ToCompactSlice(resp.Results, false)
		summary := shape.SummarizePlaces(compact, 5)

		out := SearchPlacesOutput{
			Query:     resp.Query,
			TotalHits: int(resp.TotalHits),
			Limit:     resp.Limit,
			Offset:    resp.Offset,
			Places:    compact,
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: summary},
			},
		}, out, nil
	})

	// Tool 4: find_nearby
	RegisterTool(s, &mcp.Tool{
		Name:        "find_nearby",
		Description: "Find points of interest (POIs) and destinations near coordinates within a specified radius (meters).",
		Annotations: ReadOnlyOpenWorld,
	}, func(ctx context.Context, req *mcp.CallToolRequest, in FindNearbyInput) (*mcp.CallToolResult, FindNearbyOutput, error) {
		if err := validateLatLng(in.Lat, in.Lng); err != nil {
			return nil, FindNearbyOutput{}, err
		}

		radius := in.RadiusMeters
		if radius <= 0 {
			radius = 5000
		} else {
			radius = ClampInt(radius, 1, 50000)
		}

		limit := in.Limit
		if limit <= 0 {
			limit = 10
		} else {
			limit = ClampInt(limit, 1, 25)
		}

		if in.MinStars < 0 || in.MinStars > 5 {
			return nil, FindNearbyOutput{}, ToolError("min_stars must be between 1 and 5, got %d", in.MinStars)
		}

		class, err := validateClass(in.Class, d)
		if err != nil {
			return nil, FindNearbyOutput{}, err
		}

		lang, err := resolveLanguage(in.Lang, d)
		if err != nil {
			return nil, FindNearbyOutput{}, err
		}

		resp, err := d.API.Nearby(ctx, apiclient.NearbyParams{
			Lat:          in.Lat,
			Lng:          in.Lng,
			RadiusMeters: radius,
			Class:        class,
			Subclass:     strings.TrimSpace(in.Subclass),
			Cuisine:      strings.TrimSpace(in.Cuisine),
			MinStars:     in.MinStars,
			Lang:         lang,
			Limit:        limit,
		})
		if err != nil {
			return nil, FindNearbyOutput{}, FormatAPIError("find nearby places", err)
		}

		compact := shape.ToCompactSlice(resp.Results, false)
		summary := shape.SummarizePlaces(compact, 5)

		out := FindNearbyOutput{
			Lat:          resp.Lat,
			Lng:          resp.Lng,
			RadiusMeters: resp.RadiusMeters,
			TotalHits:    int(resp.TotalHits),
			Places:       compact,
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: summary},
			},
		}, out, nil
	})

	// Tool 5: places_in_area
	RegisterTool(s, &mcp.Tool{
		Name:        "places_in_area",
		Description: "Find places and destinations within a geographic bounding box defined by minimum and maximum latitude and longitude coordinates.",
		Annotations: ReadOnlyOpenWorld,
	}, func(ctx context.Context, req *mcp.CallToolRequest, in PlacesInAreaInput) (*mcp.CallToolResult, PlacesInAreaOutput, error) {
		if in.MinLat < -90 || in.MinLat > 90 || in.MaxLat < -90 || in.MaxLat > 90 {
			return nil, PlacesInAreaOutput{}, ToolError("latitudes must be between -90 and 90, got min_lat=%f, max_lat=%f", in.MinLat, in.MaxLat)
		}
		if in.MinLng < -180 || in.MinLng > 180 || in.MaxLng < -180 || in.MaxLng > 180 {
			return nil, PlacesInAreaOutput{}, ToolError("longitudes must be between -180 and 180, got min_lng=%f, max_lng=%f", in.MinLng, in.MaxLng)
		}
		if in.MinLat >= in.MaxLat {
			return nil, PlacesInAreaOutput{}, ToolError("min_lat (%f) must be strictly less than max_lat (%f)", in.MinLat, in.MaxLat)
		}
		if in.MinLng >= in.MaxLng {
			return nil, PlacesInAreaOutput{}, ToolError("min_lng (%f) must be strictly less than max_lng (%f)", in.MinLng, in.MaxLng)
		}

		limit := in.Limit
		if limit <= 0 {
			limit = 20
		} else {
			limit = ClampInt(limit, 1, 25)
		}

		class, err := validateClass(in.Class, d)
		if err != nil {
			return nil, PlacesInAreaOutput{}, err
		}

		lang, err := resolveLanguage(in.Lang, d)
		if err != nil {
			return nil, PlacesInAreaOutput{}, err
		}

		resp, err := d.API.BBox(ctx, apiclient.BBoxParams{
			MinLat: in.MinLat,
			MinLng: in.MinLng,
			MaxLat: in.MaxLat,
			MaxLng: in.MaxLng,
			Class:  class,
			Lang:   lang,
			Limit:  limit,
		})
		if err != nil {
			return nil, PlacesInAreaOutput{}, FormatAPIError("find places in area", err)
		}

		compact := shape.ToCompactSlice(resp.Results, false)
		summary := shape.SummarizePlaces(compact, 5)

		out := PlacesInAreaOutput{
			MinLat:    resp.MinLat,
			MinLng:    resp.MinLng,
			MaxLat:    resp.MaxLat,
			MaxLng:    resp.MaxLng,
			TotalHits: int(resp.TotalHits),
			Places:    compact,
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: summary},
			},
		}, out, nil
	})

	// Tool 6: get_place
	RegisterTool(s, &mcp.Tool{
		Name:        "get_place",
		Description: "Retrieve detailed metadata for a single place by its Geomilli ID or Wikidata QID.",
		Annotations: ReadOnly,
	}, func(ctx context.Context, req *mcp.CallToolRequest, in GetPlaceInput) (*mcp.CallToolResult, GetPlaceOutput, error) {
		id := strings.TrimSpace(in.ID)
		qid := strings.TrimSpace(in.WikidataID)

		if (id == "" && qid == "") || (id != "" && qid != "") {
			return nil, GetPlaceOutput{}, ToolError("exactly one of 'id' or 'wikidata_id' must be provided")
		}

		lang, err := resolveLanguage(in.Lang, d)
		if err != nil {
			return nil, GetPlaceOutput{}, err
		}

		var resp *apiclient.PlaceDetailsResponse
		if id != "" {
			resp, err = d.API.GetPlaceByID(ctx, id, lang, in.IncludeNearby)
			if err != nil {
				return nil, GetPlaceOutput{}, FormatAPIError(fmt.Sprintf("get place by ID %q", id), err)
			}
		} else {
			qid = strings.ToUpper(qid)
			resp, err = d.API.GetPlaceByWikidataID(ctx, qid, lang, in.IncludeNearby)
			if err != nil {
				return nil, GetPlaceOutput{}, FormatAPIError(fmt.Sprintf("get place by Wikidata ID %q", qid), err)
			}
		}

		compact := shape.ToCompact(&resp.Place, in.Verbose)
		nearby := shape.ToCompactSlice(resp.NearbyPlaces, false)

		out := GetPlaceOutput{
			Place:        compact,
			NearbyPlaces: nearby,
		}
		if in.Verbose {
			out.FullEntity = &resp.Place
		}

		summary := shape.SummarizePlace(compact)
		if len(nearby) > 0 {
			summary += fmt.Sprintf("\n\nNearby places (%d):\n%s", len(nearby), shape.SummarizePlaces(nearby, 5))
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: summary},
			},
		}, out, nil
	})
}

// escapeFilterValue escapes backslashes and double quotes for Meilisearch filter strings.
func escapeFilterValue(val string) string {
	s := strings.ReplaceAll(val, `\`, `\\`)
	return strings.ReplaceAll(s, `"`, `\"`)
}

// buildSearchFilter constructs a Meilisearch filter string safely from structured parameters.
func buildSearchFilter(class, subclass, country, cuisine string, starRating int) string {
	var filters []string
	if class != "" {
		filters = append(filters, fmt.Sprintf(`class = "%s"`, escapeFilterValue(class)))
	}
	if subclass != "" {
		filters = append(filters, fmt.Sprintf(`subclass = "%s"`, escapeFilterValue(subclass)))
	}
	if country != "" {
		filters = append(filters, fmt.Sprintf(`country_code = "%s"`, escapeFilterValue(country)))
	}
	if cuisine != "" {
		filters = append(filters, fmt.Sprintf(`cuisine = "%s"`, escapeFilterValue(cuisine)))
	}
	if starRating > 0 {
		filters = append(filters, fmt.Sprintf("star_rating >= %d", starRating))
	}
	return strings.Join(filters, " AND ")
}
