package policy

import (
	"math"
)

// CalculateConfidence determines if a cached entry is reliable enough to serve.
// It uses an additive formula combining semantic similarity and freshness.
func CalculateConfidence(sim float32, ageSeconds int, p Policy) float32 {
	// 1. Hard gate: Similarity must be above the absolute minimum threshold
	if sim < p.MinSimilarity {
		return 0
	}

	// 2. Exponential freshness decay
	// Formula: freshness = exp(-age / max_staleness)
	// At age = 0, freshness = 1.0
	// At age = max_staleness, freshness = 1/e (~0.36)
	var freshness float64 
	if p.MaxStalenessSeconds > 0 {
		freshness = math.Exp(-float64(ageSeconds) / float64(p.MaxStalenessSeconds))
	} else {
		// If max staleness is 0 or negative, we treat it as "forever fresh" 
		// (though usually 0 TTL would be handled earlier).
		freshness = 1.0
	}

	// 3. Additive formula: SimWeight*sim + FreshWeight*freshness
	// This ensures that even highly similar results can be rejected if too old,
	// and vice-versa, but they don't zero each other out unless below hard gate.
	confidence := (p.SimWeight * sim) + (p.FreshWeight * float32(freshness))

	return confidence
}
