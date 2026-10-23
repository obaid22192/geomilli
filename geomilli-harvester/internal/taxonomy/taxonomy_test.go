package taxonomy

import (
	"os"
	"path/filepath"
	"sort"
	"testing"

	"gopkg.in/yaml.v3"
)

// loadTestTaxonomy returns the full production taxonomy parsed from an
// embedded YAML string. Tests use this directly instead of reading a file.
func loadTestTaxonomy(t *testing.T) *Taxonomy {
	t.Helper()
	var tax Taxonomy
	if err := yaml.Unmarshal([]byte(fullTaxonomyYAML), &tax); err != nil {
		t.Fatalf("unmarshal test taxonomy: %v", err)
	}
	return &tax
}

// fullTaxonomyYAML is the production taxonomy embedded for testing.
const fullTaxonomyYAML = `
version: 1
classes:
  - name: TrainStation
    base_score: 0.60
    layer: poi
    match:
      - key: railway
        values: [station, halt]

  - name: Stadium
    base_score: 0.56
    layer: poi
    match:
      - key: leisure
        values: [stadium]

  - name: Hotel
    base_score: 0.50
    layer: poi
    match:
      - key: tourism
        values: [hotel, hostel, motel, guest_house, apartment, camp_site, chalet]
    attributes:
      - tag: stars
        field: star_rating
        transform: int
      - tag: tourism
        field: accommodation_type

  - name: TouristAttraction
    base_score: 0.58
    layer: poi
    match:
      - key: tourism
        values: [attraction, museum, viewpoint, zoo, theme_park, gallery, aquarium]
      - key: amenity
        values: [place_of_worship]
        require_any_tag: [wikidata, wikipedia]

  - name: HistoricSite
    base_score: 0.50
    layer: poi
    match:
      - key: historic
        values: [castle, monument, memorial, archaeological_site, ruins, fort, battlefield]

  - name: Park
    base_score: 0.48
    layer: poi
    match:
      - key: leisure
        values: [park, nature_reserve, marina]
      - key: boundary
        values: [national_park]

  - name: BusStation
    base_score: 0.48
    layer: poi
    match:
      - key: amenity
        values: [bus_station]

  - name: FerryTerminal
    base_score: 0.48
    layer: poi
    match:
      - key: amenity
        values: [ferry_terminal]

  - name: Restaurant
    base_score: 0.40
    layer: poi
    match:
      - key: amenity
        values: [restaurant, cafe, bar, pub]
    attributes:
      - tag: cuisine
        field: cuisine

  - name: Entertainment
    base_score: 0.40
    layer: poi
    match:
      - key: amenity
        values: [theatre, cinema, nightclub]
      - key: leisure
        values: [water_park]

  - name: Beach
    base_score: 0.45
    layer: poi
    match:
      - key: natural
        values: [beach]
      - key: leisure
        values: [beach_resort]

  - name: NaturalFeature
    base_score: 0.38
    layer: poi
    match:
      - key: natural
        values: [peak, volcano, cave_entrance, glacier]
        geometry: [node]

  - name: Shopping
    base_score: 0.38
    layer: poi
    match:
      - key: shop
        values: [mall, department_store]

  - name: Airport
    base_score: 0.80
    layer: poi
    match:
      - key: aeroway
        values: [aerodrome, airport]
    attributes:
      - tag: iata
        field: iata_code
        transform: upper
      - tag: icao
        field: icao_code
        transform: upper

  - name: Country
    base_score: 0.75
    layer: admin
    match:
      - key: place
        values: [country]

  - name: Region
    base_score: 0.55
    layer: admin
    match:
      - key: admin_level
        values: ["4"]

  - name: City
    base_score: 0.65
    layer: admin
    match:
      - key: admin_level
        values: ["6", "7", "8"]
      - key: place
        values: [city, town, village]

  - name: Neighborhood
    base_score: 0.45
    layer: admin
    match:
      - key: admin_level
        values: ["9", "10"]
      - key: place
        values: [suburb, neighbourhood, neighborhood, quarter]

  - name: Island
    base_score: 0.50
    layer: admin
    match:
      - key: place
        values: [island, islet]
`

// ---------------------------------------------------------------------------
// TestLoadFromFile
// ---------------------------------------------------------------------------

func TestLoadFromFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "taxonomy.yaml")
	if err := os.WriteFile(path, []byte(fullTaxonomyYAML), 0644); err != nil {
		t.Fatal(err)
	}

	tax, err := LoadFromFile(path)
	if err != nil {
		t.Fatalf("LoadFromFile: %v", err)
	}
	if len(tax.Classes) != 19 {
		t.Errorf("expected 19 classes, got %d", len(tax.Classes))
	}

	want := []string{
		"TrainStation", "Stadium", "Hotel", "TouristAttraction", "HistoricSite",
		"Park", "BusStation", "FerryTerminal", "Restaurant", "Entertainment",
		"Beach", "NaturalFeature", "Shopping", "Airport",
		"Country", "Region", "City", "Neighborhood", "Island",
	}
	got := tax.ClassNames()
	if len(got) != len(want) {
		t.Fatalf("ClassNames length: got %d, want %d", len(got), len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("ClassNames[%d]: got %q, want %q", i, got[i], want[i])
		}
	}
}

func TestLoadFromFile_NotFound(t *testing.T) {
	_, err := LoadFromFile("/nonexistent/path/taxonomy.yaml")
	if err == nil {
		t.Fatal("expected error for missing file")
	}
}

// ---------------------------------------------------------------------------
// Classification Tests
// ---------------------------------------------------------------------------

func TestClassify_TrainStation(t *testing.T) {
	tax := loadTestTaxonomy(t)
	cls, _ := tax.Classify(props("railway", "station"))
	assertEqual(t, "TrainStation", cls)
}

func TestClassify_Airport(t *testing.T) {
	tax := loadTestTaxonomy(t)
	cls, _ := tax.Classify(props("aeroway", "aerodrome"))
	assertEqual(t, "Airport", cls)
}

func TestClassify_City_AdminLevel(t *testing.T) {
	tax := loadTestTaxonomy(t)
	cls, al := tax.Classify(props("admin_level", "6"))
	assertEqual(t, "City", cls)
	if al != 6 {
		t.Errorf("admin_level: got %d, want 6", al)
	}
}

func TestClassify_City_PlaceTag(t *testing.T) {
	tax := loadTestTaxonomy(t)
	cls, _ := tax.Classify(props("place", "city"))
	assertEqual(t, "City", cls)
}

func TestClassify_Country(t *testing.T) {
	tax := loadTestTaxonomy(t)
	cls, _ := tax.Classify(props("place", "country"))
	assertEqual(t, "Country", cls)
}

func TestClassify_PlaceOfWorship_WithWikidata(t *testing.T) {
	tax := loadTestTaxonomy(t)
	cls, _ := tax.Classify(props("amenity", "place_of_worship", "wikidata", "Q123"))
	assertEqual(t, "TouristAttraction", cls)
}

func TestClassify_PlaceOfWorship_WithWikipedia(t *testing.T) {
	tax := loadTestTaxonomy(t)
	cls, _ := tax.Classify(props("amenity", "place_of_worship", "wikipedia", "en:Notre-Dame"))
	assertEqual(t, "TouristAttraction", cls)
}

func TestClassify_PlaceOfWorship_WithoutWikidata(t *testing.T) {
	tax := loadTestTaxonomy(t)
	cls, _ := tax.Classify(props("amenity", "place_of_worship"))
	assertEqual(t, "", cls)
}

func TestClassify_Nightclub(t *testing.T) {
	tax := loadTestTaxonomy(t)
	cls, _ := tax.Classify(props("amenity", "nightclub"))
	assertEqual(t, "Entertainment", cls)
}

func TestClassify_WaterPark(t *testing.T) {
	tax := loadTestTaxonomy(t)
	cls, _ := tax.Classify(props("leisure", "water_park"))
	assertEqual(t, "Entertainment", cls)
}

func TestClassify_BeachResort(t *testing.T) {
	tax := loadTestTaxonomy(t)
	cls, _ := tax.Classify(props("leisure", "beach_resort"))
	assertEqual(t, "Beach", cls)
}

func TestClassify_Fort(t *testing.T) {
	tax := loadTestTaxonomy(t)
	cls, _ := tax.Classify(props("historic", "fort"))
	assertEqual(t, "HistoricSite", cls)
}

func TestClassify_Glacier(t *testing.T) {
	tax := loadTestTaxonomy(t)
	cls, _ := tax.Classify(props("natural", "glacier"))
	assertEqual(t, "NaturalFeature", cls)
}

func TestClassify_Islet(t *testing.T) {
	tax := loadTestTaxonomy(t)
	cls, _ := tax.Classify(props("place", "islet"))
	assertEqual(t, "Island", cls)
}

func TestClassify_NationalPark(t *testing.T) {
	tax := loadTestTaxonomy(t)
	cls, _ := tax.Classify(props("boundary", "national_park"))
	assertEqual(t, "Park", cls)
}

func TestClassify_NoMatch(t *testing.T) {
	tax := loadTestTaxonomy(t)
	cls, _ := tax.Classify(props("building", "yes", "highway", "residential"))
	assertEqual(t, "", cls)
}

func TestClassify_PriorityOrder(t *testing.T) {
	tax := loadTestTaxonomy(t)
	// A feature with both railway=station AND admin_level=8 must be TrainStation, not City.
	cls, al := tax.Classify(props("railway", "station", "admin_level", "8"))
	assertEqual(t, "TrainStation", cls)
	if al != 8 {
		t.Errorf("admin_level: got %d, want 8", al)
	}
}

// ---------------------------------------------------------------------------
// BaseScore tests
// ---------------------------------------------------------------------------

func TestBaseScore(t *testing.T) {
	tax := loadTestTaxonomy(t)
	cases := []struct {
		class string
		want  float64
	}{
		{"Airport", 0.80},
		{"Country", 0.75},
		{"City", 0.65},
		{"TrainStation", 0.60},
		{"TouristAttraction", 0.58},
		{"Stadium", 0.56},
		{"Region", 0.55},
		{"Hotel", 0.50},
		{"HistoricSite", 0.50},
		{"Island", 0.50},
		{"Park", 0.48},
		{"BusStation", 0.48},
		{"FerryTerminal", 0.48},
		{"Beach", 0.45},
		{"Neighborhood", 0.45},
		{"Restaurant", 0.40},
		{"Entertainment", 0.40},
		{"NaturalFeature", 0.38},
		{"Shopping", 0.38},
		{"UnknownClass", 0.40}, // default
	}
	for _, tc := range cases {
		got := tax.BaseScore(tc.class)
		if got != tc.want {
			t.Errorf("BaseScore(%q): got %.2f, want %.2f", tc.class, got, tc.want)
		}
	}
}

// ---------------------------------------------------------------------------
// AttributesFor tests
// ---------------------------------------------------------------------------

func TestAttributesFor(t *testing.T) {
	tax := loadTestTaxonomy(t)

	// Hotel has 2 attributes.
	hotelAttrs := tax.AttributesFor("Hotel")
	if len(hotelAttrs) != 2 {
		t.Fatalf("Hotel attributes: got %d, want 2", len(hotelAttrs))
	}
	if hotelAttrs[0].Tag != "stars" || hotelAttrs[0].Field != "star_rating" || hotelAttrs[0].Transform != "int" {
		t.Errorf("Hotel attr 0: got %+v", hotelAttrs[0])
	}

	// Airport has 2 attributes.
	airportAttrs := tax.AttributesFor("Airport")
	if len(airportAttrs) != 2 {
		t.Fatalf("Airport attributes: got %d, want 2", len(airportAttrs))
	}
	if airportAttrs[0].Tag != "iata" || airportAttrs[0].Transform != "upper" {
		t.Errorf("Airport attr 0: got %+v", airportAttrs[0])
	}

	// Restaurant has 1 attribute.
	restAttrs := tax.AttributesFor("Restaurant")
	if len(restAttrs) != 1 || restAttrs[0].Tag != "cuisine" {
		t.Errorf("Restaurant attrs: got %+v", restAttrs)
	}

	// Park has no attributes.
	parkAttrs := tax.AttributesFor("Park")
	if parkAttrs != nil {
		t.Errorf("Park should have no attributes, got %+v", parkAttrs)
	}
}

// ---------------------------------------------------------------------------
// AdminOsmiumArgs tests
// ---------------------------------------------------------------------------

func TestAdminOsmiumArgs(t *testing.T) {
	tax := loadTestTaxonomy(t)
	args := tax.AdminOsmiumArgs()

	mustContain := []string{
		"r/boundary=administrative",
		"n/place=country",
		"n/place=city",
		"n/place=town",
		"n/place=village",
		"n/place=suburb",
		"n/place=neighbourhood",
		"n/place=neighborhood",
		"n/place=quarter",
		"n/place=island",
		"n/place=islet",
	}
	for _, want := range mustContain {
		if !containsStr(args, want) {
			t.Errorf("AdminOsmiumArgs missing %q, got %v", want, args)
		}
	}

	// Must not contain any POI tags.
	for _, a := range args {
		if containsAny(a, []string{"railway=", "tourism=", "amenity=", "aeroway="}) {
			t.Errorf("AdminOsmiumArgs should not contain POI tag: %q", a)
		}
	}
}

// ---------------------------------------------------------------------------
// POIOsmiumArgs tests
// ---------------------------------------------------------------------------

func TestPOIOsmiumArgs(t *testing.T) {
	tax := loadTestTaxonomy(t)
	args := tax.POIOsmiumArgs()

	mustContain := []string{
		"n/railway=station",
		"w/railway=station",
		"n/leisure=stadium",
		"w/leisure=stadium",
		"n/tourism=hotel",
		"n/amenity=place_of_worship",
		"n/aeroway=aerodrome",
		"w/aeroway=aerodrome",
		"n/natural=peak",
		"n/shop=mall",
		"w/shop=mall",
		"n/boundary=national_park",
		"w/boundary=national_park",
	}
	for _, want := range mustContain {
		if !containsStr(args, want) {
			t.Errorf("POIOsmiumArgs missing %q", want)
		}
	}

	// NaturalFeature geometry is node-only: w/natural=peak must NOT appear.
	for _, a := range args {
		if a == "w/natural=peak" || a == "w/natural=volcano" || a == "w/natural=cave_entrance" || a == "w/natural=glacier" {
			t.Errorf("POIOsmiumArgs should not contain way geometry for NaturalFeature: %q", a)
		}
	}

	// Must not contain admin-layer tags.
	for _, a := range args {
		if containsAny(a, []string{"admin_level=", "place=country", "place=city", "place=town"}) {
			t.Errorf("POIOsmiumArgs should not contain admin tag: %q", a)
		}
	}

	// No duplicates.
	sorted := make([]string, len(args))
	copy(sorted, args)
	sort.Strings(sorted)
	for i := 1; i < len(sorted); i++ {
		if sorted[i] == sorted[i-1] {
			t.Errorf("POIOsmiumArgs has duplicate: %q", sorted[i])
		}
	}
}

// ---------------------------------------------------------------------------
// Validation tests
// ---------------------------------------------------------------------------

func TestValidation_EmptyName(t *testing.T) {
	yml := `
version: 1
classes:
  - name: ""
    base_score: 0.5
    layer: poi
    match:
      - key: railway
        values: [station]
`
	_, err := LoadFromBytes([]byte(yml))
	if err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestValidation_InvalidLayer(t *testing.T) {
	yml := `
version: 1
classes:
  - name: Test
    base_score: 0.5
    layer: invalid
    match:
      - key: railway
        values: [station]
`
	_, err := LoadFromBytes([]byte(yml))
	if err == nil {
		t.Fatal("expected error for invalid layer")
	}
}

func TestValidation_NoMatchRules(t *testing.T) {
	yml := `
version: 1
classes:
  - name: Test
    base_score: 0.5
    layer: poi
    match: []
`
	_, err := LoadFromBytes([]byte(yml))
	if err == nil {
		t.Fatal("expected error for empty match rules")
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// props builds a properties map from alternating key/value string pairs.
func props(kvs ...string) map[string]interface{} {
	m := make(map[string]interface{}, len(kvs)/2)
	for i := 0; i+1 < len(kvs); i += 2 {
		m[kvs[i]] = kvs[i+1]
	}
	return m
}

func assertEqual(t *testing.T, want, got string) {
	t.Helper()
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func containsStr(slice []string, val string) bool {
	for _, s := range slice {
		if s == val {
			return true
		}
	}
	return false
}

func containsAny(s string, subs []string) bool {
	for _, sub := range subs {
		if len(s) >= len(sub) {
			for i := 0; i <= len(s)-len(sub); i++ {
				if s[i:i+len(sub)] == sub {
					return true
				}
			}
		}
	}
	return false
}
