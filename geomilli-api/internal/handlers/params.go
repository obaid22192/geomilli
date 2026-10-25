package handlers

import (
	"net/http"
	"strconv"
	"strings"
)

// Named constants for radius defaults and limits per endpoint.
const (
	DefaultNearbyRadius = 5000
	MaxNearbyRadius     = 100000

	DefaultReverseRadius = 50000
	MaxReverseRadius     = 100000
)

// Named constants for limit defaults and limits per endpoint.
const (
	DefaultAutocompleteLimit = 10
	MaxAutocompleteLimit     = 100

	DefaultSearchLimit = 20
	MaxSearchLimit     = 100

	DefaultNearbyLimit = 20
	MaxNearbyLimit     = 100

	DefaultBBoxLimit = 100
	MaxBBoxLimit     = 500

	DefaultAirportsLimit = 20
	MaxAirportsLimit     = 100

	DefaultCitiesLimit = 50
	MaxCitiesLimit     = 200

	DefaultOffset = 0
)

// ParamError encapsulates an error encountered while parsing query parameters.
type ParamError struct {
	Code    ErrorCode
	Message string
}

func (e *ParamError) Error() string {
	return e.Message
}

// parseLatLng extracts and validates required 'lat' and 'lng' query parameters.
// Missing parameters return ErrMissingParameter; invalid values return ErrInvalidParameter.
func parseLatLng(r *http.Request) (float64, float64, error) {
	latStr := strings.TrimSpace(r.URL.Query().Get("lat"))
	lngStr := strings.TrimSpace(r.URL.Query().Get("lng"))

	if latStr == "" || lngStr == "" {
		return 0, 0, &ParamError{
			Code:    ErrMissingParameter,
			Message: "Both 'lat' and 'lng' query parameters are required",
		}
	}

	lat, errLat := strconv.ParseFloat(latStr, 64)
	lng, errLng := strconv.ParseFloat(lngStr, 64)
	if errLat != nil || errLng != nil || lat < -90 || lat > 90 || lng < -180 || lng > 180 {
		return 0, 0, &ParamError{
			Code:    ErrInvalidParameter,
			Message: "Invalid lat or lng coordinate values",
		}
	}

	return lat, lng, nil
}

// parseOptionalLatLng extracts and validates optional 'lat' and 'lng' query parameters.
// Returns (lat, lng, true) if both are present and valid, otherwise (0, 0, false).
func parseOptionalLatLng(r *http.Request) (float64, float64, bool) {
	latStr := strings.TrimSpace(r.URL.Query().Get("lat"))
	lngStr := strings.TrimSpace(r.URL.Query().Get("lng"))
	if latStr == "" || lngStr == "" {
		return 0, 0, false
	}

	lat, errLat := strconv.ParseFloat(latStr, 64)
	lng, errLng := strconv.ParseFloat(lngStr, 64)
	if errLat != nil || errLng != nil || lat < -90 || lat > 90 || lng < -180 || lng > 180 {
		return 0, 0, false
	}

	return lat, lng, true
}

// parseRadius parses radius in meters from 'radius_meters' (or fallback 'radius').
// Clamps values to [1, maxRadius]. Non-numeric or absent input returns defaultRadius.
func parseRadius(r *http.Request, defaultRadius, maxRadius int) int {
	rStr := strings.TrimSpace(r.URL.Query().Get("radius_meters"))
	if rStr == "" {
		rStr = strings.TrimSpace(r.URL.Query().Get("radius"))
	}
	if rStr == "" {
		return defaultRadius
	}

	val, err := strconv.Atoi(rStr)
	if err != nil {
		return defaultRadius
	}
	if val < 1 {
		return 1
	}
	if val > maxRadius {
		return maxRadius
	}
	return val
}

// parseLimit parses the 'limit' query parameter with clamping to [1, maxLimit].
// Returns defaultLimit if parameter is missing or non-numeric.
func parseLimit(r *http.Request, defaultLimit, maxLimit int) int {
	l := strings.TrimSpace(r.URL.Query().Get("limit"))
	if l == "" {
		return defaultLimit
	}

	val, err := strconv.Atoi(l)
	if err != nil {
		return defaultLimit
	}
	if val < 1 {
		return 1
	}
	if val > maxLimit {
		return maxLimit
	}
	return val
}

// parseOffset parses the 'offset' query parameter, clamped to >= 0.
// Returns DefaultOffset (0) if missing, negative, or non-numeric.
func parseOffset(r *http.Request) int {
	o := strings.TrimSpace(r.URL.Query().Get("offset"))
	if o == "" {
		return DefaultOffset
	}

	val, err := strconv.Atoi(o)
	if err != nil || val < 0 {
		return DefaultOffset
	}
	return val
}

// parseBBox parses and validates bounding box parameters: min_lat, min_lng, max_lat, max_lng.
func parseBBox(r *http.Request) (minLat, minLng, maxLat, maxLng float64, err error) {
	minLatStr := strings.TrimSpace(r.URL.Query().Get("min_lat"))
	minLngStr := strings.TrimSpace(r.URL.Query().Get("min_lng"))
	maxLatStr := strings.TrimSpace(r.URL.Query().Get("max_lat"))
	maxLngStr := strings.TrimSpace(r.URL.Query().Get("max_lng"))

	if minLatStr == "" || minLngStr == "" || maxLatStr == "" || maxLngStr == "" {
		return 0, 0, 0, 0, &ParamError{
			Code:    ErrMissingParameter,
			Message: "Parameters 'min_lat', 'min_lng', 'max_lat', and 'max_lng' are all required",
		}
	}

	minLat, errMinLat := strconv.ParseFloat(minLatStr, 64)
	minLng, errMinLng := strconv.ParseFloat(minLngStr, 64)
	maxLat, errMaxLat := strconv.ParseFloat(maxLatStr, 64)
	maxLng, errMaxLng := strconv.ParseFloat(maxLngStr, 64)

	if errMinLat != nil || errMinLng != nil || errMaxLat != nil || errMaxLng != nil ||
		minLat < -90 || minLat > 90 || maxLat < -90 || maxLat > 90 ||
		minLng < -180 || minLng > 180 || maxLng < -180 || maxLng > 180 {
		return 0, 0, 0, 0, &ParamError{
			Code:    ErrInvalidParameter,
			Message: "Invalid bounding box coordinates",
		}
	}

	return minLat, minLng, maxLat, maxLng, nil
}
