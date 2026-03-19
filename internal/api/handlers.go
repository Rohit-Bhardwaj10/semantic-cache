package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/Rohit-Bhardwaj10/semantic-cache/internal/cache"
)

// Handler handles HTTP requests for the semantic cache API.
type Handler struct {
	coord *cache.Coordinator
}

func NewHandler(coord *cache.Coordinator) *Handler {
	return &Handler{coord: coord}
}

// HandleQuery processes a POST /cache/query request.
func (h *Handler) HandleQuery(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req QueryRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Delegate to coordinator
	// In Sprint 6 we'll thread through the tenant_id from context/JWT.
	// For now (Sprint 5), we'll use a placeholder "default".
	qReq := cache.QueryRequest{
		Query:    req.Query,
		TenantID: "default",
		Domain:   req.Domain,
	}

	start := time.Now()
	res, err := h.coord.Query(r.Context(), qReq)
	if err != nil {
		http.Error(w, "Internal server error: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Prepare API response
	resp := QueryResponse{
		Answer:     res.Answer,
		Source:     res.Source,
		Hit:        res.Hit,
		Confidence: res.Confidence,
		LatencyMS:  time.Since(start).Milliseconds(),
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// HandleHealth process a GET /health or /readyz (shallow vs deep) request.
func (h *Handler) HandleHealth(w http.ResponseWriter, r *http.Request) {
	resp := HealthResponse{
		Status: "ready",
	}

	// In Sprint 5, we'll implement deep health checks later
	// For now, return OK.
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// HandleAnalytics processes a GET /analytics/cost-savings request.
func (h *Handler) HandleAnalytics(w http.ResponseWriter, r *http.Request) {
	// Sprint 5 analytics: Mock data
	resp := map[string]interface{}{
		"total_queries": 1000,
		"cache_hits":   780,
		"hit_rate":     0.78,
		"gross_savings_usd": 2.34,
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}
