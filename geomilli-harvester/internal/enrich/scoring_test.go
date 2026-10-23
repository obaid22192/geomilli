package enrich_test

import (
	"testing"

	"github.com/travayo/auto-complete/internal/enrich"
	"github.com/travayo/auto-complete/internal/schema"
	"github.com/travayo/auto-complete/internal/taxonomy"
)

func TestDerivePopularity(t *testing.T) {
	tax, err := taxonomy.LoadFromFile("../../config/taxonomy.yaml")
	if err != nil {
		t.Fatalf("failed to load taxonomy: %v", err)
	}

	// 1. High-popularity city (e.g. Brussels with 1M population, sitelinks, pageviews)
	cityPlace := schema.Place{
		Class:      "City",
		Population: 1200000,
	}
	scoreCity := enrich.DerivePopularity(cityPlace, 150, 500000, tax)
	if scoreCity <= 0.5 || scoreCity > 1.0 {
		t.Errorf("expected high score between 0.5 and 1.0 for major city, got %f", scoreCity)
	}

	// 2. Small local shop / cafe without Wikidata sitelinks
	smallPlace := schema.Place{
		Class: "Cafe",
	}
	scoreSmall := enrich.DerivePopularity(smallPlace, 0, 0, tax)
	if scoreSmall > scoreCity {
		t.Errorf("small cafe score (%f) should be less than major city (%f)", scoreSmall, scoreCity)
	}

	// 3. Score bounded in [0.0, 1.0]
	hugePlace := schema.Place{
		Class:      "City",
		Population: 100000000,
	}
	scoreMax := enrich.DerivePopularity(hugePlace, 10000, 100000000, tax)
	if scoreMax > 1.0 {
		t.Errorf("score should be capped at 1.0, got %f", scoreMax)
	}
}
