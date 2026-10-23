package osm_test

import (
	"testing"

	"github.com/travayo/auto-complete/internal/geojsonseq"
	"github.com/travayo/auto-complete/internal/osm"
	"github.com/travayo/auto-complete/internal/taxonomy"
)

func TestNormalizeFeature_City(t *testing.T) {
	tax, err := taxonomy.LoadFromFile("../../config/taxonomy.yaml")
	if err != nil {
		t.Fatalf("failed to load taxonomy: %v", err)
	}

	// Whitelisted languages: en, fr, nl
	enabledLangs := map[string]bool{"en": true, "fr": true, "nl": true}

	feature := &geojsonseq.Feature{
		ID: "node/123456",
		Properties: map[string]interface{}{
			"name":        "Brussels",
			"name:fr":     "Bruxelles",
			"name:nl":     "Brussel",
			"name:ru":     "Брюссель", // unwhitelisted, should be filtered out
			"place":       "city",
			"wikidata":    "Q239",
			"population":  "1200000",
			"admin_level": "8",
		},
		Geometry: geojsonseq.Geometry{
			Type:        "Point",
			Coordinates: []interface{}{4.3517, 50.8503},
		},
	}

	place, ok := osm.NormalizeFeature("BE", "Belgium", feature, tax, enabledLangs)
	if !ok || place == nil {
		t.Fatalf("expected feature to be normalized successfully")
	}

	if place.NameDisplay != "Brussels" {
		t.Errorf("expected NameDisplay 'Brussels', got '%s'", place.NameDisplay)
	}
	if place.Class != "City" {
		t.Errorf("expected Class 'City', got '%s'", place.Class)
	}
	if place.WikidataID != "Q239" {
		t.Errorf("expected WikidataID 'Q239', got '%s'", place.WikidataID)
	}
	if place.Population != 1200000 {
		t.Errorf("expected population 1200000, got %d", place.Population)
	}

	// Verify whitelisted languages are kept
	if place.NamesByLanguage["fr"] != "Bruxelles" {
		t.Errorf("expected French name 'Bruxelles', got '%s'", place.NamesByLanguage["fr"])
	}
	if place.NamesByLanguage["nl"] != "Brussel" {
		t.Errorf("expected Dutch name 'Brussel', got '%s'", place.NamesByLanguage["nl"])
	}

	// Verify unwhitelisted language is filtered out (Rule 3)
	if _, hasRu := place.NamesByLanguage["ru"]; hasRu {
		t.Errorf("unwhitelisted language 'ru' should have been ignored to prevent index bloat")
	}
}

func TestNormalizeFeature_Airport(t *testing.T) {
	tax, err := taxonomy.LoadFromFile("../../config/taxonomy.yaml")
	if err != nil {
		t.Fatalf("failed to load taxonomy: %v", err)
	}

	enabledLangs := map[string]bool{"en": true, "fr": true, "nl": true}

	airportFeature := &geojsonseq.Feature{
		ID: "way/7890",
		Properties: map[string]interface{}{
			"name":     "Brussels Airport",
			"aeroway":  "aerodrome",
			"iata":     "bru",
			"icao":     "ebbr",
			"wikidata": "Q494593",
		},
		Geometry: geojsonseq.Geometry{
			Type:        "Point",
			Coordinates: []interface{}{4.4844, 50.9014},
		},
	}

	place, ok := osm.NormalizeFeature("BE", "Belgium", airportFeature, tax, enabledLangs)
	if !ok || place == nil {
		t.Fatalf("expected airport to normalize")
	}

	if place.Class != "Airport" {
		t.Errorf("expected Class 'Airport', got '%s'", place.Class)
	}
	if place.IATACode != "BRU" {
		t.Errorf("expected IATA 'BRU', got '%s'", place.IATACode)
	}
	if place.ICAOCode != "EBBR" {
		t.Errorf("expected ICAO 'EBBR', got '%s'", place.ICAOCode)
	}
}

func TestNormalizeFeature_Invalid(t *testing.T) {
	tax, err := taxonomy.LoadFromFile("../../config/taxonomy.yaml")
	if err != nil {
		t.Fatalf("failed to load taxonomy: %v", err)
	}
	enabledLangs := map[string]bool{"en": true}

	// Missing name returns false
	featureNoName := &geojsonseq.Feature{
		ID: "node/999",
		Properties: map[string]interface{}{
			"place": "city",
		},
		Geometry: geojsonseq.Geometry{
			Type:        "Point",
			Coordinates: []interface{}{4.0, 50.0},
		},
	}
	_, ok := osm.NormalizeFeature("BE", "Belgium", featureNoName, tax, enabledLangs)
	if ok {
		t.Errorf("feature without name should not normalize")
	}
}
