package tools

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// ListPlaceCategoriesInput defines parameters for the list_place_categories tool (empty struct).
type ListPlaceCategoriesInput struct{}

// PlaceCategoryItem represents a canonical taxonomy class definition for LLMs.
type PlaceCategoryItem struct {
	Class      string   `json:"class" jsonschema:"Canonical place class name (e.g. 'Restaurant', 'Hotel')"`
	Layer      string   `json:"layer" jsonschema:"Taxonomy layer ('poi' or 'admin')"`
	BaseScore  float64  `json:"base_score" jsonschema:"Base ranking score for the class"`
	Subclasses []string `json:"subclasses" jsonschema:"List of known subclasses for this class"`
}

// ListPlaceCategoriesOutput models the structured result for list_place_categories.
type ListPlaceCategoriesOutput struct {
	TotalClasses int                 `json:"total_classes" jsonschema:"Total number of canonical classes"`
	Categories   []PlaceCategoryItem `json:"categories" jsonschema:"List of canonical place categories"`
}

// MeasureDistanceInput defines parameters for the measure_distance tool.
type MeasureDistanceInput struct {
	FromLat     *float64 `json:"from_lat,omitempty" jsonschema:"Starting latitude coordinate (-90 to 90)"`
	FromLng     *float64 `json:"from_lng,omitempty" jsonschema:"Starting longitude coordinate (-180 to 180)"`
	FromPlaceID string   `json:"from_place_id,omitempty" jsonschema:"Starting place ID (resolved via API if coordinates omitted)"`

	ToLat     *float64 `json:"to_lat,omitempty" jsonschema:"Destination latitude coordinate (-90 to 90)"`
	ToLng     *float64 `json:"to_lng,omitempty" jsonschema:"Destination longitude coordinate (-180 to 180)"`
	ToPlaceID string   `json:"to_place_id,omitempty" jsonschema:"Destination place ID (resolved via API if coordinates omitted)"`
}

// GeoCoordinate represents latitude and longitude coordinates.
type GeoCoordinate struct {
	Lat float64 `json:"lat" jsonschema:"Latitude coordinate"`
	Lng float64 `json:"lng" jsonschema:"Longitude coordinate"`
}

// DistancePoint represents a resolved starting or destination endpoint.
type DistancePoint struct {
	PlaceID string        `json:"place_id,omitempty" jsonschema:"Place ID if resolved"`
	Name    string        `json:"name,omitempty" jsonschema:"Place name if resolved"`
	Point   GeoCoordinate `json:"point" jsonschema:"Geographical coordinates"`
}

// MeasureDistanceOutput models the structured result for measure_distance.
type MeasureDistanceOutput struct {
	From           DistancePoint `json:"from" jsonschema:"Starting location"`
	To             DistancePoint `json:"to" jsonschema:"Destination location"`
	DistanceMeters float64       `json:"distance_meters" jsonschema:"Great-circle distance in meters (rounded to 1 decimal)"`
	DistanceKm     float64       `json:"distance_km" jsonschema:"Great-circle distance in kilometers (rounded to 2 decimals)"`
	DistanceMiles  float64       `json:"distance_miles" jsonschema:"Great-circle distance in miles (rounded to 2 decimals)"`
}

// haversineDistance calculates the great-circle distance between two points in meters.
func haversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadiusMeters = 6371000.0

	dLat := (lat2 - lat1) * (math.Pi / 180.0)
	dLon := (lon2 - lon1) * (math.Pi / 180.0)

	rLat1 := lat1 * (math.Pi / 180.0)
	rLat2 := lat2 * (math.Pi / 180.0)

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(rLat1)*math.Cos(rLat2)*math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadiusMeters * c
}

func registerReference(s *mcp.Server, d Deps) {
	// Tool 1: list_place_categories
	mcp.AddTool(s, &mcp.Tool{
		Name:        "list_place_categories",
		Description: "List the 19 canonical place categories and their taxonomy definitions, including layer (poi/admin), base ranking score, and known subclasses. Call this tool to discover valid category and class filters for search and discovery.",
		Annotations: ReadOnly,
	}, func(ctx context.Context, req *mcp.CallToolRequest, in ListPlaceCategoriesInput) (*mcp.CallToolResult, ListPlaceCategoriesOutput, error) {
		_ = d.Taxonomy.RefreshIfNeeded(ctx)
		tax := d.Taxonomy.GetTaxonomyConfig()
		if tax == nil {
			var err error
			tax, err = d.API.GetTaxonomy(ctx)
			if err != nil {
				return nil, ListPlaceCategoriesOutput{}, FormatAPIError("get taxonomy", err)
			}
		}

		var categories []PlaceCategoryItem
		for _, cd := range tax.Classes {
			cName := strings.TrimSpace(cd.Name)
			if cName == "" {
				continue
			}

			subs := d.Taxonomy.GetSubclasses(cName)
			if len(subs) == 0 {
				subSet := make(map[string]struct{})
				for _, m := range cd.Match {
					for _, v := range m.Values {
						val := strings.TrimSpace(v)
						if val != "" && val != "*" {
							subSet[val] = struct{}{}
						}
					}
				}
				subs = make([]string, 0, len(subSet))
				for s := range subSet {
					subs = append(subs, s)
				}
				sort.Strings(subs)
			}
			if subs == nil {
				subs = []string{}
			}

			categories = append(categories, PlaceCategoryItem{
				Class:      cName,
				Layer:      cd.Layer,
				BaseScore:  cd.BaseScore,
				Subclasses: subs,
			})
		}

		sort.Slice(categories, func(i, j int) bool {
			return categories[i].Class < categories[j].Class
		})

		var names []string
		for _, c := range categories {
			names = append(names, c.Class)
		}
		summary := fmt.Sprintf("Taxonomy contains %d canonical place categories: %s.", len(categories), strings.Join(names, ", "))

		out := ListPlaceCategoriesOutput{
			TotalClasses: len(categories),
			Categories:   categories,
		}

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: summary},
			},
		}, out, nil
	})

	// Tool 2: measure_distance
	mcp.AddTool(s, &mcp.Tool{
		Name:        "measure_distance",
		Description: "Calculate the great-circle distance between two geographic locations using the Haversine formula. Coordinates can be provided directly (lat/lng) or resolved automatically from Geomilli place IDs.",
		Annotations: ReadOnly,
	}, func(ctx context.Context, req *mcp.CallToolRequest, in MeasureDistanceInput) (*mcp.CallToolResult, MeasureDistanceOutput, error) {
		fromPlaceID := strings.TrimSpace(in.FromPlaceID)
		var fromPoint DistancePoint
		if fromPlaceID != "" {
			p, err := d.API.GetPlaceByID(ctx, fromPlaceID, "", false)
			if err != nil {
				return nil, MeasureDistanceOutput{}, FormatAPIError(fmt.Sprintf("resolve starting place ID %q", fromPlaceID), err)
			}
			fromPoint = DistancePoint{
				PlaceID: p.ID,
				Name:    p.NameDisplay,
				Point:   GeoCoordinate{Lat: p.Geo.Lat, Lng: p.Geo.Lng},
			}
		} else if in.FromLat != nil && in.FromLng != nil {
			fromPoint = DistancePoint{
				Point: GeoCoordinate{Lat: *in.FromLat, Lng: *in.FromLng},
			}
		} else {
			return nil, MeasureDistanceOutput{}, ToolError("Starting location requires either 'from_place_id' or both 'from_lat' and 'from_lng'")
		}

		toPlaceID := strings.TrimSpace(in.ToPlaceID)
		var toPoint DistancePoint
		if toPlaceID != "" {
			p, err := d.API.GetPlaceByID(ctx, toPlaceID, "", false)
			if err != nil {
				return nil, MeasureDistanceOutput{}, FormatAPIError(fmt.Sprintf("resolve destination place ID %q", toPlaceID), err)
			}
			toPoint = DistancePoint{
				PlaceID: p.ID,
				Name:    p.NameDisplay,
				Point:   GeoCoordinate{Lat: p.Geo.Lat, Lng: p.Geo.Lng},
			}
		} else if in.ToLat != nil && in.ToLng != nil {
			toPoint = DistancePoint{
				Point: GeoCoordinate{Lat: *in.ToLat, Lng: *in.ToLng},
			}
		} else {
			return nil, MeasureDistanceOutput{}, ToolError("Destination location requires either 'to_place_id' or both 'to_lat' and 'to_lng'")
		}

		// Validate coordinate boundaries
		if fromPoint.Point.Lat < -90 || fromPoint.Point.Lat > 90 {
			return nil, MeasureDistanceOutput{}, ToolError("Starting latitude must be between -90 and 90, got %f", fromPoint.Point.Lat)
		}
		if fromPoint.Point.Lng < -180 || fromPoint.Point.Lng > 180 {
			return nil, MeasureDistanceOutput{}, ToolError("Starting longitude must be between -180 and 180, got %f", fromPoint.Point.Lng)
		}
		if toPoint.Point.Lat < -90 || toPoint.Point.Lat > 90 {
			return nil, MeasureDistanceOutput{}, ToolError("Destination latitude must be between -90 and 90, got %f", toPoint.Point.Lat)
		}
		if toPoint.Point.Lng < -180 || toPoint.Point.Lng > 180 {
			return nil, MeasureDistanceOutput{}, ToolError("Destination longitude must be between -180 and 180, got %f", toPoint.Point.Lng)
		}

		meters := haversineDistance(fromPoint.Point.Lat, fromPoint.Point.Lng, toPoint.Point.Lat, toPoint.Point.Lng)
		metersRounded := math.Round(meters*10) / 10
		kmRounded := math.Round((meters/1000.0)*100) / 100
		milesRounded := math.Round((meters/1609.344)*100) / 100

		out := MeasureDistanceOutput{
			From:           fromPoint,
			To:             toPoint,
			DistanceMeters: metersRounded,
			DistanceKm:     kmRounded,
			DistanceMiles:  milesRounded,
		}

		fromLabel := fmt.Sprintf("(%.4f, %.4f)", fromPoint.Point.Lat, fromPoint.Point.Lng)
		if fromPoint.Name != "" {
			fromLabel = fmt.Sprintf("%s %s", fromPoint.Name, fromLabel)
		}
		toLabel := fmt.Sprintf("(%.4f, %.4f)", toPoint.Point.Lat, toPoint.Point.Lng)
		if toPoint.Name != "" {
			toLabel = fmt.Sprintf("%s %s", toPoint.Name, toLabel)
		}

		summary := fmt.Sprintf("Distance from %s to %s is %.2f km (%.2f miles, %.1f meters).", fromLabel, toLabel, kmRounded, milesRounded, metersRounded)

		return &mcp.CallToolResult{
			Content: []mcp.Content{
				&mcp.TextContent{Text: summary},
			},
		}, out, nil
	})
}
