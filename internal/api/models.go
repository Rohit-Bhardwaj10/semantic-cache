package api

import "fmt"

const (
	MaxQueryLength = 2048 // 2KB max for a single query to prevent DoS
)

// QueryRequest represents an incoming user query via POST /cache/query.
type QueryRequest struct {
	Query  string `json:"query"`
	Domain string `json:"domain,omitempty"` // optional - auto-classified if omitted
}

func (r *QueryRequest) Validate() error {
	if r.Query == "" {
		return fmt.Errorf("query cannot be empty")
	}
	if len(r.Query) > MaxQueryLength {
		return fmt.Errorf("query too long (max %d chars)", MaxQueryLength)
	}
	return nil
}

// QueryResponse is the unified response for both cache hits and misses.
type QueryResponse struct {
	Answer     string   `json:"answer"`
	Source     string   `json:"source"` // "L1", "L2a", "L2b", or "backend"
	Hit        bool     `json:"hit"`
	Confidence float32  `json:"confidence,omitempty"`
	LatencyMS  int64    `json:"latency_ms"`
	
	// Cache hit metadata
	CachedQuery string  `json:"cached_query,omitempty"`
	Similarity  float32 `json:"similarity,omitempty"`
	AgeSeconds  int     `json:"age_seconds,omitempty"`
	
	// Cache miss metadata
	CostUSD     float64 `json:"cost_usd,omitempty"`
}

// HealthResponse represents the status of the service and its dependencies.
type HealthResponse struct {
	Status   string                  `json:"status"`
	Services map[string]ServiceStatus `json:"services,omitempty"`
}

type ServiceStatus struct {
	OK        bool  `json:"ok"`
	LatencyMS int64 `json:"latency_ms,omitempty"`
}
