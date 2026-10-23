package schema

// Place is the unified autosuggest document for destination-like entities.
type Place struct {
	ID                string            `json:"id"`
	EntityID          string            `json:"entity_id"`
	Class             string            `json:"class"`
	Subclass          string            `json:"subclass,omitempty"`
	NameDisplay       string            `json:"name_display"`
	NamesByLanguage   map[string]string `json:"names_by_language,omitempty"`
	NameAltActive     []string          `json:"name_alt_active,omitempty"`
	NameAltHistorical []string          `json:"name_alt_historical,omitempty"`
	CountryCode       string            `json:"country_code,omitempty"`
	CountryName       string            `json:"country_name,omitempty"`
	AdminHierarchy    []string          `json:"admin_hierarchy,omitempty"`
	ParentEntityID    string            `json:"parent_entity_id,omitempty"`
	Geo               GeoPoint          `json:"_geo"`
	BBox              []float64         `json:"bbox,omitempty"`
	IATACode          string            `json:"iata_code,omitempty"`
	ICAOCode          string            `json:"icao_code,omitempty"`
	StarRating        int               `json:"star_rating,omitempty"`
	Cuisine           string            `json:"cuisine,omitempty"`
	AccommodationType string            `json:"accommodation_type,omitempty"`
	AdminLevel        int               `json:"admin_level,omitempty"`
	Population        int               `json:"population,omitempty"`
	WikidataID        string            `json:"wikidata_id,omitempty"`
	PopularityScore   float64           `json:"popularity_score"`
	Source            string            `json:"source"`
	UpdatedAt         string            `json:"updated_at"`
}

// GeoPoint is Meilisearch's expected geo object shape.
type GeoPoint struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}
