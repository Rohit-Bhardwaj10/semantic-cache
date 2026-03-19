package api

import (
	"encoding/json"
	"fmt"
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
	// Extract tenant_id injected by AuthMiddleware
	tenantID := GetTenantID(r.Context())

	qReq := cache.QueryRequest{
		Query:    req.Query,
		TenantID: tenantID,
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
	tenantID := GetTenantID(r.Context())
	
	// Sprint 5/6 analytics: Real per-tenant data (mock for now)
	resp := map[string]interface{}{
		"tenant_id":     tenantID,
		"total_queries": 1000,
		"cache_hits":   780,
		"hit_rate":     0.78,
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}

// HandleAdminInvalidate processes a POST /admin/invalidate request.
func (h *Handler) HandleAdminInvalidate(w http.ResponseWriter, r *http.Request) {
	// In production, check for admin role from JWT claims
	// Handled by middleware soon
	
	var req struct {
		TenantID        string `json:"tenant_id"`
		NormalizedQuery string `json:"query_normalized"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid body", http.StatusBadRequest)
		return
	}

	// For now, we manually broadcast via a logic call
	// Sprint 4 implemented StartInvalidationListener on Coordinator
	// Here we'd typically publish to Redis
	// c.l2a.Client.Publish(ctx, InvalidationChannel, payload)
	
	w.WriteHeader(http.StatusAccepted)
	fmt.Fprintf(w, "Invalidation request accepted for tenant %s", req.TenantID)
}

// HandleAdminReload processes a POST /admin/reload-policies request.
func (h *Handler) HandleAdminReload(w http.ResponseWriter, r *http.Request) {
	if err := h.coord.ReloadPolicies(); err != nil {
		http.Error(w, "Failed to reload: "+err.Error(), http.StatusInternalServerError)
		return
	}
	fmt.Fprintln(w, "Policies reloaded successfully")
}
