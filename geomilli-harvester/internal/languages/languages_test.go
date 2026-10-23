package languages

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadFromFile(t *testing.T) {
	tempDir := t.TempDir()
	yamlContent := `
version: 1
languages:
  - code: en
    name: English
    enabled: true
  - code: nl
    name: Dutch
    enabled: true
  - code: fr
    name: French
    enabled: false
  - code: ES
    name: Spanish
    enabled: true
`
	path := filepath.Join(tempDir, "languages.yaml")
	if err := os.WriteFile(path, []byte(yamlContent), 0644); err != nil {
		t.Fatal(err)
	}

	enabled, err := LoadFromFile(path)
	if err != nil {
		t.Fatalf("LoadFromFile failed: %v", err)
	}

	if len(enabled) != 3 {
		t.Errorf("expected 3 enabled languages, got %d: %v", len(enabled), enabled)
	}

	if !enabled["en"] {
		t.Error("expected 'en' to be enabled")
	}
	if !enabled["nl"] {
		t.Error("expected 'nl' to be enabled")
	}
	if !enabled["es"] {
		t.Error("expected 'es' (lowercased) to be enabled")
	}
	if enabled["fr"] {
		t.Error("expected 'fr' to be disabled")
	}
}

func TestLoadFromFile_NonExistent(t *testing.T) {
	_, err := LoadFromFile("non-existent-file.yaml")
	if err == nil {
		t.Error("expected error for non-existent file, got nil")
	}
}

func TestAddOrUpdateLanguage(t *testing.T) {
	tempDir := t.TempDir()
	path := filepath.Join(tempDir, "languages.yaml")
	if err := os.WriteFile(path, []byte("version: 1\nlanguages:\n  - code: en\n    name: English\n    enabled: true\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Add new language 'th'
	cfg, err := AddOrUpdateLanguage(path, "th", "Thai", true)
	if err != nil {
		t.Fatalf("AddOrUpdateLanguage failed: %v", err)
	}
	if len(cfg.Languages) != 2 {
		t.Errorf("expected 2 languages, got %d", len(cfg.Languages))
	}

	// Toggle 'en' to false
	cfg, err = AddOrUpdateLanguage(path, "en", "English", false)
	if err != nil {
		t.Fatalf("AddOrUpdateLanguage toggle failed: %v", err)
	}
	if cfg.Languages[0].Enabled {
		t.Errorf("expected en to be disabled")
	}
}
