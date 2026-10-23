// Package taxonomy provides a data-driven classification system for OSM features.
// It loads class definitions from a YAML config and classifies features by iterating
// match rules in priority order (first match wins).
package taxonomy

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Taxonomy holds the ordered list of class definitions loaded from config.
type Taxonomy struct {
	Version int        `yaml:"version"`
	Classes []ClassDef `yaml:"classes"`
}

// ClassDef describes a single feature class with its match rules and metadata.
type ClassDef struct {
	Name       string      `yaml:"name"`
	BaseScore  float64     `yaml:"base_score"`
	Layer      string      `yaml:"layer"` // "admin" or "poi"
	Match      []MatchRule `yaml:"match"`
	Attributes []AttrDef   `yaml:"attributes,omitempty"`
}

// MatchRule defines a single matching condition: key must have one of the listed values.
type MatchRule struct {
	Key           string   `yaml:"key"`
	Values        []string `yaml:"values"`
	Geometry      []string `yaml:"geometry,omitempty"`        // "node", "way", "relation"
	RequireAnyTag []string `yaml:"require_any_tag,omitempty"` // at least one must exist
}

// AttrDef describes how to extract an attribute from OSM tags into a schema field.
type AttrDef struct {
	Tag       string `yaml:"tag"`
	Field     string `yaml:"field"`
	Transform string `yaml:"transform,omitempty"` // "upper", "int", or "" for string
}

// LoadFromFile reads a taxonomy YAML file and returns a validated Taxonomy.
func LoadFromFile(path string) (*Taxonomy, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("taxonomy: read %s: %w", path, err)
	}
	var t Taxonomy
	if err := yaml.Unmarshal(data, &t); err != nil {
		return nil, fmt.Errorf("taxonomy: parse %s: %w", path, err)
	}
	if err := t.validate(); err != nil {
		return nil, fmt.Errorf("taxonomy: validate %s: %w", path, err)
	}
	return &t, nil
}

// LoadFromBytes parses taxonomy YAML from raw bytes and validates.
func LoadFromBytes(data []byte) (*Taxonomy, error) {
	var t Taxonomy
	if err := yaml.Unmarshal(data, &t); err != nil {
		return nil, fmt.Errorf("taxonomy: parse: %w", err)
	}
	if err := t.validate(); err != nil {
		return nil, fmt.Errorf("taxonomy: validate: %w", err)
	}
	return &t, nil
}

func (t *Taxonomy) validate() error {
	if len(t.Classes) == 0 {
		return fmt.Errorf("no classes defined")
	}
	for i, c := range t.Classes {
		if c.Name == "" {
			return fmt.Errorf("class %d has empty name", i)
		}
		if c.Layer != "admin" && c.Layer != "poi" {
			return fmt.Errorf("class %q has invalid layer %q (must be admin or poi)", c.Name, c.Layer)
		}
		if len(c.Match) == 0 {
			return fmt.Errorf("class %q has no match rules", c.Name)
		}
		for j, m := range c.Match {
			if m.Key == "" {
				return fmt.Errorf("class %q match rule %d has empty key", c.Name, j)
			}
			if len(m.Values) == 0 {
				return fmt.Errorf("class %q match rule %d has no values", c.Name, j)
			}
		}
	}
	return nil
}

// Classify iterates classes in order and returns the first matching class name.
// It always parses and returns the admin_level from props regardless of match.
func (t *Taxonomy) Classify(props map[string]interface{}) (string, int) {
	adminLevel := intValue(props, "admin_level")

	for _, c := range t.Classes {
		for _, rule := range c.Match {
			if !matchRule(props, rule, adminLevel) {
				continue
			}
			// Check require_any_tag: at least one listed tag must exist.
			if len(rule.RequireAnyTag) > 0 {
				if !hasAnyTag(props, rule.RequireAnyTag) {
					continue
				}
			}
			return c.Name, adminLevel
		}
	}
	return "", adminLevel
}

// matchRule checks whether the props satisfy a single match rule.
func matchRule(props map[string]interface{}, rule MatchRule, adminLevel int) bool {
	// Special handling: admin_level is compared as int.
	if rule.Key == "admin_level" {
		if adminLevel == 0 {
			return false
		}
		alStr := strconv.Itoa(adminLevel)
		return contains(rule.Values, alStr)
	}

	val := stringValue(props, rule.Key)
	if val == "" {
		return false
	}
	return contains(rule.Values, val)
}

// hasAnyTag returns true if at least one of the tags exists with a non-empty value.
func hasAnyTag(props map[string]interface{}, tags []string) bool {
	for _, tag := range tags {
		if stringValue(props, tag) != "" {
			return true
		}
	}
	return false
}

// BaseScore returns the base popularity score for a class, or 0.40 as default.
func (t *Taxonomy) BaseScore(className string) float64 {
	for _, c := range t.Classes {
		if c.Name == className {
			return c.BaseScore
		}
	}
	return 0.40
}

// AttributesFor returns the attribute extraction definitions for a class.
func (t *Taxonomy) AttributesFor(className string) []AttrDef {
	for _, c := range t.Classes {
		if c.Name == className {
			return c.Attributes
		}
	}
	return nil
}

// ClassNames returns all class names in definition order.
func (t *Taxonomy) ClassNames() []string {
	names := make([]string, len(t.Classes))
	for i, c := range t.Classes {
		names[i] = c.Name
	}
	return names
}

// geometryPrefixes converts geometry names to osmium tag-filter prefixes.
// Default (if geometry is empty): node and way → ["n/", "w/"].
func geometryPrefixes(geo []string) []string {
	if len(geo) == 0 {
		return []string{"n/", "w/"}
	}
	prefixes := make([]string, 0, len(geo))
	for _, g := range geo {
		switch g {
		case "node":
			prefixes = append(prefixes, "n/")
		case "way":
			prefixes = append(prefixes, "w/")
		case "relation":
			prefixes = append(prefixes, "r/")
		}
	}
	return prefixes
}

// AdminOsmiumArgs generates osmium tags-filter arguments for admin layer classes.
func (t *Taxonomy) AdminOsmiumArgs() []string {
	var args []string
	// All admin boundaries come through relations with boundary=administrative.
	args = append(args, "r/boundary=administrative")

	seen := map[string]bool{}
	for _, c := range t.Classes {
		if c.Layer != "admin" {
			continue
		}
		for _, rule := range c.Match {
			if rule.Key != "place" {
				continue
			}
			for _, v := range rule.Values {
				tag := "n/place=" + v
				if !seen[tag] {
					seen[tag] = true
					args = append(args, tag)
				}
			}
		}
	}
	return args
}

// POIOsmiumArgs generates osmium tags-filter arguments for poi layer classes.
func (t *Taxonomy) POIOsmiumArgs() []string {
	var args []string
	seen := map[string]bool{}
	for _, c := range t.Classes {
		if c.Layer != "poi" {
			continue
		}
		for _, rule := range c.Match {
			prefixes := geometryPrefixes(rule.Geometry)
			for _, v := range rule.Values {
				for _, pfx := range prefixes {
					tag := pfx + rule.Key + "=" + v
					if !seen[tag] {
						seen[tag] = true
						args = append(args, tag)
					}
				}
			}
		}
	}
	return args
}

// stringValue extracts a string from a properties map.
func stringValue(props map[string]interface{}, key string) string {
	v, ok := props[key]
	if !ok {
		return ""
	}
	return strings.TrimSpace(fmt.Sprintf("%v", v))
}

// intValue extracts an int from a properties map.
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

// contains checks if a string is in a slice.
func contains(slice []string, val string) bool {
	for _, s := range slice {
		if s == val {
			return true
		}
	}
	return false
}
