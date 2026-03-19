package backend

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

// Response represents the standard response object from an LLM.
type Response struct {
	Answer  string `json:"answer"`
	Model   string `json:"model"`
	Latency int64  `json:"latency_ms"`
}

// Backend defines the interface for calling upstream LLM services.
type Backend interface {
	Query(ctx context.Context, query string) (*Response, error)
}

// HTTPClient is a concrete implementation of the Backend interface.
type HTTPClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

func NewHTTPClient(baseURL string) *HTTPClient {
	return &HTTPClient{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 60 * time.Second,
		},
	}
}

func (c *HTTPClient) Query(ctx context.Context, query string) (*Response, error) {
	// Example payload matching common LLM API patterns
	payload := map[string]string{"prompt": query}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, "POST", c.BaseURL, bytes.NewBuffer(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("backend request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("backend returned error status: %d", resp.StatusCode)
	}

	var result Response
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}
	result.Latency = time.Since(start).Milliseconds()

	return &result, nil
}
