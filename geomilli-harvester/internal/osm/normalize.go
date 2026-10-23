package osm

import (
	"fmt"
	"hash/fnv"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/travayo/auto-complete/internal/geojsonseq"
	"github.com/travayo/auto-complete/internal/schema"
	"github.com/travayo/auto-complete/internal/taxonomy"
)

func NormalizeFeature(countryCode string, countryName string, f *geojsonseq.Feature, tax *taxonomy.Taxonomy, enabledLangs map[string]bool) (*schema.Place, bool) {
	name := stringValue(f.Properties, "name")
	if name == "" {
		return nil, false
	}
	lat, lng := centroid(f.Geometry)
	if math.IsNaN(lat) || math.IsNaN(lng) {
		return nil, false
	}

	class, adminLevel := tax.Classify(f.Properties)
	if class == "" {
		return nil, false
	}

	entityID := strings.TrimSpace(fmt.Sprintf("%v", f.ID))
	if entityID == "" || entityID == "<nil>" || strings.EqualFold(entityID, "null") {
		entityID = fallbackEntityID(countryCode, class, name, lat, lng)
	}
	sourceType, sourceID := splitOSMID(entityID)
	if sourceType == "" || sourceID == "" {
		sourceType = "feature"
		sourceID = sanitize(entityID)
	}
	if sourceID == "" {
		sourceID = fallbackEntityID(countryCode, class, name, lat, lng)
	}

	namesByLang := collectLanguageNames(f.Properties, enabledLangs)
	altNames := collectAltNames(f.Properties)
	for _, val := range namesByLang {
		val = strings.TrimSpace(val)
		if val != "" {
			altNames = append(altNames, val)
		}
	}
	altNames = deduplicate(altNames)

	// Extract class-specific attributes using taxonomy definitions.
	var iataCode, icaoCode string
	var starRating int
	var cuisine string
	var accommodationType string
	var subclass string

	attrs := tax.AttributesFor(class)
	for _, attr := range attrs {
		raw := stringValue(f.Properties, attr.Tag)
		if raw == "" {
			continue
		}
		switch attr.Transform {
		case "upper":
			raw = strings.ToUpper(raw)
		case "int":
			// Handled below via intValue.
		}
		switch attr.Field {
		case "iata_code":
			iataCode = raw
		case "icao_code":
			icaoCode = raw
		case "star_rating":
			starRating = intValue(f.Properties, attr.Tag)
		case "accommodation_type":
			accommodationType = raw
		case "cuisine":
			cuisine = raw
		}
	}

	// Derive subclass from the matching tag value that classified this feature.
	// For example, a place_of_worship classified as TouristAttraction gets subclass="place_of_worship".
	subclass = deriveSubclass(f.Properties, class, tax)

	// Append cuisine terms to alt names so they remain searchable
	// via name_alt_active (e.g. "italian", "sushi", "thai").
	if cuisine != "" {
		for _, part := range strings.Split(cuisine, ";") {
			v := strings.TrimSpace(part)
			if v != "" {
				altNames = append(altNames, v)
			}
		}
		altNames = deduplicate(altNames)
	}

	place := &schema.Place{
		ID:                fmt.Sprintf("osm-%s-%s", sourceType, sourceID),
		EntityID:          sourceID,
		Class:             class,
		Subclass:          subclass,
		NameDisplay:       name,
		NamesByLanguage:   namesByLang,
		NameAltActive:     altNames,
		CountryCode:       strings.ToUpper(countryCode),
		CountryName:       countryName,
		Geo: schema.GeoPoint{
			Lat: lat,
			Lng: lng,
		},
		BBox:              f.BBox,
		IATACode:          iataCode,
		ICAOCode:          icaoCode,
		StarRating:        starRating,
		Cuisine:           cuisine,
		AccommodationType: accommodationType,
		AdminLevel:        adminLevel,
		Population:        intValue(f.Properties, "population"),
		WikidataID:        stringValue(f.Properties, "wikidata"),
		PopularityScore:   0.2,
		Source:            "osm",
		UpdatedAt:         time.Now().UTC().Format(time.RFC3339),
	}
	return place, true
}

// deriveSubclass extracts the specific tag value that matched for this class.
// This enables downstream filtering like "show only museums" within TouristAttraction.
func deriveSubclass(props map[string]interface{}, class string, tax *taxonomy.Taxonomy) string {
	for _, c := range tax.Classes {
		if c.Name != class {
			continue
		}
		for _, rule := range c.Match {
			if rule.Key == "admin_level" {
				continue
			}
			val := stringValue(props, rule.Key)
			if val == "" {
				continue
			}
			for _, v := range rule.Values {
				if val == v {
					return val
				}
			}
		}
		break
	}
	return ""
}

func collectLanguageNames(props map[string]interface{}, enabled map[string]bool) map[string]string {
	out := map[string]string{}
	for k, v := range props {
		if !strings.HasPrefix(k, "name:") {
			continue
		}
		lang := strings.TrimPrefix(k, "name:")
		if lang == "" {
			continue
		}
		lowerLang := strings.ToLower(lang)
		if len(enabled) > 0 && !enabled[lowerLang] {
			continue
		}
		s := strings.TrimSpace(fmt.Sprintf("%v", v))
		if s != "" {
			out[lang] = s
		}
	}
	return out
}

func collectAltNames(props map[string]interface{}) []string {
	keys := []string{"alt_name", "loc_name", "official_name", "short_name", "old_name", "int_name"}
	set := map[string]struct{}{}
	out := []string{}
	for _, k := range keys {
		raw := stringValue(props, k)
		for _, part := range strings.Split(raw, ";") {
			v := strings.TrimSpace(part)
			if v == "" {
				continue
			}
			if _, ok := set[v]; ok {
				continue
			}
			set[v] = struct{}{}
			out = append(out, v)
		}
	}
	return out
}

func buildSearchTerms(primary string, alts []string, cuisine string) string {
	parts := []string{primary}
	parts = append(parts, alts...)
	if cuisine != "" {
		for _, part := range strings.Split(cuisine, ";") {
			parts = append(parts, strings.TrimSpace(part))
		}
	}
	for i, p := range parts {
		parts[i] = strings.TrimSpace(strings.ToLower(p))
	}
	return strings.Join(deduplicate(parts), " ")
}

func stringValue(props map[string]interface{}, key string) string {
	v, ok := props[key]
	if !ok {
		return ""
	}
	return strings.TrimSpace(fmt.Sprintf("%v", v))
}

func intValue(props map[string]interface{}, key string) int {
	s := stringValue(props, key)
	if s == "" {
		return 0
	}
	i, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return i
}

func centroid(g geojsonseq.Geometry) (float64, float64) {
	switch g.Type {
	case "Point":
		if p, ok := g.Coordinates.([]interface{}); ok && len(p) >= 2 {
			lng, _ := asFloat(p[0])
			lat, _ := asFloat(p[1])
			return lat, lng
		}
	case "Polygon":
		if poly, ok := g.Coordinates.([]interface{}); ok && len(poly) > 0 {
			if ring, ok := poly[0].([]interface{}); ok {
				return centroidFromRing(ring)
			}
		}
	case "MultiPolygon":
		if mpoly, ok := g.Coordinates.([]interface{}); ok && len(mpoly) > 0 {
			if poly, ok := mpoly[0].([]interface{}); ok && len(poly) > 0 {
				if ring, ok := poly[0].([]interface{}); ok {
					return centroidFromRing(ring)
				}
			}
		}
	case "LineString":
		if line, ok := g.Coordinates.([]interface{}); ok && len(line) > 0 {
			return centroidFromRing(line)
		}
	}
	return math.NaN(), math.NaN()
}

func centroidFromRing(ring []interface{}) (float64, float64) {
	if len(ring) == 0 {
		return math.NaN(), math.NaN()
	}
	var sumLat, sumLng float64
	var n float64
	for _, p := range ring {
		pair, ok := p.([]interface{})
		if !ok || len(pair) < 2 {
			continue
		}
		lng, okLng := asFloat(pair[0])
		lat, okLat := asFloat(pair[1])
		if !okLng || !okLat {
			continue
		}
		sumLat += lat
		sumLng += lng
		n++
	}
	if n == 0 {
		return math.NaN(), math.NaN()
	}
	return sumLat / n, sumLng / n
}

func asFloat(v interface{}) (float64, bool) {
	s := fmt.Sprintf("%v", v)
	f, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0, false
	}
	return f, true
}

func splitOSMID(id string) (string, string) {
	parts := strings.Split(strings.TrimSpace(id), "/")
	if len(parts) != 2 {
		return "", ""
	}
	return sanitize(parts[0]), sanitize(parts[1])
}

func sanitize(in string) string {
	in = strings.TrimSpace(in)
	if in == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(in))
	lastDash := false
	for _, r := range in {
		isAlphaNum := (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9')
		if isAlphaNum || r == '_' || r == '-' {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return ""
	}
	return out
}

func fallbackEntityID(countryCode, class, name string, lat, lng float64) string {
	h := fnv.New64a()
	_, _ = h.Write([]byte(strings.ToUpper(countryCode)))
	_, _ = h.Write([]byte("|"))
	_, _ = h.Write([]byte(class))
	_, _ = h.Write([]byte("|"))
	_, _ = h.Write([]byte(strings.ToLower(strings.TrimSpace(name))))
	_, _ = h.Write([]byte("|"))
	_, _ = h.Write([]byte(fmt.Sprintf("%.6f|%.6f", lat, lng)))
	return fmt.Sprintf("fallback-%x", h.Sum64())
}

func deduplicate(in []string) []string {
	set := map[string]struct{}{}
	out := []string{}
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s == "" {
			continue
		}
		lower := strings.ToLower(s)
		if _, ok := set[lower]; !ok {
			set[lower] = struct{}{}
			out = append(out, s)
		}
	}
	return out
}
