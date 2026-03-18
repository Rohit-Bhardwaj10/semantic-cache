package policy

import "strings"

const (
	TemporalPresent = "present"
	TemporalFuture  = "future"
	TemporalPast    = "past"
	TemporalNone    = ""
)

// TemporalClass extracts the temporal context from a normalized query string.
func TemporalClass(query string) string {
	q := strings.ToLower(query)

	// Future checks
	if containsAny(q, "tomorrow", "next week", "next month", "upcoming") {
		return TemporalFuture
	}

	// Past checks
	if containsAny(q, "yesterday", "last week", "last month", "previously") {
		return TemporalPast
	}

	// Present checks
	if containsAny(q, "today", "tonight", "now", "currently", "right now") {
		return TemporalPresent
	}

	return TemporalNone
}

func containsAny(s string, keywords ...string) bool {
	for _, kw := range keywords {
		// Use simple contains for now; normalization should have handled spacing.
		if strings.Contains(s, kw) {
			return true
		}
	}
	return false
}
