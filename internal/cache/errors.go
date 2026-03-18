package cache

import "errors"

// Sentinel errors returned by the cache layer.
// Callers should use errors.Is() for comparison.
var (
	// ErrMiss is returned when no cache entry satisfies the query
	// (no match found, or all candidates rejected by policy).
	ErrMiss = errors.New("cache: miss")

	// ErrEmbeddingUnavailable is returned when the embedding service
	// (Ollama) is unreachable or responds with an error.
	ErrEmbeddingUnavailable = errors.New("cache: embedding service unavailable")

	// ErrExpired is returned when a candidate entry is found but its
	// TTL has elapsed. Distinct from ErrMiss so callers can log the
	// expiry reason rather than treating it as a cold miss.
	ErrExpired = errors.New("cache: entry expired")

	// ErrPolicyRejected is returned when candidates are found but all
	// are rejected by the policy engine (low confidence or temporal
	// mismatch). The coordinator will then proceed to the backend.
	ErrPolicyRejected = errors.New("cache: candidates rejected by policy")

	// ErrBackendUnavailable is returned when the upstream LLM backend
	// is unreachable (circuit breaker open or timeout).
	ErrBackendUnavailable = errors.New("cache: backend unavailable")

	// ErrInvalidQuery is returned when the incoming query fails
	// validation (empty, oversized, etc.).
	ErrInvalidQuery = errors.New("cache: invalid query")
)
