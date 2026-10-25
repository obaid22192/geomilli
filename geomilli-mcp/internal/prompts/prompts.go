package prompts

import (
	"context"
	"fmt"
	"strings"

	"github.com/geomilli/geomilli-mcp/internal/tools"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Register registers all Geomilli workflow prompt templates on the provided MCP server.
func Register(s *mcp.Server, d tools.Deps) {
	// 1. explore_area
	s.AddPrompt(&mcp.Prompt{
		Name:        "explore_area",
		Title:       "Explore Area & POIs",
		Description: "Workflow prompt to geocode a place and discover points of interest, dining, attractions, and cultural landmarks.",
		Arguments: []*mcp.PromptArgument{
			{
				Name:        "place",
				Title:       "Target Place",
				Description: "Target destination, neighborhood, landmark, or city to explore (e.g. 'Montmartre, Paris', 'Atomium', 'Shibuya')",
				Required:    true,
			},
			{
				Name:        "interests",
				Title:       "Interests or Categories",
				Description: "Optional focus interests or categories (e.g. 'museums, cafes, historical architecture')",
				Required:    false,
			},
		},
	}, func(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		var place, interests string
		if req.Params != nil && req.Params.Arguments != nil {
			place = strings.TrimSpace(req.Params.Arguments["place"])
			interests = strings.TrimSpace(req.Params.Arguments["interests"])
		}

		if place == "" {
			return nil, fmt.Errorf("argument 'place' is required")
		}

		interestInstruction := "Search for top landmarks, cultural sights, parks, and dining options in the surrounding area."
		if interests != "" {
			interestInstruction = fmt.Sprintf("Focus specifically on categories matching the user's interests: %q. Call `find_nearby` or `search_places` with relevant category filters.", interests)
		}

		promptText := fmt.Sprintf(`You are an expert geospatial tour guide and local explorer assisting a user with exploring: %q.

Follow this step-by-step workflow:
1. Forward Geocode: Call `+"`geocode`"+` with query=%q to resolve the destination's exact geographic coordinates (latitude and longitude) and enclosing administrative hierarchy.
2. Discover Points of Interest: Using the coordinates retrieved from step 1, call `+"`find_nearby`"+` to search for prominent destinations and points of interest.
   - %s
3. Deep Dive: For any notable places discovered, call `+"`get_place`"+` to retrieve full details, localized names, and background facts.
4. Final Recommendations: Synthesize a structured, vivid area exploration guide including:
   - Location overview and administrative context (neighborhood, city, region)
   - Highlighted points of interest with distances, categories, and key attributes
   - Curated recommendations tailored to the traveler's interests`, place, place, interestInstruction)

		return &mcp.GetPromptResult{
			Description: fmt.Sprintf("Explore points of interest and destinations around %s", place),
			Messages: []*mcp.PromptMessage{
				{
					Role:    "user",
					Content: &mcp.TextContent{Text: promptText},
				},
			},
		}, nil
	})

	// 2. describe_location
	s.AddPrompt(&mcp.Prompt{
		Name:        "describe_location",
		Title:       "Describe Geographic Location",
		Description: "Workflow prompt to reverse-geocode a coordinate point, discover surrounding landmarks, and describe the geographical context.",
		Arguments: []*mcp.PromptArgument{
			{
				Name:        "lat",
				Title:       "Latitude",
				Description: "Latitude coordinate in decimal degrees (-90 to 90)",
				Required:    true,
			},
			{
				Name:        "lng",
				Title:       "Longitude",
				Description: "Longitude coordinate in decimal degrees (-180 to 180)",
				Required:    true,
			},
		},
	}, func(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		var lat, lng string
		if req.Params != nil && req.Params.Arguments != nil {
			lat = strings.TrimSpace(req.Params.Arguments["lat"])
			lng = strings.TrimSpace(req.Params.Arguments["lng"])
		}

		if lat == "" || lng == "" {
			return nil, fmt.Errorf("arguments 'lat' and 'lng' are required")
		}

		promptText := fmt.Sprintf(`You are a geospatial analysis assistant. Your task is to investigate and thoroughly describe the geographic location at coordinates (%s, %s).

Follow this step-by-step workflow:
1. Reverse Geocode: Call `+"`reverse_geocode`"+` with lat=%s and lng=%s to determine the exact address, administrative hierarchy (neighborhood, municipality/city, state/region, country), and the closest identified place.
2. Identify Surrounding Landmarks: Call `+"`find_nearby`"+` with lat=%s, lng=%s, and radius_meters=5000 to discover notable landmarks, attractions, transit hubs, and commercial activity in the vicinity.
3. Geographic Narrative: Provide a vivid, comprehensive description of this location including:
   - Full administrative context and jurisdiction
   - Immediate vicinity and closest landmarks with exact distances
   - Landscape and functional character (e.g. dense metropolitan center, suburban residential, coastal, rural, or transit corridor)`, lat, lng, lat, lng, lat, lng)

		return &mcp.GetPromptResult{
			Description: fmt.Sprintf("Describe location at coordinates (%s, %s)", lat, lng),
			Messages: []*mcp.PromptMessage{
				{
					Role:    "user",
					Content: &mcp.TextContent{Text: promptText},
				},
			},
		}, nil
	})

	// 3. plan_airport_transfer
	s.AddPrompt(&mcp.Prompt{
		Name:        "plan_airport_transfer",
		Title:       "Plan Airport Transfer & Transit",
		Description: "Workflow prompt to find airports serving a city, measure distances, and evaluate transfer logistics.",
		Arguments: []*mcp.PromptArgument{
			{
				Name:        "city",
				Title:       "City Name",
				Description: "Target city or metropolitan area (e.g. 'Brussels', 'Tokyo', 'San Francisco')",
				Required:    true,
			},
		},
	}, func(ctx context.Context, req *mcp.GetPromptRequest) (*mcp.GetPromptResult, error) {
		var city string
		if req.Params != nil && req.Params.Arguments != nil {
			city = strings.TrimSpace(req.Params.Arguments["city"])
		}

		if city == "" {
			return nil, fmt.Errorf("argument 'city' is required")
		}

		promptText := fmt.Sprintf(`You are an aviation and travel transit planning assistant. Your task is to locate airports serving %q and evaluate transfer options and distances.

Follow this step-by-step workflow:
1. Geocode City Center: Call `+"`geocode`"+` with query=%q to identify the exact coordinates and administrative region of the city center.
2. Discover Serving Airports: Call `+"`find_airports`"+` with query=%q to find all international, domestic, and regional airports serving the metropolitan area.
3. Evaluate Distances & Codes: For each discovered airport:
   - Retrieve its IATA and ICAO codes.
   - Use `+"`measure_distance`"+` to determine the direct distance between the city center and the airport.
4. Transfer Recommendations: Synthesize a clear comparison and transfer plan:
   - List of airports sorted by proximity, noting primary international hubs vs secondary/budget airports
   - Distance and orientation from the city center
   - Practical guidance for travelers choosing among the available airport options`, city, city, city)

		return &mcp.GetPromptResult{
			Description: fmt.Sprintf("Plan airport transit and options for %s", city),
			Messages: []*mcp.PromptMessage{
				{
					Role:    "user",
					Content: &mcp.TextContent{Text: promptText},
				},
			},
		}, nil
	})
}
