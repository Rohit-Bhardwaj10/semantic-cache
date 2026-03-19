package cache

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"golang.org/x/sync/singleflight"

	"github.com/Rohit-Bhardwaj10/semantic-cache/internal/backend"
	"github.com/Rohit-Bhardwaj10/semantic-cache/internal/policy"
	"github.com/Rohit-Bhardwaj10/semantic-cache/internal/resilience"
	"github.com/Rohit-Bhardwaj10/semantic-cache/pkg/embeddings"
)

// Coordinator orchestrates the L0 -> L1 -> L2a -> L2b -> Backend flow.
type Coordinator struct {
	normalizer *Normalizer
	l1         *L1Cache
	l2a        *L2aCache
	l2b        *L2bCache
	embeddings *embeddings.Client
	policy     *policy.Engine
	classifier *policy.DomainClassifier
	backend    backend.Backend
	breaker    *resilience.CircuitBreaker

	// Group for deduplicating concurrent backend calls
	sfGroup singleflight.Group
}

// Config holds the dependencies for the Coordinator.
type Config struct {
	Normalizer *Normalizer
	L1         *L1Cache
	L2a        *L2aCache
	L2b        *L2bCache
	Embeddings *embeddings.Client
	Policy     *policy.Engine
	Classifier *policy.DomainClassifier
	Backend    backend.Backend
	Breaker    *resilience.CircuitBreaker
}

func NewCoordinator(cfg Config) *Coordinator {
	return &Coordinator{
		normalizer: cfg.Normalizer,
		l1:         cfg.L1,
		l2a:        cfg.L2a,
		l2b:        cfg.L2b,
		embeddings: cfg.Embeddings,
		policy:     cfg.Policy,
		classifier: cfg.Classifier,
		backend:    cfg.Backend,
		breaker:    cfg.Breaker,
	}
}

// QueryRequest represents an incoming user query.
type QueryRequest struct {
	Query    string
	TenantID string
	Domain   string // optional
}

// QueryResponse represents the final result returned to the client.
type QueryResponse struct {
	Answer     string
	Source     string
	Hit        bool
	Confidence float32
	LatencyMS  int64
}

// Query handles the full multi-tier caching logic.
func (c *Coordinator) Query(ctx context.Context, req QueryRequest) (*QueryResponse, error) {
	start := time.Now()

	// Step 0: Normalize (L0)
	normalized := c.normalizer.Normalize(req.Query)
	domain := req.Domain
	if domain == "" {
		domain = c.classifier.Classify(normalized)
	}

	// Step 1: L1 Check (Exact Match)
	if ans, ok := c.l1.Get(req.TenantID, normalized); ok {
		return &QueryResponse{
			Answer:    ans,
			Source:    "L1",
			Hit:       true,
			LatencyMS: time.Since(start).Milliseconds(),
		}, nil
	}

	// Step 2: L2a Check (Redis Exact Match)
	if c.l2a != nil {
		ans, err := c.l2a.Get(ctx, req.TenantID, normalized)
		if err == nil && ans != "" {
			// Backfill L1
			c.l1.Set(req.TenantID, normalized, ans, 1*time.Hour)
			return &QueryResponse{
				Answer:    ans,
				Source:    "L2a",
				Hit:       true,
				LatencyMS: time.Since(start).Milliseconds(),
			}, nil
		}
	}

	// Step 3: L2b Check (Semantic Match)
	// First, we need an embedding
	emb, err := c.embeddings.Embed(ctx, normalized)
	if err == nil {
		// Search in Postgres
		candidates, err := c.l2b.Search(ctx, req.TenantID, emb, 5)
		if err == nil && len(candidates) > 0 {
			p := c.policy.GetPolicy(domain)

			// Check each candidate against policy
			for _, candle := range candidates {
				// Temporal check (Step 5.3 in Build Plan)
				if policy.TemporalClass(normalized) != policy.TemporalClass(candle.QueryNormalized) {
					continue
				}

				// Scoring (Step 5.2)
				confidence := policy.CalculateConfidence(candle.Similarity, candle.AgeSeconds(), p)
				if confidence >= p.ConfidenceThreshold {
					// Semantic Hit!
					// Backfill L1/L2a
					c.backfill(ctx, req.TenantID, normalized, candle.Answer)

					return &QueryResponse{
						Answer:     candle.Answer,
						Source:     "L2b",
						Hit:        true,
						Confidence: confidence,
						LatencyMS:  time.Since(start).Milliseconds(),
					}, nil
				}
			}
		}
	}

	// Step 4: Backend Call (with singleflight and circuit breaker)
	res, err := c.fetchFromBackend(ctx, req.TenantID, normalized, domain, req.Query, emb)
	if err != nil {
		return nil, err
	}

	return &QueryResponse{
		Answer:    res.Answer,
		Source:    "backend",
		Hit:       false,
		LatencyMS: time.Since(start).Milliseconds(),
	}, nil
}

func (c *Coordinator) fetchFromBackend(ctx context.Context, tenantID, normalized, domain, original string, embedding []float32) (*backend.Response, error) {
	// Deduplicate concurrent requests for the same exact normalized query
	key := fmt.Sprintf("%s:%s", tenantID, normalized)
	
	val, err, _ := c.sfGroup.Do(key, func() (interface{}, error) {
		// Call backend through circuit breaker
		var resp *backend.Response
		cbErr := c.breaker.Execute(func() error {
			var err error
			resp, err = c.backend.Query(ctx, original)
			return err
		})

		if cbErr != nil {
			return nil, cbErr
		}

		// Write-through to all tiers asynchronously
		go c.persist(context.Background(), tenantID, normalized, original, domain, resp.Answer, embedding)

		return resp, nil
	})

	if err != nil {
		return nil, err
	}

	return val.(*backend.Response), nil
}

func (c *Coordinator) persist(ctx context.Context, tenantID, normalized, original, domain, answer string, embedding []float32) {
	// 1. L1
	c.l1.Set(tenantID, normalized, answer, 1*time.Hour)

	// 2. L2a
	if c.l2a != nil {
		_ = c.l2a.Set(ctx, tenantID, normalized, answer, 24*time.Hour)
	}

	// 3. L2b
	if c.l2b != nil && len(embedding) > 0 {
		hash := sha256.Sum256([]byte(normalized))
		queryHash := hex.EncodeToString(hash[:])
		
		entry := &CacheEntry{
			TenantID:       tenantID,
			QueryRaw:       original,
			QueryNormalized: normalized,
			QueryHash:      queryHash,
			QueryDomain:    domain,
			Answer:         answer,
			TTLSeconds:     86400, // 24h
		}
		_ = c.l2b.Write(ctx, entry, embedding)
	}
}

func (c *Coordinator) backfill(ctx context.Context, tenantID, normalized, answer string) {
	// Quick backfill to L1 and L2a
	c.l1.Set(tenantID, normalized, answer, 1*time.Hour)
	if c.l2a != nil {
		go func() {
			_ = c.l2a.Set(context.Background(), tenantID, normalized, answer, 24*time.Hour)
		}()
	}
}
