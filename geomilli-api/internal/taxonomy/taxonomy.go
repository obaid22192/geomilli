package taxonomy

import (
	_ "embed"
	"fmt"
	"sync"

	"gopkg.in/yaml.v3"
)

//go:embed config/taxonomy.yaml
var rawTaxonomyYAML []byte

//go:embed config/languages.yaml
var rawLanguagesYAML []byte

// AttributeDefinition defines a mapped attribute on a taxonomy class.
type AttributeDefinition struct {
	Tag       string `yaml:"tag" json:"tag"`
	Field     string `yaml:"field" json:"field"`
	Transform string `yaml:"transform,omitempty" json:"transform,omitempty"`
}

// MatchRule defines an OSM tag matching rule.
type MatchRule struct {
	Key           string   `yaml:"key" json:"key"`
	Values        []string `yaml:"values" json:"values"`
	RequireAnyTag []string `yaml:"require_any_tag,omitempty" json:"require_any_tag,omitempty"`
	Geometry      []string `yaml:"geometry,omitempty" json:"geometry,omitempty"`
}

// ClassDefinition defines a single taxonomy class.
type ClassDefinition struct {
	Name       string                `yaml:"name" json:"name"`
	BaseScore  float64               `yaml:"base_score" json:"base_score"`
	Layer      string                `yaml:"layer" json:"layer"`
	Match      []MatchRule           `yaml:"match" json:"match"`
	Attributes []AttributeDefinition `yaml:"attributes,omitempty" json:"attributes,omitempty"`
}

// TaxonomyConfig holds the parsed taxonomy.yaml.
type TaxonomyConfig struct {
	Version int               `yaml:"version" json:"version"`
	Classes []ClassDefinition `yaml:"classes" json:"classes"`
}

// LanguageItem models a supported language entry.
type LanguageItem struct {
	Code    string `yaml:"code" json:"code"`
	Name    string `yaml:"name" json:"name"`
	Enabled bool   `yaml:"enabled" json:"enabled"`
}

// LanguagesConfig holds the parsed languages.yaml.
type LanguagesConfig struct {
	Version   int            `yaml:"version" json:"version"`
	Languages []LanguageItem `yaml:"languages" json:"languages"`
}

var (
	once sync.Once
	tax  *TaxonomyConfig
	lang *LanguagesConfig
	errInit error
)

// GetTaxonomy returns the loaded taxonomy configuration.
func GetTaxonomy() (*TaxonomyConfig, error) {
	initData()
	return tax, errInit
}

// GetLanguages returns the loaded languages configuration.
func GetLanguages() (*LanguagesConfig, error) {
	initData()
	return lang, errInit
}

func initData() {
	once.Do(func() {
		tax = &TaxonomyConfig{}
		if err := yaml.Unmarshal(rawTaxonomyYAML, tax); err != nil {
			errInit = fmt.Errorf("failed to parse embedded taxonomy.yaml: %w", err)
			return
		}

		lang = &LanguagesConfig{}
		if err := yaml.Unmarshal(rawLanguagesYAML, lang); err != nil {
			errInit = fmt.Errorf("failed to parse embedded languages.yaml: %w", err)
			return
		}
	})
}
