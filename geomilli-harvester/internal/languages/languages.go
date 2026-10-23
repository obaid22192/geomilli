// Package languages provides a configuration-driven whitelist of languages
// to filter multilingual translations extracted during OSM normalization.
package languages

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type LanguageDef struct {
	Code    string `yaml:"code"`
	Name    string `yaml:"name"`
	Enabled bool   `yaml:"enabled"`
}

type Config struct {
	Version   int           `yaml:"version"`
	Languages []LanguageDef `yaml:"languages"`
}

// LoadFromFile reads a languages YAML file and returns a map of enabled language codes (lowercased).
func LoadFromFile(path string) (map[string]bool, error) {
	c, err := LoadFullConfig(path)
	if err != nil {
		return nil, err
	}

	enabled := make(map[string]bool)
	for _, l := range c.Languages {
		if l.Enabled {
			code := strings.ToLower(strings.TrimSpace(l.Code))
			if code != "" {
				enabled[code] = true
			}
		}
	}
	return enabled, nil
}

// LoadFullConfig reads and parses the full languages configuration.
func LoadFullConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("languages: read %s: %w", path, err)
	}
	var c Config
	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("languages: parse %s: %w", path, err)
	}
	return &c, nil
}

// SaveToFile serializes the language configuration back to YAML.
func SaveToFile(path string, c *Config) error {
	data, err := yaml.Marshal(c)
	if err != nil {
		return fmt.Errorf("languages: marshal: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("languages: write %s: %w", path, err)
	}
	return nil
}

// AddOrUpdateLanguage enables, disables, or adds a new language to the YAML config file.
func AddOrUpdateLanguage(path string, code string, name string, enabled bool) (*Config, error) {
	c, err := LoadFullConfig(path)
	if err != nil {
		return nil, err
	}

	code = strings.ToLower(strings.TrimSpace(code))
	if code == "" {
		return nil, fmt.Errorf("languages: code cannot be empty")
	}

	found := false
	for i := range c.Languages {
		if strings.EqualFold(c.Languages[i].Code, code) {
			c.Languages[i].Enabled = enabled
			if name != "" {
				c.Languages[i].Name = name
			}
			found = true
			break
		}
	}

	if !found {
		if name == "" {
			name = strings.ToUpper(code)
		}
		c.Languages = append(c.Languages, LanguageDef{
			Code:    code,
			Name:    name,
			Enabled: enabled,
		})
	}

	if err := SaveToFile(path, c); err != nil {
		return nil, err
	}
	return c, nil
}
