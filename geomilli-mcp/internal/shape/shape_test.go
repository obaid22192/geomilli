package shape

import (
	"strings"
	"testing"

	"github.com/geomilli/geomilli-mcp/internal/apiclient"
)

func TestToCompact_NonVerbose(t *testing.T) {
	dist := 412.5
	p := &apiclient.Place{
		ID:          "osm_n123",
		EntityID:    "123",
		Class:       "TouristAttraction",
		Subclass:    "monument",
		NameDisplay: "Atomium",
		NamesByLanguage: map[string]string{
			"en": "Atomium",
			"nl": "Atomium",
		},
		NameAltActive:     []string{"The Atomium"},
		NameAltHistorical: []string{"Old Atomium"},
		CountryCode:       "BE",
		CountryName:       "Belgium",
		AdminHierarchy:    []string{"Laeken", "Brussels", "Brussels-Capital Region", "Belgium"},
		Geo: apiclient.GeoPoint{
			Lat: 50.8949,
			Lng: 4.3415,
		},
		WikidataID:      "Q207",
		PopularityScore: 0.91,
		DistanceMeters:  &dist,
		Source:          "osm",
		UpdatedAt:       "2026-10-24T00:00:00Z",
	}

	compact := ToCompact(p, false)

	if compact.ID != "osm_n123" {
		t.Errorf("expected ID=osm_n123, got %s", compact.ID)
	}
	if compact.Name != "Atomium" {
		t.Errorf("expected Name=Atomium, got %s", compact.Name)
	}
	if compact.Class != "TouristAttraction" {
		t.Errorf("expected Class=TouristAttraction, got %s", compact.Class)
	}
	if compact.Subclass != "monument" {
		t.Errorf("expected Subclass=monument, got %s", compact.Subclass)
	}
	if compact.Lat != 50.8949 || compact.Lng != 4.3415 {
		t.Errorf("expected coords 50.8949, 4.3415, got %f, %f", compact.Lat, compact.Lng)
	}
	if compact.DistanceMeters == nil || *compact.DistanceMeters != 412.5 {
		t.Errorf("expected distance 412.5, got %v", compact.DistanceMeters)
	}

	// Verify non-verbose stripped fields
	if len(compact.NamesByLanguage) != 0 {
		t.Errorf("expected NamesByLanguage stripped in non-verbose mode, got %v", compact.NamesByLanguage)
	}
	if len(compact.NameAltActive) != 0 {
		t.Errorf("expected NameAltActive stripped in non-verbose mode, got %v", compact.NameAltActive)
	}
	if len(compact.NameAltHistorical) != 0 {
		t.Errorf("expected NameAltHistorical stripped in non-verbose mode, got %v", compact.NameAltHistorical)
	}
	if compact.Source != "" {
		t.Errorf("expected Source stripped, got %q", compact.Source)
	}
	if compact.UpdatedAt != "" {
		t.Errorf("expected UpdatedAt stripped, got %q", compact.UpdatedAt)
	}
}

func TestToCompact_Verbose(t *testing.T) {
	p := &apiclient.Place{
		ID:          "osm_n123",
		NameDisplay: "Atomium",
		Class:       "TouristAttraction",
		NamesByLanguage: map[string]string{
			"en": "Atomium",
		},
		NameAltActive:     []string{"The Atomium"},
		NameAltHistorical: []string{"Old Atomium"},
		Source:            "osm",
		UpdatedAt:         "2026-10-24T00:00:00Z",
	}

	compact := ToCompact(p, true)

	if len(compact.NamesByLanguage) != 1 || compact.NamesByLanguage["en"] != "Atomium" {
		t.Errorf("expected NamesByLanguage preserved in verbose mode, got %v", compact.NamesByLanguage)
	}
	if len(compact.NameAltActive) != 1 || compact.NameAltActive[0] != "The Atomium" {
		t.Errorf("expected NameAltActive preserved in verbose mode, got %v", compact.NameAltActive)
	}
	if len(compact.NameAltHistorical) != 1 || compact.NameAltHistorical[0] != "Old Atomium" {
		t.Errorf("expected NameAltHistorical preserved in verbose mode, got %v", compact.NameAltHistorical)
	}
	if compact.Source != "osm" {
		t.Errorf("expected Source=osm, got %q", compact.Source)
	}
	if compact.UpdatedAt != "2026-10-24T00:00:00Z" {
		t.Errorf("expected UpdatedAt preserved, got %q", compact.UpdatedAt)
	}
}

func TestSummarizePlaceAndPlaces(t *testing.T) {
	dist := 1250.0
	p1 := CompactPlace{
		ID:             "p1",
		Name:           "Atomium",
		Class:          "TouristAttraction",
		Subclass:       "monument",
		Lat:            50.8949,
		Lng:            4.3415,
		AdminHierarchy: []string{"Brussels", "Belgium"},
		DistanceMeters: &dist,
	}

	summary := SummarizePlace(p1)
	if !strings.Contains(summary, "Atomium") || !strings.Contains(summary, "TouristAttraction, monument") {
		t.Errorf("unexpected place summary: %q", summary)
	}
	if !strings.Contains(summary, "1.2 km away") {
		t.Errorf("expected distance in summary, got %q", summary)
	}

	p2 := CompactPlace{
		ID:          "p2",
		Name:        "Manneken Pis",
		Class:       "TouristAttraction",
		Lat:         50.8450,
		Lng:         4.3500,
		CountryName: "Belgium",
	}

	placesSummary := SummarizePlaces([]CompactPlace{p1, p2}, 10)
	if !strings.Contains(placesSummary, "Found 2 place(s):") {
		t.Errorf("unexpected places summary header: %q", placesSummary)
	}
	if !strings.Contains(placesSummary, "1. Atomium") || !strings.Contains(placesSummary, "2. Manneken Pis") {
		t.Errorf("expected both places in summary: %q", placesSummary)
	}

	// Test truncation
	limitedSummary := SummarizePlaces([]CompactPlace{p1, p2}, 1)
	if !strings.Contains(limitedSummary, "... and 1 more.") {
		t.Errorf("expected truncation message, got: %q", limitedSummary)
	}
}
