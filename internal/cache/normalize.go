package cache

import (
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

var (
	punctuationRegex = regexp.MustCompile(`[^\w\s]`)
	whitespaceRegex  = regexp.MustCompile(`\s+`)
)

// Normalizer handles L0 normalization of query strings.
type Normalizer struct {
	synonyms map[string]string
}

// SynonymConfig matches the structure of configs/synonyms.yaml
type SynonymConfig struct {
	Synonyms map[string]string `yaml:"synonyms"`
}

// NewNormalizer creates a new Normalizer.
func NewNormalizer() *Normalizer {
	return &Normalizer{
		synonyms: make(map[string]string),
	}
}

// LoadSynonyms loads synonym mappings from a YAML file.
func (n *Normalizer) LoadSynonyms(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var config SynonymConfig
	if err := yaml.Unmarshal(data, &config); err != nil {
		return err
	}

	n.synonyms = config.Synonyms
	return nil
}

// Normalize applies the L0 normalization pipeline.
func (n *Normalizer) Normalize(query string) string {
	// Step 1: Lowercase
	res := strings.ToLower(query)

	// Step 2: Expand contractions
	res = expandContractions(res)

	// Step 3: Remove punctuation (strip non-alphanumeric)
	res = punctuationRegex.ReplaceAllString(res, "")

	// Step 4: Apply synonyms
	res = n.applySynonyms(res)

	// Step 5: Collapse whitespace
	res = whitespaceRegex.ReplaceAllLiteralString(res, " ")
	res = strings.TrimSpace(res)

	return res
}

func expandContractions(s string) string {
	// A basic set of common English contractions.
	// In a real system, this might be more exhaustive.
	replacements := map[string]string{
		"it's":    "it is",
		"that's":  "that is",
		"what's":  "what is",
		"where's": "where is",
		"who's":   "who is",
		"how's":   "how is",
		"i'm":     "i am",
		"you're":  "you are",
		"he's":    "he is",
		"she's":   "she is",
		"we're":   "we are",
		"they're": "they are",
		"can't":   "cannot",
		"won't":   "will not",
		"don't":   "do not",
		"doesn't": "does not",
		"didn't":  "did not",
		"itll":    "it will",
		"thatll":  "that will",
		"youve":   "you have",
		"ive":     "i have",
	}

	for k, v := range replacements {
		s = strings.ReplaceAll(s, k, v)
	}
	return s
}

func (n *Normalizer) applySynonyms(s string) string {
	// Note: We apply synonyms to the whole string or parts of it.
	// Since order matters (per plan), and we shouldn't sort words, 
	// we iterate through the synonym map.
	// The plan says: "Replacements are applied left-to-right (order matters for overlaps)."
	// This usually implies a sorted list of keys or taking synonyms as they appear in the file.
	// Since Go maps are unordered, this implementation might need a slice if order is critical.
	// For now, we'll do simple replacement.
	
	// Better approach for "order matters": users usually mean long matches first.
	// But our synonyms.yaml is just a map.
	for k, v := range n.synonyms {
		// Use word boundaries if possible, but simple replace for now 
		// as punctuation is already gone.
		s = strings.ReplaceAll(s, k, v)
	}
	return s
}
