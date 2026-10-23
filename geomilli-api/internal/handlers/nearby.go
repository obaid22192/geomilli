package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/geomilli/geomilli-api/internal/meili"
	appMiddleware "github.com/geomilli/geomilli-api/internal/middleware"
	"github.com/geomilli/geomilli-api/internal/schema"
)

// NearbyResponse represents the response for /v1/nearby.
type NearbyResponse struct {
	Lat              float64        `json:"lat"`
	Lng              float64        `json:"lng"`
	RadiusMeters     int            `json:"radius_meters"`
	ProcessingTimeMs int64          `json:"took_ms"`
	Limit            int            `json:"limit"`
	Offset           int            `json:"offset"`
	TotalHits        int64          `json:"total_hits"`
	Results          []schema.Place `json:"results"`
}

// Nearby queries POIs and places within a geo-radius.
func (h *Handler) Nearby(w http.ResponseWriter, r *http.Request) {
	lang := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("lang")))
	latStr := strings.TrimSpace(r.URL.Query().Get("lat"))
	lngStr := strings.TrimSpace(r.URL.Query().Get("lng"))

	if latStr == "" || lngStr == "" {
		writeError(w, http.StatusBadRequest, "Both 'lat' and 'lng' query parameters are required")
		return
	}

	lat, errLat := strconv.ParseFloat(latStr, 64)
	lng, errLng := strconv.ParseFloat(lngStr, 64)
	if errLat != nil || errLng != nil || lat < -90 || lat > 90 || lng < -180 || lng > 180 {
		writeError(w, http.StatusBadRequest, "Invalid lat or lng coordinate values")
		return
	}

	radius := 5000
	if rStr := r.URL.Query().Get("radius_meters"); rStr != "" {
		if val, err := strconv.Atoi(rStr); err == nil && val > 0 {
			if val > 100000 {
				val = 100000
			}
			radius = val
		}
	}

	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil && val > 0 {
			if val > 100 {
				val = 100
			}
			limit = val
		}
	}

	offset := 0
	if o := r.URL.Query().Get("offset"); o != "" {
		if val, err := strconv.Atoi(o); err == nil && val >= 0 {
			offset = val
		}
	}

	var filters []string
	filters = append(filters, fmt.Sprintf("_geoRadius(%f, %f, %d)", lat, lng, radius))

	if class := strings.TrimSpace(r.URL.Query().Get("class")); class != "" {
		filters = append(filters, fmt.Sprintf(`class = "%s"`, class))
	}
	if subclass := strings.TrimSpace(r.URL.Query().Get("subclass")); subclass != "" {
		filters = append(filters, fmt.Sprintf(`subclass = "%s"`, subclass))
	}
	if cuisine := strings.TrimSpace(r.URL.Query().Get("cuisine")); cuisine != "" {
		filters = append(filters, fmt.Sprintf(`cuisine = "%s"`, cuisine))
	}
	if minStars := r.URL.Query().Get("min_stars"); minStars != "" {
		if stars, err := strconv.Atoi(minStars); err == nil && stars > 0 {
			filters = append(filters, fmt.Sprintf("star_rating >= %d", stars))
		}
	}

	sortRules := []string{
		fmt.Sprintf("_geoPoint(%f, %f):asc", lat, lng),
		"popularity_score:desc",
	}

	searchReq := meili.SearchRequest{
		Filter: strings.Join(filters, " AND "),
		Sort:   sortRules,
		Limit:  limit,
		Offset: offset,
	}

	result, err := h.meili.SearchPlaces(r.Context(), searchReq)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Nearby query error: "+err.Error())
		return
	}

	// Calculate exact distance for each hit
	for i := range result.Hits {
		p := &result.Hits[i]
		if p.Geo.Lat != 0 && p.Geo.Lng != 0 {
			d := meili.HaversineDistance(lat, lng, p.Geo.Lat, p.Geo.Lng)
			p.DistanceMeters = &d
		}
	}

	localizePlaces(result.Hits, lang)
	*r = *appMiddleware.SetResultHits(r, len(result.Hits))

	writeJSON(w, http.StatusOK, NearbyResponse{
		Lat:              lat,
		Lng:              lng,
		RadiusMeters:     radius,
		ProcessingTimeMs: result.ProcessingTimeMs,
		Limit:            limit,
		Offset:           offset,
		TotalHits:        result.TotalHits,
		Results:          result.Hits,
	})
}
