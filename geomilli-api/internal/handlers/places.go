package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/geomilli/geomilli-api/internal/meili"
	appMiddleware "github.com/geomilli/geomilli-api/internal/middleware"
	"github.com/geomilli/geomilli-api/internal/schema"
	"github.com/go-chi/chi/v5"
)

// PlaceDetailsResponse models the detailed place response.
type PlaceDetailsResponse struct {
	schema.Place
	NearbyPlaces []schema.Place `json:"nearby_places,omitempty"`
}

// GetPlaceByID returns details for a place by Meilisearch ID or OSM entity ID.
func (h *Handler) GetPlaceByID(w http.ResponseWriter, r *http.Request) {
	lang := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("lang")))
	id := chi.URLParam(r, "id")
	if id == "" {
		writeError(w, http.StatusBadRequest, ErrMissingParameter, "Missing place ID")
		return
	}

	// Try direct document fetch first
	place, err := h.meili.GetDocument(r.Context(), id)
	if err != nil || place == nil {
		// Fallback to searching by entity_id or id filter
		filter := fmt.Sprintf(`entity_id = "%s" OR id = "%s"`, id, id)
		place, err = h.meili.FindOneByFilter(r.Context(), filter)
		if err != nil {
			writeError(w, http.StatusBadGateway, ErrUpstreamUnavailable, "Error retrieving place: "+err.Error())
			return
		}
	}

	if place == nil {
		writeError(w, http.StatusNotFound, ErrNotFound, fmt.Sprintf("Place not found for ID: %s", id))
		return
	}

	localizePlace(place, lang)
	resp := PlaceDetailsResponse{Place: *place}

	if strings.ToLower(r.URL.Query().Get("include_nearby")) == "true" && place.Geo.Lat != 0 && place.Geo.Lng != 0 {
		resp.NearbyPlaces = h.fetchNearbyPOIs(r, place.Geo.Lat, place.Geo.Lng, place.ID)
		localizePlaces(resp.NearbyPlaces, lang)
	}

	*r = *appMiddleware.SetResultHits(r, 1)
	writeJSON(w, http.StatusOK, resp)
}

// GetPlaceByWikidataID returns details for a place by its Wikidata QID (e.g. Q31).
func (h *Handler) GetPlaceByWikidataID(w http.ResponseWriter, r *http.Request) {
	lang := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("lang")))
	qid := strings.ToUpper(strings.TrimSpace(chi.URLParam(r, "qid")))
	if qid == "" {
		writeError(w, http.StatusBadRequest, ErrMissingParameter, "Missing Wikidata QID")
		return
	}

	filter := fmt.Sprintf(`wikidata_id = "%s"`, qid)
	place, err := h.meili.FindOneByFilter(r.Context(), filter)
	if err != nil {
		writeError(w, http.StatusBadGateway, ErrUpstreamUnavailable, "Error retrieving place by Wikidata ID: "+err.Error())
		return
	}

	if place == nil {
		writeError(w, http.StatusNotFound, ErrNotFound, fmt.Sprintf("Place not found for Wikidata ID: %s", qid))
		return
	}

	localizePlace(place, lang)
	resp := PlaceDetailsResponse{Place: *place}

	if strings.ToLower(r.URL.Query().Get("include_nearby")) == "true" && place.Geo.Lat != 0 && place.Geo.Lng != 0 {
		resp.NearbyPlaces = h.fetchNearbyPOIs(r, place.Geo.Lat, place.Geo.Lng, place.ID)
		localizePlaces(resp.NearbyPlaces, lang)
	}

	*r = *appMiddleware.SetResultHits(r, 1)
	writeJSON(w, http.StatusOK, resp)
}

func (h *Handler) fetchNearbyPOIs(r *http.Request, lat, lng float64, excludeID string) []schema.Place {
	searchReq := meili.SearchRequest{
		Filter: fmt.Sprintf("_geoRadius(%f, %f, 2000)", lat, lng),
		Sort:   []string{fmt.Sprintf("_geoPoint(%f, %f):asc", lat, lng)},
		Limit:  15,
	}

	result, err := h.meili.SearchPlaces(r.Context(), searchReq)
	if err != nil {
		return nil
	}

	var pois []schema.Place
	for _, p := range result.Hits {
		if p.ID == excludeID {
			continue
		}
		dist := meili.HaversineDistance(lat, lng, p.Geo.Lat, p.Geo.Lng)
		p.DistanceMeters = &dist
		pois = append(pois, p)
		if len(pois) >= 10 {
			break
		}
	}
	return pois
}
