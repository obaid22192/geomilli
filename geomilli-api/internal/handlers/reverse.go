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
	Address        ReverseAddress `json:"address"`
	AdminHierarchy []string       `json:"admin_hierarchy"`
	ClosestPlace   *schema.Place  `json:"closest_place,omitempty"`
	NearestPlaces  []schema.Place `json:"nearest_places"`
}

// Reverse resolves latitude and longitude coordinates to address and admin hierarchy.
func (h *Handler) Reverse(w http.ResponseWriter, r *http.Request) {
	latStr := strings.TrimSpace(r.URL.Query().Get("lat"))
	lngStr := strings.TrimSpace(r.URL.Query().Get("lng"))

	if latStr == "" || lngStr == "" {
		writeError(w, http.StatusBadRequest, "Both 'lat' and 'lng' query parameters are required")
		return
	}

	lat, errLat := strconv.ParseFloat(latStr, 64)
	lng, errLng := strconv.ParseFloat(lngStr, 64)
	if errLat != nil || errLng != nil || lat < -90 || lat > 90 || lng < -180 || lng > 180 {
		writeError(w, http.StatusBadRequest, "Invalid lat or lng coordinate values")
		return
	}

	radius := 50000 // 50km
	if rStr := r.URL.Query().Get("radius_meters"); rStr != "" {
		if val, err := strconv.Atoi(rStr); err == nil && val > 0 {
			radius = val
		}
	}

	searchReq := meili.SearchRequest{
		Filter: fmt.Sprintf("_geoRadius(%f, %f, %d)", lat, lng, radius),
		Sort:   []string{fmt.Sprintf("_geoPoint(%f, %f):asc", lat, lng)},
		Limit:  30,
	}

	result, err := h.meili.SearchPlaces(r.Context(), searchReq)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "Reverse geocode error: "+err.Error())
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
		Address:        addr,
		AdminHierarchy: adminHierarchy,
		ClosestPlace:   closest,
		NearestPlaces:  nearestPlaces,
	})
}
