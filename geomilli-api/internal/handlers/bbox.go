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
	minLatStr := strings.TrimSpace(r.URL.Query().Get("min_lat"))
	minLngStr := strings.TrimSpace(r.URL.Query().Get("min_lng"))
	maxLatStr := strings.TrimSpace(r.URL.Query().Get("max_lat"))
	maxLngStr := strings.TrimSpace(r.URL.Query().Get("max_lng"))

	if minLatStr == "" || minLngStr == "" || maxLatStr == "" || maxLngStr == "" {
		writeError(w, http.StatusBadRequest, "Parameters 'min_lat', 'min_lng', 'max_lat', and 'max_lng' are all required")
		return
	}

	minLat, errMinLat := strconv.ParseFloat(minLatStr, 64)
	minLng, errMinLng := strconv.ParseFloat(minLngStr, 64)
	maxLat, errMaxLat := strconv.ParseFloat(maxLatStr, 64)
	maxLng, errMaxLng := strconv.ParseFloat(maxLngStr, 64)

	if errMinLat != nil || errMinLng != nil || errMaxLat != nil || errMaxLng != nil {
		writeError(w, http.StatusBadRequest, "Invalid bounding box coordinates")
		return
	}

	limit := 100
	if l := r.URL.Query().Get("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil && val > 0 {
			if val > 500 {
				val = 500
			}
			limit = val
		}
	}

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
		writeError(w, http.StatusInternalServerError, "Bounding box search error: "+err.Error())
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
