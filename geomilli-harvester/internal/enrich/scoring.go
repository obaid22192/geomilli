package enrich

import (
	"math"

	"github.com/travayo/auto-complete/internal/schema"
	"github.com/travayo/auto-complete/internal/taxonomy"
)

// DerivePopularity computes a [0.0, 1.0] popularity score for a place
// using the taxonomy-defined base score plus adjustments from population,
// Wikidata sitelink count, and Wikipedia pageview volume.
func DerivePopularity(p schema.Place, sitelinks int, pageviews int, tax *taxonomy.Taxonomy) float64 {
	base := tax.BaseScore(p.Class)
	popAdj := 0.0
	if p.Population > 0 {
		popAdj = math.Min(0.30, math.Log10(float64(p.Population)+1)/25.0)
	}
	sitelinkAdj := math.Min(0.15, math.Log10(float64(sitelinks)+1)/12.0)
	pageviewAdj := math.Min(0.15, math.Log10(float64(pageviews)+1)/20.0)
	score := base + popAdj + sitelinkAdj + pageviewAdj
	if score > 1.0 {
		return 1.0
	}
	if score < 0.0 {
		return 0.0
	}
	return score
}
