package shape

import (
	"fmt"
	"strings"

	"github.com/geomilli/geomilli-mcp/internal/apiclient"
)

// CompactPlace is a token-efficient representation of a Place for LLMs per RFC §5.6.
type CompactPlace struct {
	ID                string    `json:"id"`
	Name              string    `json:"name"`
	Class             string    `json:"class"`
	Subclass          string    `json:"subclass,omitempty"`
	Lat               float64   `json:"lat"`
	Lng               float64   `json:"lng"`
	CountryCode       string    `json:"country_code,omitempty"`
	CountryName       string    `json:"country_name,omitempty"`
	AdminHierarchy    []string  `json:"admin_hierarchy,omitempty"`
	WikidataID        string    `json:"wikidata_id,omitempty"`
	PopularityScore   float64   `json:"popularity_score,omitempty"`
	DistanceMeters    *float64  `json:"distance_meters,omitempty"`
	IATACode          string    `json:"iata_code,omitempty"`
	ICAOCode          string    `json:"icao_code,omitempty"`
	StarRating        int       `json:"star_rating,omitempty"`
	Cuisine           string    `json:"cuisine,omitempty"`
	AccommodationType string    `json:"accommodation_type,omitempty"`
	Population        int       `json:"population,omitempty"`
	AdminLevel        int       `json:"admin_level,omitempty"`
	BBox              []float64 `json:"bbox,omitempty"`

	// Kept only in verbose mode
	NamesByLanguage   map[string]string `json:"names_by_language,omitempty"`
	NameAltActive     []string          `json:"name_alt_active,omitempty"`
	NameAltHistorical []string          `json:"name_alt_historical,omitempty"`
	Source            string            `json:"source,omitempty"`
	UpdatedAt         string            `json:"updated_at,omitempty"`
}

// ToCompact converts an apiclient.Place to a CompactPlace.
// In non-verbose mode, bulky alias lists, updated_at, and source are omitted.
func ToCompact(p *apiclient.Place, verbose bool) CompactPlace {
	if p == nil {
		return CompactPlace{}
	}

	cp := CompactPlace{
		ID:                p.ID,
		Name:              p.NameDisplay,
		Class:             p.Class,
		Subclass:          p.Subclass,
		Lat:               p.Geo.Lat,
		Lng:               p.Geo.Lng,
		CountryCode:       p.CountryCode,
		CountryName:       p.CountryName,
		AdminHierarchy:    p.AdminHierarchy,
		WikidataID:        p.WikidataID,
		PopularityScore:   p.PopularityScore,
		DistanceMeters:    p.DistanceMeters,
		IATACode:          p.IATACode,
		ICAOCode:          p.ICAOCode,
		StarRating:        p.StarRating,
		Cuisine:           p.Cuisine,
		AccommodationType: p.AccommodationType,
		Population:        p.Population,
		AdminLevel:        p.AdminLevel,
		BBox:              p.BBox,
	}

	if verbose {
		cp.NamesByLanguage = p.NamesByLanguage
		cp.NameAltActive = p.NameAltActive
		cp.NameAltHistorical = p.NameAltHistorical
		cp.Source = p.Source
		cp.UpdatedAt = p.UpdatedAt
	}

	return cp
}

// ToCompactSlice converts a slice of apiclient.Place to []CompactPlace.
func ToCompactSlice(places []apiclient.Place, verbose bool) []CompactPlace {
	if len(places) == 0 {
		return []CompactPlace{}
	}
	res := make([]CompactPlace, len(places))
	for i := range places {
		res[i] = ToCompact(&places[i], verbose)
	}
	return res
}

// SummarizePlace formats a single place into a concise human/LLM-readable summary.
func SummarizePlace(p CompactPlace) string {
	parts := []string{p.Name}

	typeDesc := p.Class
	if p.Subclass != "" {
		typeDesc = fmt.Sprintf("%s, %s", p.Class, p.Subclass)
	}
	parts = append(parts, fmt.Sprintf("(%s)", typeDesc))

	locParts := []string{}
	if len(p.AdminHierarchy) > 0 {
		locParts = append(locParts, strings.Join(p.AdminHierarchy, ", "))
	} else if p.CountryName != "" {
		locParts = append(locParts, p.CountryName)
	} else if p.CountryCode != "" {
		locParts = append(locParts, p.CountryCode)
	}

	if len(locParts) > 0 {
		parts = append(parts, "in", strings.Join(locParts, ", "))
	}

	parts = append(parts, fmt.Sprintf("[lat: %.4f, lng: %.4f]", p.Lat, p.Lng))

	if p.DistanceMeters != nil {
		if *p.DistanceMeters >= 1000 {
			parts = append(parts, fmt.Sprintf("(%.1f km away)", *p.DistanceMeters/1000.0))
		} else {
			parts = append(parts, fmt.Sprintf("(%.0f m away)", *p.DistanceMeters))
		}
	}

	return strings.Join(parts, " ")
}

// SummarizePlaces formats a list of compact places into a bulleted summary.
func SummarizePlaces(places []CompactPlace, maxItems int) string {
	if len(places) == 0 {
		return "No places found."
	}

	if maxItems <= 0 || maxItems > len(places) {
		maxItems = len(places)
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Found %d place(s):\n", len(places)))
	for i := 0; i < maxItems; i++ {
		sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, SummarizePlace(places[i])))
	}

	if len(places) > maxItems {
		sb.WriteString(fmt.Sprintf("... and %d more.", len(places)-maxItems))
	}

	return strings.TrimRight(sb.String(), "\n")
}
