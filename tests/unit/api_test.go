package unit

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Rohit-Bhardwaj10/semantic-cache/internal/api"
	"github.com/Rohit-Bhardwaj10/semantic-cache/internal/cache"
	"github.com/Rohit-Bhardwaj10/semantic-cache/internal/policy"
)

func TestAPI_Query(t *testing.T) {
	// Setup Mocks
	l1 := cache.NewL1Cache(1024)
	l1.Set("default", "hello", "world", 0)

	coord := cache.NewCoordinator(cache.Config{
		L1:         l1,
		Normalizer: cache.NewNormalizer(),
		Classifier: policy.NewDomainClassifier(),
	})

	h := api.NewHandler(coord)

	// 1. Valid Cache Hit
	reqBody, _ := json.Marshal(api.QueryRequest{Query: "hello"})
	r := httptest.NewRequest("POST", "/cache/query", bytes.NewBuffer(reqBody))
	w := httptest.NewRecorder()

	h.HandleQuery(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	var resp api.QueryResponse
	_ = json.NewDecoder(w.Body).Decode(&resp)

	if resp.Answer != "world" || resp.Source != "L1" {
		t.Errorf("Unexpected response: %+v", resp)
	}

	// 2. Invalid Method
	r = httptest.NewRequest("GET", "/cache/query", nil)
	w = httptest.NewRecorder()
	h.HandleQuery(w, r)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("Expected 405, got %d", w.Code)
	}
}

func TestAPI_Health(t *testing.T) {
	h := api.NewHandler(nil)
	w := httptest.NewRecorder()
	r := httptest.NewRequest("GET", "/health", nil)

	h.HandleHealth(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	var resp api.HealthResponse
	_ = json.NewDecoder(w.Body).Decode(&resp)
	if resp.Status != "ready" {
		t.Errorf("Expected ready, got %s", resp.Status)
	}
}
