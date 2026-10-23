package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/geomilli/geomilli-api/internal/meili"
	appMiddleware "github.com/geomilli/geomilli-api/internal/middleware"
	"github.com/geomilli/geomilli-api/internal/schema"
	"github.com/go-chi/chi/v5"
)

// AirportsResponse models search response for /v1/airports.
type AirportsResponse struct {
	Query            string         `json:"query,omitempty"`
	ProcessingTimeMs int64          `json:"took_ms"`
	TotalHits        int64          `json:"total_hits"`
	Results          []schema.Place `json:"results"`
}

// SearchAirports searches airports by text query, country, or proximity.
func (h *Handler) SearchAirports(w http.ResponseWriter, r *http.Request) {
	lang := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("lang")))
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	country := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("country")))

	limit := 20
	if l := r.URL.Query().Get("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil && val > 0 {
			if val > 100 {
				val = 100
			}
			limit = val
		}
	}

	filters := []string{`class = "Airport"`}
	if country != "" {
		filters = append(filters, fmt.Sprintf(`country_code = "%s"`, country))
	}
	if code := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("code"))); code != "" {
		filters = append(filters, fmt.Sprintf(`(iata_code = "%s" OR icao_code = "%s")`, code, code))
	}
	if iata := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("iata"))); iata != "" {
		filters = append(filters, fmt.Sprintf(`iata_code = "%s"`, iata))
	}
	if icao := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("icao"))); icao != "" {
		filters = append(filters, fmt.Sprintf(`icao_code = "%s"`, icao))
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

	searchReq := meili.SearchRequest{
		Q:                     q,
		Filter:                strings.Join(filters, " AND "),
		Sort:                  sortRules,
		Limit:                 limit,
		AttributesToHighlight: []string{"name_display"},
	}

	result, err := h.meili.SearchPlaces(r.Context(), searchReq)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Search airports error: "+err.Error())
		return
	}

	if hasGeo {
		for i := range result.Hits {
			p := &result.Hits[i]
			if p.Geo.Lat != 0 && p.Geo.Lng != 0 {
				d := meili.HaversineDistance(userLat, userLng, p.Geo.Lat, p.Geo.Lng)
				p.DistanceMeters = &d
			}
		}
	}

	localizePlaces(result.Hits, lang)
	*r = *appMiddleware.SetResultHits(r, len(result.Hits))

	writeJSON(w, http.StatusOK, AirportsResponse{
		Query:            q,
		ProcessingTimeMs: result.ProcessingTimeMs,
		TotalHits:        result.TotalHits,
		Results:          result.Hits,
	})
}

// GetAirportByCode retrieves an airport by its 3-letter IATA code or 4-letter ICAO code.
func (h *Handler) GetAirportByCode(w http.ResponseWriter, r *http.Request) {
	lang := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("lang")))
	code := strings.ToUpper(strings.TrimSpace(chi.URLParam(r, "code")))
	if code == "" {
		writeError(w, http.StatusBadRequest, "Missing airport code")
		return
	}

	filter := fmt.Sprintf(`class = "Airport" AND (iata_code = "%s" OR icao_code = "%s")`, code, code)
	place, err := h.meili.FindOneByFilter(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Error retrieving airport: "+err.Error())
		return
	}

	if place == nil {
		writeError(w, http.StatusNotFound, fmt.Sprintf("Airport not found for code: %s", code))
		return
	}

	localizePlace(place, lang)
	*r = *appMiddleware.SetResultHits(r, 1)
	writeJSON(w, http.StatusOK, place)
}
