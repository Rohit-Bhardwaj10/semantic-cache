package embeddings

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/redis/go-redis/v9"
)

var (
	ErrEmbedServer  = errors.New("ollama server error")
	ErrEmbedDecode  = errors.New("failed to decode embedding result")
	ErrEmbedMarshal = errors.New("failed to marshal embedding request")
)

// Breaker defined the interface needed for circuit breaking.
type Breaker interface {
	Execute(func() error) error
}

type Client struct {
	BaseURL    string
	Model      string
	HTTPClient *http.Client
	Redis      *redis.Client
	Breaker    Breaker
}

type EmbeddingRequest struct {
	Model  string `json:"model"`
	Prompt string `json:"prompt"`
}

type EmbeddingResponse struct {
	Embedding []float32 `json:"embedding"`
}

func NewOllamaClient(baseURL, model string, rdb *redis.Client, breaker Breaker) *Client {
	return &Client{
		BaseURL: baseURL,
		Model:   model,
		Redis:   rdb,
		Breaker: breaker,
		HTTPClient: &http.Client{
			Timeout: 15 * time.Second,
		},
	}
}

// Embed generates a vector embedding for the given text.
// It first checks Redis for a cached result. If not found, it calls Ollama
// within a circuit breaker and caches the result on success.
func (c *Client) Embed(ctx context.Context, text string) ([]float32, error) {
	// 1. Check Redis cache first if available
	cacheKey := fmt.Sprintf("emb:%s:%s", c.Model, text)
	if c.Redis != nil {
		val, err := c.Redis.Get(ctx, cacheKey).Bytes()
		if err == nil {
			var emb []float32
			if err := json.Unmarshal(val, &emb); err == nil {
				return emb, nil
			}
		}
	}

	// 2. Call Ollama within Circuit Breaker
	var result []float32
	embedFunc := func() error {
		reqBody := EmbeddingRequest{
			Model:  c.Model,
			Prompt: text,
		}

		data, err := json.Marshal(reqBody)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrEmbedMarshal, err)
		}

		url := fmt.Sprintf("%s/api/embeddings", c.BaseURL)
		resp, err := c.HTTPClient.Post(url, "application/json", bytes.NewBuffer(data))
		if err != nil {
			return fmt.Errorf("ollama request failed: %w", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("%w: status %d", ErrEmbedServer, resp.StatusCode)
		}

		var embedResp EmbeddingResponse
		if err := json.NewDecoder(resp.Body).Decode(&embedResp); err != nil {
			return fmt.Errorf("%w: %v", ErrEmbedDecode, err)
		}

		result = embedResp.Embedding
		return nil
	}

	var err error
	if c.Breaker != nil {
		err = c.Breaker.Execute(embedFunc)
	} else {
		err = embedFunc()
	}

	if err != nil {
		return nil, err
	}

	// 3. Cache the result in Redis if available
	if c.Redis != nil {
		data, _ := json.Marshal(result)
		// Cache for a long time (e.g., 7 days) as embeddings for the same model/text are deterministic
		c.Redis.Set(ctx, cacheKey, data, 7*24*time.Hour)
	}

	return result, nil
}
