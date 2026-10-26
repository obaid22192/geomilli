package contract

import (
	"context"
	"math"
	"strings"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// TestInputValidation_TableDriven runs comprehensive edge cases and adversarial inputs
// across Geomilli MCP tools to verify that inputs are properly sanitized, clamped, or rejected
// with actionable LLM error messages, and that the server never panics.
func TestInputValidation_TableDriven(t *testing.T) {
	clientSession, cleanup := setupInMemoryServer(t, true)
	defer cleanup()

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	tests := []struct {
		name          string
		tool          string
		args          map[string]any
		expectIsError bool
		errorContains string
	}{
		// 1. geocode validation
		{
			name:          "geocode_empty_query",
			tool:          "geocode",
			args:          map[string]any{"query": ""},
			expectIsError: true,
			errorContains: "query parameter is required",
		},
		{
			name:          "geocode_whitespace_query",
			tool:          "geocode",
			args:          map[string]any{"query": "   \t\n  "},
			expectIsError: true,
			errorContains: "query parameter is required",
		},
		{
			name:          "geocode_invalid_country",
			tool:          "geocode",
			args:          map[string]any{"query": "Brussels", "country": "Atlantis"},
			expectIsError: true,
			errorContains: "Unrecognized country \"Atlantis\"",
		},
		{
			name:          "geocode_invalid_country_code_length_1",
			tool:          "geocode",
			args:          map[string]any{"query": "Brussels", "country": "B"},
			expectIsError: true,
			errorContains: "Unrecognized country \"B\"",
		},
		{
			name:          "geocode_lat_without_lng",
			tool:          "geocode",
			args:          map[string]any{"query": "Brussels", "lat": 50.85},
			expectIsError: true,
			errorContains: "both lat and lng must be provided together",
		},
		{
			name:          "geocode_lat_out_of_range_high",
			tool:          "geocode",
			args:          map[string]any{"query": "Brussels", "lat": 95.0, "lng": 4.35},
			expectIsError: true,
			errorContains: "lat must be between -90 and 90",
		},
		{
			name:          "geocode_lng_out_of_range_low",
			tool:          "geocode",
			args:          map[string]any{"query": "Brussels", "lat": 50.85, "lng": -190.0},
			expectIsError: true,
			errorContains: "lng must be between -180 and 180",
		},
		{
			name:          "geocode_unknown_class",
			tool:          "geocode",
			args:          map[string]any{"query": "Brussels", "class": "FlyingSaucer"},
			expectIsError: true,
			errorContains: "unknown class \"FlyingSaucer\"",
		},
		{
			name:          "geocode_unsupported_language",
			tool:          "geocode",
			args:          map[string]any{"query": "Brussels", "lang": "klingon"},
			expectIsError: true,
			errorContains: "unsupported language code \"klingon\"",
		},
		{
			name: "geocode_filter_injection_attempt",
			tool: "geocode",
			args: map[string]any{
				"query": "Brussels",
				"class": "Restaurant\" OR 1=1 --",
			},
			expectIsError: true,
			errorContains: "unknown class",
		},

		// 2. reverse_geocode validation
		{
			name:          "reverse_geocode_lat_out_of_range",
			tool:          "reverse_geocode",
			args:          map[string]any{"lat": 91.0, "lng": 4.35},
			expectIsError: true,
			errorContains: "lat must be between -90 and 90",
		},
		{
			name:          "reverse_geocode_lng_out_of_range",
			tool:          "reverse_geocode",
			args:          map[string]any{"lat": 50.85, "lng": 185.0},
			expectIsError: true,
			errorContains: "lng must be between -180 and 180",
		},

		// 3. places_in_area validation
		{
			name: "places_in_area_min_lat_greater_than_max",
			tool: "places_in_area",
			args: map[string]any{
				"min_lat": 51.0, "max_lat": 50.0,
				"min_lng": 4.0, "max_lng": 5.0,
			},
			expectIsError: true,
			errorContains: "min_lat (51.000000) must be strictly less than max_lat (50.000000)",
		},
		{
			name: "places_in_area_min_lng_greater_than_max",
			tool: "places_in_area",
			args: map[string]any{
				"min_lat": 50.0, "max_lat": 51.0,
				"min_lng": 5.0, "max_lng": 4.0,
			},
			expectIsError: true,
			errorContains: "min_lng (5.000000) must be strictly less than max_lng (4.000000)",
		},
		{
			name: "places_in_area_lat_out_of_range",
			tool: "places_in_area",
			args: map[string]any{
				"min_lat": -95.0, "max_lat": 50.0,
				"min_lng": 4.0, "max_lng": 5.0,
			},
			expectIsError: true,
			errorContains: "latitudes must be between -90 and 90",
		},

		// 4. get_place validation
		{
			name:          "get_place_missing_both_id_and_wikidata",
			tool:          "get_place",
			args:          map[string]any{},
			expectIsError: true,
			errorContains: "exactly one of 'id' or 'wikidata_id' must be provided",
		},
		{
			name:          "get_place_both_id_and_wikidata_provided",
			tool:          "get_place",
			args:          map[string]any{"id": "osm_n123", "wikidata_id": "Q207"},
			expectIsError: true,
			errorContains: "exactly one of 'id' or 'wikidata_id' must be provided",
		},

		// 5. get_airport validation
		{
			name:          "get_airport_empty_code",
			tool:          "get_airport",
			args:          map[string]any{"code": ""},
			expectIsError: true,
			errorContains: "Airport code is required",
		},
		{
			name:          "get_airport_code_too_short",
			tool:          "get_airport",
			args:          map[string]any{"code": "BR"},
			expectIsError: true,
			errorContains: "must be a 3-letter IATA code or 4-letter ICAO code",
		},
		{
			name:          "get_airport_code_too_long",
			tool:          "get_airport",
			args:          map[string]any{"code": "BRUSSELS"},
			expectIsError: true,
			errorContains: "must be a 3-letter IATA code or 4-letter ICAO code",
		},

		// 6. admin hierarchy validation
		{
			name:          "list_regions_empty_country",
			tool:          "list_regions",
			args:          map[string]any{"country_code": ""},
			expectIsError: true,
			errorContains: "Country code is required",
		},
		{
			name:          "list_regions_invalid_country",
			tool:          "list_regions",
			args:          map[string]any{"country_code": "Atlantis"},
			expectIsError: true,
			errorContains: "Unrecognized country \"Atlantis\"",
		},
		{
			name:          "list_cities_empty_country",
			tool:          "list_cities",
			args:          map[string]any{"country_code": ""},
			expectIsError: true,
			errorContains: "Country code is required",
		},

		// 7. measure_distance validation
		{
			name:          "measure_distance_missing_both_points",
			tool:          "measure_distance",
			args:          map[string]any{},
			expectIsError: true,
			errorContains: "Starting location requires either 'from_place_id' or both 'from_lat' and 'from_lng'",
		},
		{
			name:          "measure_distance_missing_to_point",
			tool:          "measure_distance",
			args:          map[string]any{"from_lat": 50.85, "from_lng": 4.35},
			expectIsError: true,
			errorContains: "Destination location requires either 'to_place_id' or both 'to_lat' and 'to_lng'",
		},
		{
			name: "measure_distance_from_lat_out_of_range",
			tool: "measure_distance",
			args: map[string]any{
				"from_lat": 95.0, "from_lng": 4.35,
				"to_lat": 48.85, "to_lng": 2.35,
			},
			expectIsError: true,
			errorContains: "Starting latitude must be between -90 and 90",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			res, err := clientSession.CallTool(ctx, &mcp.CallToolParams{
				Name:      tc.tool,
				Arguments: tc.args,
			})

			// Protocol-level error
			if err != nil {
				if !tc.expectIsError {
					t.Fatalf("unexpected protocol error: %v", err)
				}
				return
			}

			// Tool-level error (IsError: true with actionable text message)
			if tc.expectIsError {
				if !res.IsError {
					t.Errorf("expected tool error for input %+v, but got success result: %+v", tc.args, res)
					return
				}
				if tc.errorContains != "" {
					var contentText string
					for _, c := range res.Content {
						if tc, ok := c.(*mcp.TextContent); ok {
							contentText += tc.Text + " "
						}
					}
					if !strings.Contains(contentText, tc.errorContains) {
						t.Errorf("expected error message to contain %q, got: %q", tc.errorContains, contentText)
					}
				}
			} else {
				if res.IsError {
					t.Errorf("unexpected tool error: %+v", res)
				}
			}
		})
	}
}

// FuzzToolInputs runs fuzz testing on tool inputs to ensure the server never panics
// on random, malformed, or hostile byte strings and boundary floats.
func FuzzToolInputs(f *testing.F) {
	// Seed corpus with interesting strings and injection candidates
	seeds := []string{
		"Brussels",
		"",
		"   ",
		"A",
		"BE",
		"BEL",
		"Q207",
		"osm_n123456",
		"Restaurant\" OR 1=1 --",
		"'; DROP TABLE places; --",
		"<script>alert(1)</script>",
		"../../../../etc/passwd",
		"Café & Restaurant 🍺",
		"日本語",
		"العربية",
		strings.Repeat("A", 1000),
		"\x00\x01\x02\xff\xfe",
		"nan",
		"inf",
		"-inf",
	}

	for _, s := range seeds {
		f.Add(s, float64(50.85), float64(4.35), 10)
	}

	f.Fuzz(func(t *testing.T, s string, lat float64, lng float64, limit int) {
		// Filter out NaN/Inf floats which are invalid JSON
		if math.IsNaN(lat) || math.IsInf(lat, 0) || math.IsNaN(lng) || math.IsInf(lng, 0) {
			return
		}

		clientSession, cleanup := setupInMemoryServer(t, true)
		defer cleanup()

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		// Call geocode
		_, _ = clientSession.CallTool(ctx, &mcp.CallToolParams{
			Name: "geocode",
			Arguments: map[string]any{
				"query": s,
				"lat":   lat,
				"lng":   lng,
				"limit": limit,
			},
		})

		// Call reverse_geocode
		_, _ = clientSession.CallTool(ctx, &mcp.CallToolParams{
			Name: "reverse_geocode",
			Arguments: map[string]any{
				"lat": lat,
				"lng": lng,
			},
		})

		// Call get_airport
		_, _ = clientSession.CallTool(ctx, &mcp.CallToolParams{
			Name: "get_airport",
			Arguments: map[string]any{
				"code": s,
			},
		})
	})
}
