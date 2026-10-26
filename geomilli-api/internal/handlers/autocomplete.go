package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/geomilli/geomilli-api/internal/countries"
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
	countryParam := strings.TrimSpace(r.URL.Query().Get("country"))
	class := strings.TrimSpace(r.URL.Query().Get("class"))
	subclass := strings.TrimSpace(r.URL.Query().Get("subclass"))

	var countryCode string
	if countryParam != "" {
		if code, ok := countries.NormalizeCode(countryParam); ok {
			countryCode = code
		} else if len(countryParam) == 2 {
			countryCode = strings.ToUpper(countryParam)
		}
	}

	// Smart query extraction if country parameter was not explicitly provided
	if countryCode == "" && q != "" {
		if cleaned, code, found := countries.ExtractFromQuery(q); found && cleaned != "" {
			countryCode = code
			q = cleaned
		}
	}

	limit := parseLimit(r, DefaultAutocompleteLimit, MaxAutocompleteLimit)

	var filters []string
	if countryCode != "" {
		filters = append(filters, fmt.Sprintf(`country_code = "%s"`, countryCode))
	} else if countryParam != "" {
		filters = append(filters, fmt.Sprintf(`(country_code = "%s" OR country_name = "%s")`, countryParam, countryParam))
	}
	if class != "" {
		filters = append(filters, fmt.Sprintf(`class = "%s"`, class))
	}
	if subclass != "" {
		filters = append(filters, fmt.Sprintf(`subclass = "%s"`, subclass))
	}

	var sortRules []string
	userLat, userLng, hasGeo := parseOptionalLatLng(r)
	if hasGeo {
		sortRules = append(sortRules, fmt.Sprintf("_geoPoint(%f, %f):asc", userLat, userLng))
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
		writeError(w, http.StatusBadGateway, ErrUpstreamUnavailable, "Search error: "+err.Error())
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
