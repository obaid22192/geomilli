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
	lat, lng, err := parseLatLng(r)
	if err != nil {
		if pErr, ok := err.(*ParamError); ok {
			writeError(w, http.StatusBadRequest, pErr.Code, pErr.Message)
			return
		}
		writeError(w, http.StatusBadRequest, ErrInvalidParameter, err.Error())
		return
	}

	radius := parseRadius(r, DefaultNearbyRadius, MaxNearbyRadius)
	limit := parseLimit(r, DefaultNearbyLimit, MaxNearbyLimit)
	offset := parseOffset(r)

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
		writeError(w, http.StatusBadGateway, ErrUpstreamUnavailable, "Nearby query error: "+err.Error())
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
