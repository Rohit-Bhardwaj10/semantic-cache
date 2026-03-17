Intelligent Multi-Tier
Cache System
MVP Build Plan — Complete Technical Roadmap

Document type	MVP Build Plan
System	Semantic Cache Proxy (Go)
Version	1.0 — Greenfield build incorporating all remediation findings
Date	March 2026
Build phases	5 phases · 10 sprints · ~14 weeks to production MVP
Scope	Architecture, project structure, tech stack, sequenced build plan, concepts


1. What We Are Building
The Intelligent Multi-Tier Cache System is a reverse-proxy that sits between any client application and an expensive backend API (typically an LLM). It intercepts queries, determines whether a semantically equivalent query has been answered before, and returns the cached answer — saving cost and latency. Unlike a simple key-value cache, it understands meaning, not just exact text.
The MVP delivers the corrected, production-hardened version of the system described in the original technical document, incorporating all 17 remediation changes identified in the feasibility analysis.
1.1 Core value proposition
Without this system
•	1M queries/day × $0.003 = $3,000/day
•	500ms average latency per query
•	No reuse of semantically identical queries
•	"weather nyc" and "NYC weather" are different	With this system
•	78% hit rate → ~$660/day (75% savings)
•	8ms P99 latency on cache hits
•	Semantic matching catches paraphrases
•	Policy engine prevents wrong answers

1.2 Key concepts to understand before building
These are the conceptual building blocks. Every engineer on the team should understand all five before writing a line of code.
CONCEPT	Multi-tier caching (L1 → L2a → L2b → Backend)
Think of it like CPU cache hierarchy. L1 is in-RAM, sub-millisecond, limited capacity. L2a is Redis — fast, shared, survives restarts. L2b is Postgres with vector search — slower but can match semantically similar queries. Backend is the last resort. Each tier adds latency but increases hit surface. The system always tries the cheapest tier first.

CONCEPT	Embeddings and cosine similarity
An embedding is a list of ~768 numbers that represents the meaning of a sentence in mathematical space. Two semantically similar sentences will have vectors that point in nearly the same direction — cosine similarity measures this angle. A similarity of 1.0 means identical; 0.9 means very similar; below 0.7 is usually different meaning. The L2b tier converts every query to an embedding and finds the closest match in the database.

CONCEPT	Policy-driven cache reuse decisions
Just because two queries are 92% similar does not mean their answers are interchangeable. "Weather today" and "weather tomorrow" are 94% similar but require different answers. The policy engine applies domain-specific rules: minimum similarity threshold, maximum staleness, and temporal keyword mismatch detection. A cached answer is only served if ALL policy checks pass.

CONCEPT	Write-through caching with singleflight deduplication
When a query misses all tiers and reaches the backend, the answer is stored in all three tiers simultaneously (write-through). If two identical queries arrive within milliseconds of each other while the first is still in-flight to the backend, singleflight ensures only ONE backend call is made — both callers receive the same result. This prevents thundering-herd problems and duplicate backend spend.

CONCEPT	Circuit breaker pattern
If Ollama (the embedding service) or the backend starts failing, a naive system retries repeatedly, making things worse. A circuit breaker tracks failure rate and "opens" (stops trying) after a threshold is breached. While open, the system degrades gracefully: L2b is skipped and queries go straight to L1/L2a or the backend. The breaker transitions to "half-open" after a timeout and probes whether the service has recovered.


2. Technology Stack
Every technology choice is justified below. The stack is deliberately minimal for an MVP — no Kubernetes, no distributed tracing, no service mesh. Those come later.
2.1 Stack overview
Go 1.22
Core proxy	Redis 7.2
L2a cache	PostgreSQL 16
L2b + storage	pgvector
Vector search

Ollama
Embeddings	Docker Compose
Dev deployment	Prometheus
Metrics	Grafana
Dashboards

2.2 Why these choices
Go 1.22	Fast compilation, goroutines make concurrent multi-tier lookups natural, single binary deployment, excellent HTTP libraries. sync/singleflight is in the standard library.
Redis 7.2	Shared L2a cache across multiple proxy instances. Built-in TTL, pub/sub for invalidation events, fast enough for 2ms lookup SLO. Persistence means L2a survives proxy restarts.
PostgreSQL 16	Single database for all persistent state (cache entries, audit log, policy versions). ACID transactions, familiar SQL, free and open-source, easy to operate. No additional vector DB needed.
pgvector ext.	Adds VECTOR type and cosine distance operator (<=>). HNSW index gives approximate nearest-neighbour search at ~5ms. Keeps the architecture simple — one DB does everything.
Ollama	Local embedding inference at 8ms vs 100ms+ for cloud APIs. Zero per-call cost. No rate limits. nomic-embed-text model produces 768-dimensional vectors with excellent semantic quality. Runs on CPU (slower) or GPU.
Docker Compose	For MVP development, a single compose file is sufficient. All five services (proxy, redis, postgres, ollama, grafana) start with one command. Kubernetes manifests are planned for Phase 5.
Prometheus	Pull-based metrics scraping. Go has an official client library. Zero additional infrastructure needed — Prometheus scrapes the /metrics endpoint directly.
Grafana	Dashboards on top of Prometheus. Runs as a Docker container. Provides the eight dashboard panels described in the original specification.


3. Project Structure
The structure follows Go's standard project layout. Every package has a single clear responsibility. New engineers should be able to locate any piece of logic within 30 seconds.
semantic-cache/
├── cmd/
│   └── server/
│       └── main.go              # Entry point: wires all deps, starts HTTP server
├── internal/
│   ├── api/
│   │   ├── server.go            # HTTP server setup, middleware chain, route registration
│   │   ├── middleware.go         # Auth (JWT), rate limiting, request ID injection
│   │   ├── handlers.go          # HTTP handlers: query, stream, health, analytics
│   │   ├── feedback.go          # POST /feedback — marks answers as correct/incorrect
│   │   └── models.go            # Request/response structs (QueryRequest, CacheResponse...)
│   ├── cache/
│   │   ├── entry.go             # CacheEntry struct, IsExpired()
│   │   ├── errors.go            # Typed cache errors (ErrMiss, ErrEmbeddingUnavailable...)
│   │   ├── normalize.go         # L0: lowercase, contractions, synonyms (NO word sort)
│   │   ├── l1.go                # In-memory LRU, memory-budget eviction (bytes, not count)
│   │   ├── l2a.go               # Redis normalized-match tier
│   │   ├── l2b.go               # Postgres vector search tier
│   │   ├── coordinator.go       # Orchestrates L0→L1→L2a→L2b→backend, singleflight
│   │   ├── warmup.go            # Startup cache warming from Postgres access history
│   │   └── invalidation.go      # Redis pub/sub invalidation listener
│   ├── policy/
│   │   ├── types.go             # Policy struct: MinSim, MaxStaleness, SimWeight, FreshWeight
│   │   ├── engine.go            # Policy loader from YAML, hot-reload on SIGHUP
│   │   ├── scoring.go           # CalculateConfidence() — additive weighted formula
│   │   ├── temporal.go          # TemporalClass(): detect today/tomorrow/now keywords
│   │   ├── classifier.go        # DomainClassifier: keyword fast-path + centroid fallback
│   │   └── tuner.go             # Adaptive threshold tuner (Phase 4)
│   ├── backend/
│   │   ├── client.go            # Backend interface + HTTP implementation
│   │   └── mock.go              # Mock backend for tests and development
│   ├── resilience/
│   │   └── circuit_breaker.go   # Generic circuit breaker: closed/open/half-open states
│   ├── audit/
│   │   └── logger.go            # Append-only structured audit log (hashed query, no PII)
│   └── metrics/
│       └── prometheus.go        # All metric definitions and helper recording functions
├── pkg/
│   └── embeddings/
│       └── ollama.go            # Ollama HTTP client + embedding Redis cache
├── configs/
│   ├── policies.yaml            # Per-domain policy: MinSim, MaxStaleness, SimWeight...
│   └── synonyms.yaml            # Synonym mappings: nyc→new york city, etc.
├── migrations/
│   ├── 001_initial_schema.sql   # Base schema: cache_entries with all corrected columns
│   └── 002_add_tenant.sql       # tenant_id column + scoped indexes
├── tests/
│   ├── unit/
│   │   ├── normalize_test.go
│   │   ├── scoring_test.go
│   │   ├── temporal_test.go
│   │   └── l1_test.go
│   └── integration/
│       ├── cache_flow_test.go   # Full L0→L1→L2b→backend test with test containers
│       └── policy_test.go
├── k8s/                         # Kubernetes manifests (Phase 5, not MVP)
│   └── .gitkeep
├── docker-compose.yml           # All 5 services for local development
├── schema.sql                   # Canonical schema (generated from migrations)
└── README.md

3.1 Key architectural decisions in the structure
•	internal/ vs pkg/: packages in internal/ are private to this module. Only the Ollama client lives in pkg/ because it could realistically be extracted as a standalone library.
•	No circular dependencies: api → cache → policy → (no upward deps). Metrics and audit are leaf packages imported by anything that needs them.
•	migrations/ directory: schema changes are versioned SQL files, not applied ad-hoc. Run with golang-migrate or a simple shell script.
•	configs/ separation: policies.yaml and synonyms.yaml are hot-reloadable. Changing a similarity threshold should not require a redeploy.


4. Database Schema
The complete corrected schema incorporates all findings from the remediation analysis: tenant isolation, embedding model versioning, TTL column, access tracking, and the HNSW index.
-- migrations/001_initial_schema.sql

CREATE EXTENSION IF NOT EXISTS vector;

CREATE TABLE cache_entries (
    id               BIGSERIAL PRIMARY KEY,
    tenant_id        TEXT        NOT NULL DEFAULT 'default',
    query_original   TEXT        NOT NULL,
    query_normalized TEXT        NOT NULL,
    query_domain     TEXT        NOT NULL DEFAULT 'general',
    embedding        VECTOR(768) NOT NULL,
    embed_model      TEXT        NOT NULL DEFAULT 'nomic-embed-text',
    embed_version    TEXT        NOT NULL DEFAULT 'v1',
    answer           TEXT        NOT NULL,
    ttl_seconds      INTEGER     NOT NULL DEFAULT 3600,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    last_accessed_at TIMESTAMPTZ,
    access_count     INTEGER     NOT NULL DEFAULT 0
);

-- Scoped HNSW index for fast per-tenant vector search
CREATE INDEX cache_entries_embedding_idx
    ON cache_entries USING hnsw (embedding vector_cosine_ops);

-- Supporting indexes for maintenance and analytics
CREATE INDEX ON cache_entries (tenant_id, query_domain);
CREATE INDEX ON cache_entries (created_at);
CREATE INDEX ON cache_entries (tenant_id, access_count DESC);

-- Autovacuum tuning for high-churn table
ALTER TABLE cache_entries SET (
    autovacuum_vacuum_scale_factor  = 0.01,
    autovacuum_analyze_scale_factor = 0.005
);

-- Audit log (separate from cache — append-only, never updated)
CREATE TABLE audit_log (
    id          BIGSERIAL   PRIMARY KEY,
    request_id  TEXT        NOT NULL,
    tenant_id   TEXT        NOT NULL,
    query_hash  TEXT        NOT NULL,  -- SHA-256, no PII
    domain      TEXT,
    decision    TEXT        NOT NULL,  -- L1_hit / L2b_accept / backend
    reason      TEXT,
    confidence  REAL,
    client_ip   TEXT,
    ts          TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
CREATE INDEX ON audit_log (tenant_id, ts DESC);


5. Corrected Algorithm Specification
This section defines the exact algorithms that must be implemented. Any deviation from these specifications reproduces the flaws identified in the original system.
5.1 L0 Normalization pipeline (corrected)
Run in this exact order. Step 5 (word sort) from the original document is permanently removed.
// internal/cache/normalize.go
func Normalize(q string, synonyms map[string]string) string {
    q = strings.ToLower(q)           // Step 1: lowercase
    q = expandContractions(q)         // Step 2: what's → what is
    q = removePunctuation(q)          // Step 3: strip non-alphanumeric
    q = applySynonyms(q, synonyms)    // Step 4: nyc → new york city
    q = collapseWhitespace(q)         // Step 5: normalize spaces
    // NEVER sort words — word order carries semantic meaning
    return strings.TrimSpace(q)
}

5.2 Confidence scoring (corrected additive formula)
The original multiplicative formula collapsed to near-zero under realistic conditions. The corrected formula uses configurable per-domain weights:
// internal/policy/scoring.go
func CalculateConfidence(sim float32, ageSeconds int, p Policy) float32 {
    // Hard gates — return 0 immediately if these fail
    if sim < p.MinSimilarity { return 0 }

    // Exponential freshness decay
    freshness := float32(math.Exp(
        -float64(ageSeconds) / float64(p.MaxStalenessSeconds),
    ))

    // Weighted additive combination (NOT multiplicative)
    // SimWeight + FreshWeight should sum to 1.0
    return p.SimWeight*sim + p.FreshWeight*freshness
}

// Example policy config (configs/policies.yaml):
// weather:
//   min_similarity: 0.88
//   max_staleness_seconds: 1800
//   confidence_threshold: 0.72
//   sim_weight: 0.40
//   fresh_weight: 0.60

5.3 Temporal mismatch detection
// internal/policy/temporal.go
var temporalMap = map[string]string{
    "today": "present",   "tonight": "present",
    "now": "present",     "currently": "present",
    "right now": "present",
    "tomorrow": "future", "next week": "future",
    "next month": "future",
    "yesterday": "past",  "last week": "past",
    "last month": "past",
}

func TemporalClass(query string) string {
    q := strings.ToLower(query)
    for kw, class := range temporalMap {
        if strings.Contains(q, kw) { return class }
    }
    return "" // no temporal marker
}

// Called in coordinator before policy scoring:
// if TemporalClass(incoming) != TemporalClass(cached) → reject

5.4 Full request flow (corrected)
Every query follows this exact path. Deviating from the sequence breaks the hit rate or correctness guarantees.
1.	L0: Normalize the query (preserving word order). Use normalized form as L1 and L2a key.
2.	L1: Check in-memory LRU. Key = tenantID + ":" + normalized. Hit → return immediately (100μs).
3.	L2a: Check Redis. Key = "norm:" + tenantID + ":" + normalized. Hit → backfill L1, return (2ms).
4.	Embedding: Call Ollama (or return from embedding Redis cache). ~8ms on cache miss.
5.	L2b: Query Postgres HNSW index for top-5 candidates with cosine similarity.
6.	Temporal check: For each candidate, compare TemporalClass(incoming) vs TemporalClass(cached). Mismatch → skip.
7.	Policy scoring: CalculateConfidence(sim, age, policy). Confidence >= threshold → accept.
8.	If accepted: backfill L1 + L2a. Return answer with source=L2b.
9.	Circuit breaker: Check backend circuit breaker state before proceeding.
10.	Backend: Call backend API. On success: write-through all tiers asynchronously via singleflight.


6. Phased Build Plan
The 5 phases are sequenced so that each phase produces working, testable software. Never start a phase until the previous one has passing tests. Each phase builds on the last and nothing is throwaway — every component written in Phase 1 ships in the final MVP.
Phase	Name	Duration	Deliverable
Phase 1	Data layer + algorithms	Weeks 1–2	Schema, normalization, confidence scoring, unit tests
Phase 2	Cache tiers + coordinator	Weeks 3–4	Working L1/L2a/L2b, singleflight, circuit breakers
Phase 3	API + security	Weeks 5–7	HTTP API, auth, tenant isolation, rate limiting
Phase 4	Observability + correctness	Weeks 8–10	Metrics, Grafana, audit log, invalidation, warming
Phase 5	Hardening + advanced	Weeks 11–14	K8s, streaming, adaptive tuning, A/B testing

PHASE 1
Data Layer & Core Algorithms
Weeks 1–2 · 2 sprints   —   Foundation everything else builds on
S1	Sprint 1: Project scaffolding and schema	
S1	Init Go module: go mod init github.com/[user]/semantic-cache	
S1	Create full directory structure as defined in section 3	
S1	Write docker-compose.yml: postgres (+ pgvector), redis, ollama, prometheus, grafana	
S1	Write migrations/001_initial_schema.sql (full corrected schema from section 4)	
S1	Write a db/migrate.go helper that applies all migrations on startup	
S2	Sprint 2: Core algorithm package	
S2	internal/cache/normalize.go — corrected pipeline, no word sort, synonym loading	
S2	internal/policy/temporal.go — TemporalClass() function	
S2	internal/policy/scoring.go — additive CalculateConfidence()	
S2	internal/policy/engine.go — load policies.yaml, hot-reload on SIGHUP	
S2	internal/policy/classifier.go — keyword + centroid domain auto-detection	
S2	Write unit tests for all of the above: 100% coverage target on algorithm code	
S2	Validate: go test ./internal/policy/... ./internal/cache/... must pass	

KEY	Why algorithms first?
All three cache tiers depend on normalization and scoring. If you build the cache first and fix the algorithm later, you must invalidate all stored data and re-tune thresholds. Building and testing algorithms in isolation (no HTTP, no DB) means defects are caught cheaply at unit test time, not after 10,000 entries have been cached with a broken formula.

PHASE 2
Cache Tiers & Coordinator
Weeks 3–4 · 2 sprints   —   The actual caching machinery
S3	Sprint 3: Individual cache tier implementations	
S3	pkg/embeddings/ollama.go — HTTP client, embedding result Redis cache, error types	
S3	internal/resilience/circuit_breaker.go — generic closed/open/half-open implementation	
S3	internal/cache/l1.go — LRU with memory-budget eviction (bytes, not entry count)	
S3	internal/cache/l2a.go — Redis GET/SET with TTL, tenant-scoped keys	
S3	internal/cache/l2b.go — Postgres HNSW search, embedding model version check	
S3	Write unit tests for L1 (eviction correctness, memory budget, TTL expiry)	
S4	Sprint 4: Coordinator and write-through	
S4	internal/cache/coordinator.go — orchestrate L0→L1→L2a→L2b→backend	
S4	Add golang.org/x/sync/singleflight — replace inflight.go stub with real impl	
S4	internal/cache/invalidation.go — Redis pub/sub listener, propagate to all tiers	
S4	internal/cache/warmup.go — load top 5000 entries from Postgres on startup	
S4	Integration test: full cache miss → backend → write-through → cache hit flow	
S4	Integration test: concurrent identical queries produce exactly one backend call	
S4	Integration test: temporal mismatch (today vs tomorrow) never serves cached answer	
S4	Validate: docker-compose up, run integration tests against live containers	

KEY	Singleflight is not optional
Under 1,000 QPS with a 500ms backend, dozens of concurrent requests for the same popular query will all miss cache simultaneously without singleflight. Each one generates a backend call, multiplying cost and defeating the purpose of the system. golang.org/x/sync/singleflight deduplicates in-flight requests to a single backend call — all waiters receive the same result when it resolves.

PHASE 3
HTTP API & Security
Weeks 5–7 · 3 sprints   —   The public surface — must be correct before anything goes near production
S5	Sprint 5: Core API endpoints	
S5	cmd/server/main.go — wire all dependencies, graceful shutdown on SIGTERM	
S5	internal/api/server.go — HTTP server, middleware chain (auth → rate limit → request ID → handler)	
S5	internal/api/handlers.go — POST /cache/query (buffered response)	
S5	GET /health (shallow), GET /readyz (deep: checks redis/postgres/ollama), GET /livez (process alive)	
S5	GET /metrics (Prometheus exposition format)	
S5	GET /analytics/cost-savings (net savings = gross - infra costs)	
S6	Sprint 6: Authentication and tenant isolation	
S6	internal/api/middleware.go — JWT validation, extract tenant_id from claims	
S6	Add per-client rate limiting middleware (token bucket, configurable via env)	
S6	Thread tenant_id through all cache operations (L1 key prefix, Redis key prefix, SQL WHERE)	
S6	migrations/002_add_tenant.sql — add tenant_id column, backfill existing rows	
S6	POST /admin/invalidate — pattern-based invalidation, propagate all tiers (requires admin JWT)	
S6	POST /admin/reload-policies — hot reload YAML from disk without restart	
S7	Sprint 7: Input validation, audit, and streaming	
S7	internal/api/handlers.go — add ValidateQueryRequest(), ShouldCache(), SanitizeAnswer()	
S7	internal/audit/logger.go — structured audit events, hashed query (no PII), append-only writes	
S7	internal/api/handlers.go — StreamQuery() handler using http.Flusher + SSE format	
S7	internal/api/feedback.go — POST /feedback endpoint, store signal in Postgres	
S7	End-to-end API tests: auth failure, rate limit, tenant isolation, input validation	

KEY	Tenant isolation is a schema migration — plan it carefully
Adding tenant_id to cache_entries is a breaking schema change. In production you would need to backfill existing rows, update all indexes, and deploy the new code atomically. In the MVP development environment, it is cleanest to drop and recreate the database when applying migration 002. Document this clearly so the team does not run the migration against a populated dev database without reading the notes.

PHASE 4
Observability & Operational Correctness
Weeks 8–10 · 2 sprints   —   You cannot operate what you cannot see
S8	Sprint 8: Metrics and dashboards	
S8	internal/metrics/prometheus.go — define all counters, histograms, gauges	
S8	Instrument coordinator: cache_hits_total{tier}, cache_misses_total{tier}, backend_calls_total{domain}	
S8	Instrument scoring: l2b_confidence_score (histogram), policy_rejection_total{reason}	
S8	Instrument latency: cache_latency_seconds (histogram by tier), embedding_duration_seconds	
S8	Instrument cost: backend_cost_usd_total, cache_cost_saved_usd_total	
S8	Create Grafana dashboard JSON: 8 panels matching original spec + FPR + SLO burn rate	
S9	Sprint 9: Embedding versioning, maintenance, and SLOs	
S9	Startup check: query Postgres for embed_model/embed_version mismatches, refuse to start if found	
S9	scripts/maintenance.sql — weekly DELETE expired + REINDEX CONCURRENTLY + VACUUM ANALYZE	
S9	Add maintenance cron job to docker-compose (using pg_cron extension or external scheduler)	
S9	configs/alerts.yaml — Prometheus alerting rules for all 6 SLOs from remediation document	
S9	Fix benchmark: single canonical run config, reconcile P99 (12ms in benchmarks, not 8ms in summary)	
S9	Fix cost savings: deduct Ollama + Redis + Postgres infra costs from gross savings	
S9	Load test: 1000 QPS sustained for 10 min, confirm P99 < 20ms (cache hit) and > 70% hit rate	

KEY	Define SLOs before the load test, not after
Without defined SLOs, a load test only tells you what the numbers are. With SLOs, it tells you whether the system passes or fails. Define the six SLOs from the remediation document in configs/alerts.yaml before running the load test in Sprint 9. If any SLO is breached, it becomes a blocking bug for Phase 5.

PHASE 5
Hardening & Advanced Features
Weeks 11–14 · last sprints   —   Production deployment + best-in-class capabilities
S10	Sprint 10: Production deployment and advanced features	
S10	k8s/ — Deployment (replicas:3), HPA (min3/max20), PodDisruptionBudget, Service manifests	
S10	Update README: label docker-compose as dev-only, document kubectl apply steps	
S10	internal/policy/tuner.go — PolicyTuner: 7-day rolling FPR window, threshold ±0.005/0.01 adjustment	
S10	Wire tuner to /feedback signals, run nightly via time.Ticker goroutine	
S10	internal/policy/experiment.go — shadow A/B mode: evaluate control + candidate, never use candidate for serving	
S10	Security review: pen-test auth endpoints, verify tenant isolation (cross-tenant query test)	
S10	Final load test: 1000 QPS × 10 min, all 6 SLOs must pass — this is the go/no-go gate for MVP	
S10	Tag v1.0.0, write CHANGELOG, update all README metrics to match canonical benchmark	


7. Build Sequence & Dependency Map
This shows which components must exist before others can be built. Never build a component before its dependencies are tested.
Component	Depends on	What breaks if built out of order
normalize.go	synonyms.yaml only	Word-sort bug re-introduced; all downstream tiers use wrong keys
scoring.go	policy types.go	Multiplicative formula; near-zero confidence; 0% L2b hit rate
temporal.go	normalize.go	Time-keyword mismatches served as hits; wrong answers silently
l1.go	cache/entry.go	Memory OOM under long LLM responses (10× underestimation)
ollama.go (pkg)	resilience/circuit_breaker	No fallback if Ollama down; every request errors at L2b
l2b.go	ollama.go, schema migrated	Wrong embedding keys if model not tracked; silent version drift
coordinator.go	l1, l2a, l2b, singleflight	Race condition: N concurrent requests → N backend calls
middleware.go (auth)	JWT library configured	Unauthenticated endpoint; cache poisoning attack surface
handlers.go	coordinator, middleware	No auth; no tenant scoping; poisoning possible
metrics/prometheus.go	handlers.go wired	No visibility; cannot verify SLOs; cannot detect regressions
tuner.go	feedback endpoint, metrics	Tunes without signal data; thresholds drift randomly


8. API Contract
Define the API contract before implementation. Frontend, backend integration, and test code should all be written against these specs, not against the implementation.
8.1 POST /cache/query
Auth	Bearer JWT required
Rate limit	1000 req/min per client
Content-Type	application/json

// Request
{
  "query":  "What's the weather in NYC?",
  "domain": "weather"   // optional — auto-classified if omitted
}

// Response — cache hit
{
  "answer":        "Sunny, 22C, humidity 45%",
  "source":        "L2b",
  "hit":           true,
  "confidence":    0.78,
  "latency_ms":    13,
  "cached_query":  "How's NYC weather?",
  "similarity":    0.91,
  "age_seconds":   300
}

// Response — cache miss
{
  "answer":      "Sunny, 22C, humidity 45%",
  "source":      "backend",
  "hit":         false,
  "latency_ms":  487,
  "cost_usd":    0.003
}

8.2 GET /readyz (deep health check)
// 200 OK when all dependencies are healthy
{
  "status": "ready",
  "services": {
    "redis":    { "ok": true,  "latency_ms": 1 },
    "postgres": { "ok": true,  "latency_ms": 3 },
    "ollama":   { "ok": true,  "latency_ms": 8 },
    "backend":  { "ok": true,  "latency_ms": 45 }
  }
}

// 503 Service Unavailable when any critical dep is down
{ "status": "not_ready", "reason": "ollama: connection refused" }

8.3 GET /analytics/cost-savings
{
  "period":             "last_30_days",
  "total_queries":       1000000,
  "cache_hits":          780000,
  "hit_rate":            0.78,
  "gross_savings_usd":   2340.00,
  "infra_cost_usd":      180.00,
  "net_savings_usd":     2160.00,
  "roi_percent":         1200,
  "breakeven_qps":       12.4
}


9. Development Environment
The complete docker-compose.yml for local development. All five services start with docker-compose up -d.
version: '3.9'
services:

  cache-proxy:
    build: .
    ports: ["8080:8080", "9090:9090"]  # API + metrics
    environment:
      REDIS_URL:        redis:6379
      POSTGRES_URL:     postgres://cache:cache@postgres:5432/cache
      OLLAMA_URL:       http://ollama:11434
      BACKEND_URL:      http://mock-backend:8081
      L1_MAX_BYTES:     134217728  # 128MB
      L1_DEFAULT_TTL:   3600
      CB_MAX_FAILURES:  5
      CB_TIMEOUT:       30s
      LOG_LEVEL:        info
      JWT_SECRET:       dev-secret-change-in-prod
    depends_on: [redis, postgres, ollama]

  postgres:
    image: pgvector/pgvector:pg16
    environment: { POSTGRES_DB: cache, POSTGRES_USER: cache, POSTGRES_PASSWORD: cache }
    volumes: [pgdata:/var/lib/postgresql/data]
    ports: ["5432:5432"]

  redis:
    image: redis:7.2-alpine
    command: redis-server --appendonly yes
    volumes: [redisdata:/data]
    ports: ["6379:6379"]

  ollama:
    image: ollama/ollama:latest
    ports: ["11434:11434"]
    volumes: [ollamadata:/root/.ollama]
    # Pull model on first start:
    # docker exec ollama ollama pull nomic-embed-text

  prometheus:
    image: prom/prometheus:latest
    volumes: ["./configs/prometheus.yml:/etc/prometheus/prometheus.yml"]
    ports: ["9091:9090"]

  grafana:
    image: grafana/grafana:latest
    ports: ["3000:3000"]
    environment: { GF_SECURITY_ADMIN_PASSWORD: admin }

volumes: { pgdata, redisdata, ollamadata }


10. MVP Definition of Done
The MVP is complete when ALL of the following are true. No partial credit.
10.1 Functional requirements
1.	POST /cache/query returns correct answers for all query types (exact, normalized, semantic, temporal)
2.	L1 hit rate > 45% on a repeated-query workload (50% exact repeats)
3.	L2b does NOT return a cached answer when queries have different temporal classes
4.	Two concurrent identical requests (arriving < 50ms apart) produce exactly ONE backend call
5.	Unauthenticated requests to /cache/query return HTTP 401
6.	Queries from tenant A cannot retrieve cached answers belonging to tenant B
7.	System starts up with a warm cache (top 5000 entries loaded) within 30 seconds
8.	When Ollama is unavailable, queries fall through to backend without 500 errors

10.2 SLO requirements (measured over 10-minute load test at 1000 QPS)
1.	P99 latency (cache hit) < 20ms
2.	P99 latency (backend path) < 700ms
3.	Overall cache hit rate > 70%
4.	Error rate < 0.1%
5.	False positive rate (wrong answers served) < 2%

10.3 Operational requirements
•	All 8 Grafana dashboard panels render with real data
•	All 6 Prometheus alerting rules defined and firing correctly on synthetic failures
•	/readyz returns 503 within 5 seconds of postgres being stopped
•	HNSW maintenance SQL script runs without error on a populated database
•	docker-compose up --build starts all services cleanly from a fresh checkout
•	README documents: setup, first query, Grafana URL, how to add a domain policy

10.4 Code quality requirements
•	go vet ./... and golangci-lint run ./... produce zero errors
•	Unit test coverage >= 80% for internal/policy/ and internal/cache/normalize.go
•	Integration tests pass with real Docker containers (testcontainers-go)
•	No hardcoded secrets anywhere in the codebase
•	All environment variables documented in README with example values


End of MVP Build Plan. Start at Phase 1 Sprint 1 and work forward sequentially. Each sprint's tests must pass before the next sprint begins.
