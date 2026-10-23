package countries_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/travayo/auto-complete/internal/countries"
)

func TestResolveFromConfig(t *testing.T) {
	// 1. Resolve from actual config/countries.yaml
	list, err := countries.Resolve("../../config/countries.yaml", "")
	if err != nil {
		t.Fatalf("failed to resolve countries from config: %v", err)
	}
	if len(list) == 0 {
		t.Fatalf("expected countries to be non-empty")
	}

	// Verify Belgium (BE) is in the list
	foundBE := false
	for _, c := range list {
		if c.Code == "BE" {
			foundBE = true
			if c.GeofabrikPath != "europe/belgium" {
				t.Errorf("expected GeofabrikPath 'europe/belgium', got '%s'", c.GeofabrikPath)
			}
			break
		}
	}
	if !foundBE {
		t.Errorf("expected Belgium (BE) in countries catalog")
	}

	// 2. Custom mock config
	tmpDir, err := os.MkdirTemp("", "countries_test_*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	mockYAML := `
countries:
  - code: lu
    geofabrik_path: europe/luxembourg
    name: Luxembourg
  - code: nl
    geofabrik_path: europe/netherlands
`
	cfgPath := filepath.Join(tmpDir, "countries.yaml")
	if err := os.WriteFile(cfgPath, []byte(mockYAML), 0o644); err != nil {
		t.Fatalf("failed to write mock yaml: %v", err)
	}

	customList, err := countries.Resolve(cfgPath, "")
	if err != nil {
		t.Fatalf("failed to resolve custom countries: %v", err)
	}
	if len(customList) != 2 {
		t.Fatalf("expected 2 countries, got %d", len(customList))
	}
	if customList[0].Code != "LU" || customList[1].Code != "NL" {
		t.Errorf("expected sorted [LU, NL], got [%s, %s]", customList[0].Code, customList[1].Code)
	}
}
