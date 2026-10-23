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

// AutocompleteResponse represents the API response for /v1/autocomplete.
type AutocompleteResponse struct {
	Query            string         `json:"query"`
	Lang             string         `json:"lang,omitempty"`
	ProcessingTimeMs int64          `json:"took_ms"`
	TotalHits        int64          `json:"total_hits"`
	Results          []schema.Place `json:"results"`
}

// Autocomplete handles prefix search suggestions with filters, language resolution, and geo-proximity boost.
func (h *Handler) Autocomplete(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	lang := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("lang")))
	country := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("country")))
	class := strings.TrimSpace(r.URL.Query().Get("class"))
	subclass := strings.TrimSpace(r.URL.Query().Get("subclass"))

	limit := 10
	if l := r.URL.Query().Get("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil && val > 0 {
			if val > 100 {
				val = 100
			}
			limit = val
		}
	}

	var filters []string
	if country != "" {
		filters = append(filters, fmt.Sprintf(`country_code = "%s"`, country))
	}
	if class != "" {
		filters = append(filters, fmt.Sprintf(`class = "%s"`, class))
	}
	if subclass != "" {
		filters = append(filters, fmt.Sprintf(`subclass = "%s"`, subclass))
	}

	var sortRules []string
	var userLat, userLng float64
	hasGeo := false
	latStr := r.URL.Query().Get("lat")
	lngStr := r.URL.Query().Get("lng")
	if latStr != "" && lngStr != "" {
		lat, errLat := strconv.ParseFloat(latStr, 64)
		lng, errLng := strconv.ParseFloat(lngStr, 64)
		if errLat == nil && errLng == nil {
			userLat = lat
			userLng = lng
			hasGeo = true
			sortRules = append(sortRules, fmt.Sprintf("_geoPoint(%f, %f):asc", lat, lng))
		}
	}
	sortRules = append(sortRules, "popularity_score:desc")

	filterExpr := ""
	if len(filters) > 0 {
		filterExpr = strings.Join(filters, " AND ")
	}

	searchReq := meili.SearchRequest{
		Q:                     q,
		Filter:                filterExpr,
		Sort:                  sortRules,
		Limit:                 limit,
		AttributesToHighlight: []string{"name_display"},
	}

	result, err := h.meili.SearchPlaces(r.Context(), searchReq)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Search error: "+err.Error())
		return
	}

	// Post-process places: localize name and calculate distance if geo provided
	for i := range result.Hits {
		p := &result.Hits[i]
		if lang != "" && len(p.NamesByLanguage) > 0 {
			if loc, ok := p.NamesByLanguage[lang]; ok && loc != "" {
				p.NameDisplay = loc
			}
		}
		if hasGeo && p.Geo.Lat != 0 && p.Geo.Lng != 0 {
			d := meili.HaversineDistance(userLat, userLng, p.Geo.Lat, p.Geo.Lng)
			p.DistanceMeters = &d
		}
	}

	*r = *appMiddleware.SetResultHits(r, len(result.Hits))

	writeJSON(w, http.StatusOK, AutocompleteResponse{
		Query:            q,
		Lang:             lang,
		ProcessingTimeMs: result.ProcessingTimeMs,
		TotalHits:        result.TotalHits,
		Results:          result.Hits,
	})
}
