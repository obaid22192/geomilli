package handlers

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"

	"github.com/geomilli/geomilli-api/internal/meili"
	appMiddleware "github.com/geomilli/geomilli-api/internal/middleware"
	"github.com/geomilli/geomilli-api/internal/schema"
	"github.com/go-chi/chi/v5"
)

// CountrySummary models an indexed country with its place count.
type CountrySummary struct {
	CountryCode string `json:"country_code"`
	CountryName string `json:"country_name"`
	PlaceCount  int    `json:"place_count"`
}

// CountriesListResponse models /v1/countries response.
type CountriesListResponse struct {
	TotalCountries int              `json:"total_countries"`
	Countries      []CountrySummary `json:"countries"`
}

// RegionsListResponse models /v1/countries/{code}/regions response.
type RegionsListResponse struct {
	CountryCode string         `json:"country_code"`
	TotalHits   int64          `json:"total_hits"`
	Regions     []schema.Place `json:"regions"`
}

// CitiesListResponse models /v1/countries/{code}/cities response.
type CitiesListResponse struct {
	CountryCode string         `json:"country_code"`
	Limit       int            `json:"limit"`
	Offset      int            `json:"offset"`
	TotalHits   int64          `json:"total_hits"`
	Cities      []schema.Place `json:"cities"`
}

// ListCountries returns all indexed countries with entity counts.
func (h *Handler) ListCountries(w http.ResponseWriter, r *http.Request) {
	// First fetch facet distribution of country_code
	facetReq := meili.SearchRequest{
		Facets: []string{"country_code"},
		Limit:  0,
	}
	facetRes, err := h.meili.Search(r.Context(), facetReq)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to query countries: "+err.Error())
		return
	}

	codeCounts := make(map[string]int)
	if facetRes.FacetDistribution != nil {
		if counts, ok := facetRes.FacetDistribution["country_code"]; ok {
			for code, count := range counts {
				if code != "" {
					codeCounts[strings.ToUpper(code)] = count
				}
			}
		}
	}

	// Also query class = "Country" places to get official display names
	countryPlaces, _ := h.meili.SearchPlaces(r.Context(), meili.SearchRequest{
		Filter: `class = "Country"`,
		Limit:  100,
	})

	nameMap := make(map[string]string)
	if countryPlaces != nil {
		for _, cp := range countryPlaces.Hits {
			if cp.CountryCode != "" {
				nameMap[strings.ToUpper(cp.CountryCode)] = cp.NameDisplay
			}
		}
	}

	var summaries []CountrySummary
	for code, count := range codeCounts {
		name := nameMap[code]
		if name == "" {
			name = code
		}
		summaries = append(summaries, CountrySummary{
			CountryCode: code,
			CountryName: name,
			PlaceCount:  count,
		})
	}

	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].PlaceCount > summaries[j].PlaceCount
	})

	*r = *appMiddleware.SetResultHits(r, len(summaries))

	writeJSON(w, http.StatusOK, CountriesListResponse{
		TotalCountries: len(summaries),
		Countries:      summaries,
	})
}

// ListCountryRegions returns provinces/regions for a specific country.
func (h *Handler) ListCountryRegions(w http.ResponseWriter, r *http.Request) {
	code := strings.ToUpper(strings.TrimSpace(chi.URLParam(r, "code")))
	if code == "" {
		writeError(w, http.StatusBadRequest, "Missing country code")
		return
	}

	filter := fmt.Sprintf(`class = "Region" AND country_code = "%s"`, code)
	res, err := h.meili.SearchPlaces(r.Context(), meili.SearchRequest{
		Filter: filter,
		Sort:   []string{"popularity_score:desc"},
		Limit:  100,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to query regions: "+err.Error())
		return
	}

	*r = *appMiddleware.SetResultHits(r, len(res.Hits))

	writeJSON(w, http.StatusOK, RegionsListResponse{
		CountryCode: code,
		TotalHits:   res.TotalHits,
		Regions:     res.Hits,
	})
}

// ListCountryCities returns cities and towns for a specific country, sorted by population.
func (h *Handler) ListCountryCities(w http.ResponseWriter, r *http.Request) {
	code := strings.ToUpper(strings.TrimSpace(chi.URLParam(r, "code")))
	if code == "" {
		writeError(w, http.StatusBadRequest, "Missing country code")
		return
	}

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if val, err := strconv.Atoi(l); err == nil && val > 0 {
			if val > 200 {
				val = 200
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

	filter := fmt.Sprintf(`class = "City" AND country_code = "%s"`, code)
	res, err := h.meili.SearchPlaces(r.Context(), meili.SearchRequest{
		Filter: filter,
		Sort:   []string{"population:desc", "popularity_score:desc"},
		Limit:  limit,
		Offset: offset,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Failed to query cities: "+err.Error())
		return
	}

	*r = *appMiddleware.SetResultHits(r, len(res.Hits))

	writeJSON(w, http.StatusOK, CitiesListResponse{
		CountryCode: code,
		Limit:       limit,
		Offset:      offset,
		TotalHits:   res.TotalHits,
		Cities:      res.Hits,
	})
}
