package taxonomy

import "testing"

func TestTaxonomyAndLanguages(t *testing.T) {
	tax, err := GetTaxonomy()
	if err != nil {
		t.Fatalf("failed to get taxonomy: %v", err)
	}
	if len(tax.Classes) != 19 {
		t.Fatalf("expected 19 classes, got %d", len(tax.Classes))
	}

	lang, err := GetLanguages()
	if err != nil {
		t.Fatalf("failed to get languages: %v", err)
	}
	if len(lang.Languages) == 0 {
		t.Fatalf("expected languages, got 0")
	}
}
