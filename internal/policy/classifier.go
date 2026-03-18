package policy

import (
	"strings"
)

// DomainClassifier categorizes queries into predefined domains for policy application.
type DomainClassifier struct {
	// keywords maps a domain name to a list of trigger words.
	keywords map[string][]string
}

// NewDomainClassifier initializes the classifier with core domain keywords.
func NewDomainClassifier() *DomainClassifier {
	return &DomainClassifier{
		keywords: map[string][]string{
			"weather": {"weather", "temperature", "forecast", "rain", "snow", "sunny", "humidity", "wind"},
			"finance": {"price", "stock", "market", "finance", "bitcoin", "crypto", "trading", "shares", "nasdaq", "dow jones"},
			"news":    {"news", "today", "headlines", "politics", "events", "happened", "current"},
			"coding":  {"code", "function", "golang", "javascript", "error", "how to", "debug", "library", "implementation"},
			"medical": {"symptom", "pain", "doctor", "health", "medicine", "disease", "treatment", "clinic"},
			"legal":   {"law", "legal", "statute", "court", "lawyer", "attorney", "regulation", "compliance"},
		},
	}
}

// Classify determines the domain of a query.
// It uses a keyword fast-path and will eventually support vector centroid fallback.
func (c *DomainClassifier) Classify(query string) string {
	q := strings.ToLower(query)

	// Step 1: Keyword fast-path
	for domain, kws := range c.keywords {
		for _, kw := range kws {
			if strings.Contains(q, kw) {
				return domain
			}
		}
	}

	// Step 2: Centroid-based vector fallback (Phase 2 feature)
	// For now, return "general" if no keywords match.
	return "general"
}
