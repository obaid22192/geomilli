package handlers

import (
	"fmt"
	"net/http"

	"github.com/geomilli/geomilli-api/internal/meili"
	appMiddleware "github.com/geomilli/geomilli-api/internal/middleware"
	"github.com/geomilli/geomilli-api/internal/schema"
)

// ReverseAddress models formatted address components from reverse geocoding.
type ReverseAddress struct {
	Landmark     string `json:"landmark,omitempty"`
	Neighborhood string `json:"neighborhood,omitempty"`
	City         string `json:"city,omitempty"`
	Region       string `json:"region,omitempty"`
	Country      string `json:"country,omitempty"`
	CountryCode  string `json:"country_code,omitempty"`
}

// ReverseResponse represents the response for /v1/reverse.
type ReverseResponse struct {
	Lat            float64        `json:"lat"`
	Lng            float64        `json:"lng"`
	RadiusMeters   int            `json:"radius_meters"`
	Address        ReverseAddress `json:"address"`
	AdminHierarchy []string       `json:"admin_hierarchy"`
	ClosestPlace   *schema.Place  `json:"closest_place,omitempty"`
	NearestPlaces  []schema.Place `json:"nearest_places"`
}

// Reverse resolves latitude and longitude coordinates to address and admin hierarchy.
func (h *Handler) Reverse(w http.ResponseWriter, r *http.Request) {
	lat, lng, err := parseLatLng(r)
	if err != nil {
		if pErr, ok := err.(*ParamError); ok {
			writeError(w, http.StatusBadRequest, pErr.Code, pErr.Message)
			return
		}
		writeError(w, http.StatusBadRequest, ErrInvalidParameter, err.Error())
		return
	}

	radius := parseRadius(r, DefaultReverseRadius, MaxReverseRadius)

	searchReq := meili.SearchRequest{
		Filter: fmt.Sprintf("_geoRadius(%f, %f, %d)", lat, lng, radius),
		Sort:   []string{fmt.Sprintf("_geoPoint(%f, %f):asc", lat, lng)},
		Limit:  30,
	}

	result, err := h.meili.SearchPlaces(r.Context(), searchReq)
	if err != nil {
		writeError(w, http.StatusBadGateway, ErrUpstreamUnavailable, "Reverse geocode error: "+err.Error())
		return
	}

	var closest *schema.Place
	var nearestPlaces []schema.Place
	var addr ReverseAddress
	var adminHierarchy []string

	for i := range result.Hits {
		p := result.Hits[i]
		dist := meili.HaversineDistance(lat, lng, p.Geo.Lat, p.Geo.Lng)
		p.DistanceMeters = &dist

		if i == 0 {
			closest = &p
			if addr.CountryCode == "" && p.CountryCode != "" {
				addr.CountryCode = p.CountryCode
			}
			if addr.Country == "" && p.CountryName != "" {
				addr.Country = p.CountryName
			}
			if len(p.AdminHierarchy) > 0 && len(adminHierarchy) == 0 {
				adminHierarchy = p.AdminHierarchy
			}
		}

		// Identify admin components
		switch p.Class {
		case "Neighborhood":
			if addr.Neighborhood == "" {
				addr.Neighborhood = p.NameDisplay
			}
		case "City":
			if addr.City == "" {
				addr.City = p.NameDisplay
			}
		case "Region":
			if addr.Region == "" {
				addr.Region = p.NameDisplay
			}
		case "Country":
			if addr.Country == "" {
				addr.Country = p.NameDisplay
			}
			if addr.CountryCode == "" {
				addr.CountryCode = p.CountryCode
			}
		default:
			if addr.Landmark == "" && (p.Class == "TouristAttraction" || p.Class == "HistoricSite" || p.Class == "TrainStation" || p.Class == "Airport") {
				addr.Landmark = p.NameDisplay
			}
		}

		nearestPlaces = append(nearestPlaces, p)
	}

	// Fallback admin hierarchy resolution if empty
	if len(adminHierarchy) == 0 {
		if addr.Neighborhood != "" {
			adminHierarchy = append(adminHierarchy, addr.Neighborhood)
		}
		if addr.City != "" {
			adminHierarchy = append(adminHierarchy, addr.City)
		}
		if addr.Region != "" {
			adminHierarchy = append(adminHierarchy, addr.Region)
		}
		if addr.Country != "" {
			adminHierarchy = append(adminHierarchy, addr.Country)
		}
	}

	*r = *appMiddleware.SetResultHits(r, len(nearestPlaces))

	writeJSON(w, http.StatusOK, ReverseResponse{
		Lat:            lat,
		Lng:            lng,
		RadiusMeters:   radius,
		Address:        addr,
		AdminHierarchy: adminHierarchy,
		ClosestPlace:   closest,
		NearestPlaces:  nearestPlaces,
	})
}
