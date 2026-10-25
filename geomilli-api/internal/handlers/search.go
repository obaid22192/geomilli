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

// SearchResponse represents the response for /v1/search.
type SearchResponse struct {
	Query             string                    `json:"query"`
	ProcessingTimeMs  int64                     `json:"took_ms"`
	Limit             int                       `json:"limit"`
	Offset            int                       `json:"offset"`
	TotalHits         int64                     `json:"total_hits"`
	FacetDistribution map[string]map[string]int `json:"facets,omitempty"`
	Results           []schema.Place            `json:"results"`
}

// Search handles full-text and faceted search queries.
func (h *Handler) Search(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	lang := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("lang")))
	filter := strings.TrimSpace(r.URL.Query().Get("filter"))
	sortParam := strings.TrimSpace(r.URL.Query().Get("sort"))
	facetsParam := strings.TrimSpace(r.URL.Query().Get("facets"))

	limit := parseLimit(r, DefaultSearchLimit, MaxSearchLimit)
	offset := parseOffset(r)

	var sortRules []string
	if sortParam != "" {
		for _, s := range strings.Split(sortParam, ",") {
			s = strings.TrimSpace(s)
			if s != "" {
				sortRules = append(sortRules, s)
			}
		}
	} else {
		sortRules = []string{"popularity_score:desc"}
	}

	var facets []string
	if facetsParam != "" {
		if facetsParam == "*" || facetsParam == "all" {
			facets = []string{"class", "subclass", "country_code", "cuisine", "star_rating", "accommodation_type"}
		} else {
			for _, f := range strings.Split(facetsParam, ",") {
				f = strings.TrimSpace(f)
				if f != "" {
					facets = append(facets, f)
				}
			}
		}
	}

	var filters []string
	if filter != "" {
		filters = append(filters, filter)
	}
	if class := strings.TrimSpace(r.URL.Query().Get("class")); class != "" {
		filters = append(filters, fmt.Sprintf(`class = "%s"`, class))
	}
	if subclass := strings.TrimSpace(r.URL.Query().Get("subclass")); subclass != "" {
		filters = append(filters, fmt.Sprintf(`subclass = "%s"`, subclass))
	}
	if country := strings.ToUpper(strings.TrimSpace(r.URL.Query().Get("country"))); country != "" {
		filters = append(filters, fmt.Sprintf(`country_code = "%s"`, country))
	}
	if cuisine := strings.TrimSpace(r.URL.Query().Get("cuisine")); cuisine != "" {
		filters = append(filters, fmt.Sprintf(`cuisine = "%s"`, cuisine))
	}
	if starRatingStr := strings.TrimSpace(r.URL.Query().Get("star_rating")); starRatingStr != "" {
		if stars, err := strconv.Atoi(starRatingStr); err == nil && stars > 0 {
			filters = append(filters, fmt.Sprintf("star_rating >= %d", stars))
		}
	}

	filterExpr := ""
	if len(filters) > 0 {
		filterExpr = strings.Join(filters, " AND ")
	}

	searchReq := meili.SearchRequest{
		Q:                     q,
		Filter:                filterExpr,
		Sort:                  sortRules,
		Facets:                facets,
		Limit:                 limit,
		Offset:                offset,
		AttributesToHighlight: []string{"name_display"},
	}

	result, err := h.meili.SearchPlaces(r.Context(), searchReq)
	if err != nil {
		writeError(w, http.StatusBadGateway, ErrUpstreamUnavailable, "Search error: "+err.Error())
		return
	}

	*r = *appMiddleware.SetResultHits(r, len(result.Hits))
	localizePlaces(result.Hits, lang)

	writeJSON(w, http.StatusOK, SearchResponse{
		Query:             q,
		ProcessingTimeMs:  result.ProcessingTimeMs,
		Limit:             limit,
		Offset:            offset,
		TotalHits:         result.TotalHits,
		FacetDistribution: result.FacetDistribution,
		Results:           result.Hits,
	})
}
