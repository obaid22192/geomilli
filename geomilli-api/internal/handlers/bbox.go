package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/geomilli/geomilli-api/internal/meili"
	appMiddleware "github.com/geomilli/geomilli-api/internal/middleware"
	"github.com/geomilli/geomilli-api/internal/schema"
)

// BBoxResponse represents the response for /v1/bbox.
type BBoxResponse struct {
	MinLat           float64        `json:"min_lat"`
	MinLng           float64        `json:"min_lng"`
	MaxLat           float64        `json:"max_lat"`
	MaxLng           float64        `json:"max_lng"`
	ProcessingTimeMs int64          `json:"took_ms"`
	TotalHits        int64          `json:"total_hits"`
	Results          []schema.Place `json:"results"`
}

// BBox queries places within a map bounding box viewport.
func (h *Handler) BBox(w http.ResponseWriter, r *http.Request) {
	lang := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("lang")))
	minLat, minLng, maxLat, maxLng, err := parseBBox(r)
	if err != nil {
		if pErr, ok := err.(*ParamError); ok {
			writeError(w, http.StatusBadRequest, pErr.Code, pErr.Message)
			return
		}
		writeError(w, http.StatusBadRequest, ErrInvalidParameter, err.Error())
		return
	}

	limit := parseLimit(r, DefaultBBoxLimit, MaxBBoxLimit)

	// Meilisearch expects: _geoBoundingBox([top_left_lat, top_left_lng], [bottom_right_lat, bottom_right_lng])
	var filters []string
	filters = append(filters, fmt.Sprintf("_geoBoundingBox([%f, %f], [%f, %f])", maxLat, minLng, minLat, maxLng))

	if class := strings.TrimSpace(r.URL.Query().Get("class")); class != "" {
		filters = append(filters, fmt.Sprintf(`class = "%s"`, class))
	}

	searchReq := meili.SearchRequest{
		Filter: strings.Join(filters, " AND "),
		Sort:   []string{"popularity_score:desc"},
		Limit:  limit,
	}

	result, err := h.meili.SearchPlaces(r.Context(), searchReq)
	if err != nil {
		writeError(w, http.StatusBadGateway, ErrUpstreamUnavailable, "Bounding box search error: "+err.Error())
		return
	}

	localizePlaces(result.Hits, lang)
	*r = *appMiddleware.SetResultHits(r, len(result.Hits))

	writeJSON(w, http.StatusOK, BBoxResponse{
		MinLat:           minLat,
		MinLng:           minLng,
		MaxLat:           maxLat,
		MaxLng:           maxLng,
		ProcessingTimeMs: result.ProcessingTimeMs,
		TotalHits:        result.TotalHits,
		Results:          result.Hits,
	})
}
