package policy

// Policy defines the scoring and threshold parameters for a specific query domain.
type Policy struct {
	MinSimilarity       float32 `yaml:"min_similarity"`
	MaxStalenessSeconds int     `yaml:"max_staleness_seconds"`
	ConfidenceThreshold float32 `yaml:"confidence_threshold"`
	SimWeight           float32 `yaml:"sim_weight"`
	FreshWeight         float32 `yaml:"fresh_weight"`
}

// PolicyConfig maps domain names to their specific Policy.
type PolicyConfig map[string]Policy
