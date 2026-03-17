# semantic-cache — Build Progress Tracker

> **Rule:** Never start a phase until the previous one has passing tests.  
> Each sprint's tests must pass before the next sprint begins.  
> Updated as work progresses — check off tasks as they are completed.

---

## Legend
- `[ ]` — Not started  
- `[~]` — In progress  
- `[x]` — Done  
- `[!]` — Blocked / needs attention

---

## Current Status

| Phase | Status | Sprint | Started | Completed |
|---|---|---|---|---|
| Phase 1 — Data Layer & Algorithms | 🟡 In Progress | S2 | 2026-03-18 | — |
| Phase 2 — Cache Tiers & Coordinator | ⬜ Not started | — | — | — |
| Phase 3 — HTTP API & Security | ⬜ Not started | — | — | — |
| Phase 4 — Observability & Correctness | ⬜ Not started | — | — | — |
| Phase 5 — Hardening & Advanced | ⬜ Not started | — | — | — |

---

## Phase 1 — Data Layer & Core Algorithms
**Weeks 1–2 · 2 sprints · Foundation everything else builds on**

> ⚠️ Algorithms MUST come before cache tiers. Normalization and scoring bugs caught here are cheap. Caught after 10,000 cached entries they require full data invalidation and re-tuning.

### Sprint 1 — Project Scaffolding & Schema

#### Project Setup
- [x] `go mod init github.com/[user]/semantic-cache`
- [x] Create full directory structure as defined in Build_plan.md §3
  - [x] `cmd/server/`
  - [x] `internal/api/`
  - [x] `internal/cache/`
  - [x] `internal/policy/`
  - [x] `internal/backend/`
  - [x] `internal/resilience/`
  - [x] `internal/audit/`
  - [x] `internal/metrics/`
  - [x] `pkg/embeddings/`
  - [x] `configs/`
  - [x] `migrations/`
  - [x] `tests/unit/`
  - [x] `tests/integration/`
  - [x] `k8s/` (placeholder with `.gitkeep`)

#### Docker & Infrastructure
- [x] Write `docker-compose.yml` with all 5 services:
  - [x] `postgres` (image: `pgvector/pgvector:pg16`) — with healthcheck
  - [x] `redis` (image: `redis:7.2-alpine`, append-only persistence) — with healthcheck
  - [x] `ollama` (image: `ollama/ollama:latest`)
  - [x] `prometheus` (image: `prom/prometheus:latest`)
  - [x] `grafana` (image: `grafana/grafana:latest`) — datasource auto-provisioned
- [x] `Dockerfile` — multi-stage build (Go builder → Alpine runtime)
- [x] `Makefile` — `make up`, `make down`, `make ollama-pull`, `make psql`, `make test`, etc.
- [ ] Verify `docker-compose up -d` starts all 5 services cleanly
- [ ] Pull Ollama model: `make ollama-pull`

#### Database Schema
- [x] Write `migrations/001_initial_schema.sql`:
  - [x] `CREATE EXTENSION IF NOT EXISTS vector`
  - [x] `cache_entries` table (all corrected columns: `tenant_id`, `embed_model`, `embed_version`, `ttl_seconds`, `last_accessed_at`, `access_count`)
  - [x] HNSW index on `embedding` column (`vector_cosine_ops`) — m=16, ef_construction=64
  - [x] Supporting indexes: `(tenant_id, query_domain)`, `(created_at)`, `(tenant_id, access_count DESC)`
  - [x] Autovacuum tuning (`vacuum_scale_factor=0.01`, `analyze_scale_factor=0.005`)
  - [x] `audit_log` table (append-only, `query_hash` SHA-256, no PII)
  - [x] `feedback` table (for Phase 5 adaptive tuner)
  - [x] Index on `audit_log (tenant_id, ts DESC)`
- [x] Write `migrations/002_add_tenant.sql` — idempotent tenant_id migration
- [x] Write `db/migrate.go` helper — idempotent, tracks applied files in `schema_migrations` table
- [ ] Verify schema applies cleanly against live Postgres container

#### Configs
- [x] `configs/prometheus.yml` — scrape config targeting `cache-proxy:9090/metrics`
- [x] `configs/policies.yaml` — 7 domains (weather, finance, news, coding, medical, legal, general)
- [x] `configs/synonyms.yaml` — geographic, tech, finance, and common abbreviations
- [x] `configs/grafana/provisioning/datasources/prometheus.yaml` — auto-provisions Prometheus datasource

#### Sprint 1 Gate ✅
- [ ] `docker-compose up --build` starts all services with zero errors
- [ ] Schema applies without errors on fresh Postgres
- [ ] `make ollama-pull` succeeds and model is available

---

### Sprint 2 — Core Algorithm Package

#### `internal/cache/`
- [x] `internal/cache/entry.go` — `CacheEntry` struct, `IsExpired()` method
- [ ] `internal/cache/errors.go` — Typed errors: `ErrMiss`, `ErrEmbeddingUnavailable`, etc.
- [ ] `internal/cache/normalize.go` — Corrected L0 pipeline (in this exact order):
  - [ ] Step 1: `strings.ToLower()`
  - [ ] Step 2: `expandContractions()` — `what's → what is`, etc.
  - [ ] Step 3: `removePunctuation()` — strip non-alphanumeric
  - [ ] Step 4: `applySynonyms()` — load from `configs/synonyms.yaml`
  - [ ] Step 5: `collapseWhitespace()`
  - [ ] ⚠️ NO word sort — word order carries semantic meaning

#### `internal/policy/`
- [ ] `internal/policy/types.go` — `Policy` struct: `MinSimilarity`, `MaxStalenessSeconds`, `ConfidenceThreshold`, `SimWeight`, `FreshWeight`
- [ ] `internal/policy/temporal.go` — `TemporalClass(query string) string`:
  - [ ] `present` keywords: `today`, `tonight`, `now`, `currently`, `right now`
  - [ ] `future` keywords: `tomorrow`, `next week`, `next month`
  - [ ] `past` keywords: `yesterday`, `last week`, `last month`
  - [ ] Returns `""` if no temporal marker found
- [ ] `internal/policy/scoring.go` — `CalculateConfidence(sim float32, ageSeconds int, p Policy) float32`:
  - [ ] Hard gate: if `sim < p.MinSimilarity` return `0`
  - [ ] Exponential freshness decay: `math.Exp(-ageSeconds / MaxStalenessSeconds)`
  - [ ] ⚠️ Additive formula: `SimWeight*sim + FreshWeight*freshness` (NOT multiplicative)
- [ ] `internal/policy/engine.go`:
  - [ ] Load `configs/policies.yaml` on startup
  - [ ] Hot-reload on `SIGHUP` signal (no restart required)
- [ ] `internal/policy/classifier.go` — `DomainClassifier`:
  - [ ] Keyword fast-path for known domains
  - [ ] Centroid-based vector fallback for unknown domains

#### `configs/`
- [ ] `configs/synonyms.yaml` — synonym mappings (e.g. `nyc → new york city`, `ml → machine learning`)
- [ ] `configs/policies.yaml` — per-domain policy config:
  - [ ] `weather` domain: `min_similarity: 0.88`, `max_staleness_seconds: 1800`, `confidence_threshold: 0.72`, `sim_weight: 0.40`, `fresh_weight: 0.60`
  - [ ] `general` domain (default)
  - [ ] Additional domains as needed

#### Unit Tests — Sprint 2
- [ ] `tests/unit/normalize_test.go`:
  - [ ] Test lowercase
  - [ ] Test contraction expansion
  - [ ] Test punctuation removal
  - [ ] Test synonym substitution
  - [ ] Test word-order is preserved (not sorted)
  - [ ] Test `"weather nyc"` ≠ `"nyc weather"` after normalization
- [ ] `tests/unit/scoring_test.go`:
  - [ ] Test hard gate (sim below min → 0)
  - [ ] Test freshness decay at t=0, t=half-life, t=2×half-life
  - [ ] Test additive formula with known weights
  - [ ] Test that old multiplicative formula would have failed these cases
- [ ] `tests/unit/temporal_test.go`:
  - [ ] Test each `present`/`future`/`past` keyword
  - [ ] Test no temporal marker returns `""`
  - [ ] Test mismatch detection: `"today"` vs `"tomorrow"` → reject
- [ ] `tests/unit/l1_test.go` — stubbed (L1 implementation is Phase 2, but stub tests now)

#### Sprint 2 Gate ✅
- [ ] `go test ./internal/policy/... ./internal/cache/...` — all pass
- [ ] 100% test coverage on `normalize.go`, `scoring.go`, `temporal.go`
- [ ] Zero `go vet` warnings on Phase 1 packages

---

## Phase 2 — Cache Tiers & Coordinator
**Weeks 3–4 · 2 sprints · The actual caching machinery**

> ⚠️ Do not begin Phase 2 until Sprint 2 gate is fully green.

### Sprint 3 — Individual Cache Tier Implementations

#### Embeddings & Resilience
- [ ] `pkg/embeddings/ollama.go`:
  - [ ] HTTP client to Ollama (`POST /api/embeddings`)
  - [ ] Embedding result Redis cache (avoid re-embedding same query)
  - [ ] Typed error types for connection failures
- [ ] `internal/resilience/circuit_breaker.go` — Generic circuit breaker:
  - [ ] `Closed` state: normal operation, counting failures
  - [ ] `Open` state: fast-fail, stop calling downstream
  - [ ] `Half-open` state: probe one request after timeout
  - [ ] Configurable: `CB_MAX_FAILURES`, `CB_TIMEOUT` from env

#### Cache Tiers
- [ ] `internal/cache/l1.go` — In-memory LRU:
  - [ ] Memory-budget eviction (max bytes, NOT max entry count)
  - [ ] Key format: `tenantID + ":" + normalizedQuery`
  - [ ] TTL expiry check on read (`IsExpired()`)
  - [ ] Thread-safe (`sync.RWMutex` or equivalent)
- [ ] `internal/cache/l2a.go` — Redis tier:
  - [ ] `GET` / `SET` with TTL
  - [ ] Tenant-scoped key format: `"norm:" + tenantID + ":" + normalizedQuery`
  - [ ] Graceful handling when Redis is unreachable
- [ ] `internal/cache/l2b.go` — Postgres vector search tier:
  - [ ] HNSW cosine search: `SELECT ... ORDER BY embedding <=> $1 LIMIT 5`
  - [ ] Embedding model version check (reject entries with different `embed_model`/`embed_version`)
  - [ ] Return top-5 candidates for policy evaluation
- [ ] `internal/backend/client.go` — Backend interface + HTTP implementation
- [ ] `internal/backend/mock.go` — Mock backend for tests and development

#### Unit Tests — Sprint 3
- [ ] `tests/unit/l1_test.go`:
  - [ ] Memory budget eviction (adding entries beyond budget triggers eviction)
  - [ ] LRU eviction order (least recently used is evicted first)
  - [ ] TTL expiry (expired entries are not returned)
  - [ ] Thread-safety (concurrent reads/writes)

#### Sprint 3 Gate ✅
- [ ] All Sprint 3 unit tests pass
- [ ] `ollama.go` can successfully embed a test query against live Ollama container
- [ ] Circuit breaker transitions correctly through closed → open → half-open states

---

### Sprint 4 — Coordinator & Write-Through

#### Coordinator
- [ ] `internal/cache/coordinator.go` — Orchestrate full flow:
  - [ ] Step 1: L0 Normalize
  - [ ] Step 2: L1 check
  - [ ] Step 3: L2a check → backfill L1 on hit
  - [ ] Step 4: Get embedding (Ollama + Redis cache)
  - [ ] Step 5: L2b search (top-5 candidates)
  - [ ] Step 6: Temporal check per candidate (`TemporalClass` mismatch → skip)
  - [ ] Step 7: Policy scoring (`CalculateConfidence` ≥ threshold → accept)
  - [ ] Step 8: On L2b hit → backfill L1 + L2a
  - [ ] Step 9: Circuit breaker check before backend
  - [ ] Step 10: Backend call → write-through all tiers
- [ ] Add `golang.org/x/sync/singleflight` — deduplicate concurrent in-flight requests to backend
- [ ] `internal/cache/invalidation.go` — Redis pub/sub listener:
  - [ ] Subscribe to invalidation channel
  - [ ] On message: evict from L1 + delete from L2a
  - [ ] Propagate invalidation across all tiers
- [ ] `internal/cache/warmup.go`:
  - [ ] On startup: query Postgres for top 5000 entries by `access_count DESC`
  - [ ] Load into L1 cache
  - [ ] Complete within 30 seconds

#### Integration Tests — Sprint 4
- [ ] `tests/integration/cache_flow_test.go`:
  - [ ] Full cache miss → backend → write-through → cache hit flow
  - [ ] Concurrent identical queries (< 50ms apart) produce exactly ONE backend call
  - [ ] L1 hit returns before L2a/L2b are checked
  - [ ] L2a hit backfills L1
  - [ ] L2b hit backfills L1 + L2a
- [ ] `tests/integration/policy_test.go`:
  - [ ] Temporal mismatch (`"today"` vs `"tomorrow"`) never serves cached answer
  - [ ] Confidence below threshold → cache miss, goes to backend
  - [ ] Confidence at/above threshold → cache hit returned

#### Sprint 4 Gate ✅
- [ ] `docker-compose up` → run integration tests against live containers — all pass
- [ ] Singleflight verified: concurrent requests benchmark shows 1 backend call per unique query
- [ ] Warmup loads 5000 entries within 30 seconds on a populated test database

---

## Phase 3 — HTTP API & Security
**Weeks 5–7 · 3 sprints · Public surface — must be correct before production**

> ⚠️ Do not begin Phase 3 until Phase 2 integration tests are fully green.

### Sprint 5 — Core API Endpoints

- [ ] `cmd/server/main.go`:
  - [ ] Wire all dependencies (DB, Redis, Ollama client, coordinator, metrics)
  - [ ] Graceful shutdown on `SIGTERM`
  - [ ] Apply DB migrations on startup
  - [ ] Start cache warmup goroutine
- [ ] `internal/api/server.go`:
  - [ ] HTTP server setup
  - [ ] Middleware chain: `auth → rate-limit → request-ID → handler`
  - [ ] Route registration for all endpoints
- [ ] `internal/api/handlers.go`:
  - [ ] `POST /cache/query` — buffered response (non-streaming)
  - [ ] `GET /health` — shallow (process alive)
  - [ ] `GET /readyz` — deep check: Redis + Postgres + Ollama + Backend
  - [ ] `GET /livez` — process alive only
  - [ ] `GET /metrics` — Prometheus exposition format
  - [ ] `GET /analytics/cost-savings` — net savings (gross minus infra costs)
- [ ] `internal/api/models.go`:
  - [ ] `QueryRequest` struct
  - [ ] `CacheResponse` struct (hit: source, confidence, latency, similarity, age)
  - [ ] `MissResponse` struct (miss: source=backend, latency, cost_usd)

#### Sprint 5 Gate ✅
- [ ] `curl -X POST /cache/query` returns correct response shape
- [ ] `GET /readyz` returns 200 when all deps are up, 503 within 5s of postgres being stopped
- [ ] `GET /metrics` returns Prometheus text format

---

### Sprint 6 — Authentication & Tenant Isolation

- [ ] `internal/api/middleware.go`:
  - [ ] JWT validation (Bearer token)
  - [ ] Extract `tenant_id` from JWT claims
  - [ ] Return HTTP 401 on missing/invalid token
  - [ ] Per-client rate limiting (token bucket, configurable via env `RATE_LIMIT_RPM`)
- [ ] Thread `tenant_id` through all cache operations:
  - [ ] L1 key prefix: `tenantID + ":"`
  - [ ] L2a key prefix: `"norm:" + tenantID + ":"`
  - [ ] L2b SQL `WHERE tenant_id = $1`
- [ ] `migrations/002_add_tenant.sql`:
  - [ ] Add `tenant_id` column if not exists
  - [ ] Backfill existing rows with `'default'`
  - [ ] Add scoped indexes
  - [ ] ⚠️ Note: in dev, safest to drop+recreate DB before running this migration
- [ ] `POST /admin/invalidate` — pattern-based invalidation across all tiers (requires admin JWT)
- [ ] `POST /admin/reload-policies` — hot-reload `policies.yaml` from disk without restart

#### Sprint 6 Gate ✅
- [ ] Unauthenticated request → HTTP 401
- [ ] Tenant A query cannot retrieve Tenant B cached answers (cross-tenant isolation test)
- [ ] Rate limit triggers correctly at configured threshold
- [ ] `POST /admin/reload-policies` reloads config without process restart

---

### Sprint 7 — Input Validation, Audit & Streaming

- [ ] `internal/api/handlers.go` additions:
  - [ ] `ValidateQueryRequest()` — reject empty query, oversized payload, invalid domain
  - [ ] `ShouldCache()` — determine if response is cacheable (e.g. error responses are not)
  - [ ] `SanitizeAnswer()` — strip any control characters from cached answers
- [ ] `internal/audit/logger.go`:
  - [ ] Structured JSON audit events
  - [ ] `query_hash` = SHA-256 of normalized query (no PII stored)
  - [ ] Fields: `request_id`, `tenant_id`, `query_hash`, `domain`, `decision`, `reason`, `confidence`, `client_ip`, `ts`
  - [ ] Append-only writes (never update audit records)
  - [ ] Decision values: `L1_hit`, `L2a_hit`, `L2b_accept`, `L2b_reject`, `backend`
- [ ] `internal/api/handlers.go` — `StreamQuery()`:
  - [ ] Use `http.Flusher` interface
  - [ ] SSE (Server-Sent Events) format
  - [ ] Handle streaming backend responses
- [ ] `internal/api/feedback.go` — `POST /feedback`:
  - [ ] Accept `{ request_id, correct: bool, reason?: string }`
  - [ ] Store feedback signal in Postgres (used by Phase 5 tuner)

#### End-to-End Tests — Sprint 7
- [ ] Auth failure → HTTP 401
- [ ] Rate limit exceeded → HTTP 429
- [ ] Tenant isolation (cross-tenant query test)
- [ ] Input validation: empty query → HTTP 400
- [ ] Input validation: oversized payload → HTTP 413
- [ ] Invalid domain → graceful fallback to `general` policy

#### Sprint 7 Gate ✅
- [ ] All end-to-end API tests pass
- [ ] Audit log contains structured entries for every request
- [ ] Streaming endpoint returns SSE-formatted chunks

---

## Phase 4 — Observability & Operational Correctness
**Weeks 8–10 · 2 sprints · You cannot operate what you cannot see**

> ⚠️ Do not begin Phase 4 until Phase 3 all sprints are fully green.

### Sprint 8 — Metrics & Dashboards

- [ ] `internal/metrics/prometheus.go` — Define all metrics:
  - [ ] `cache_hits_total` counter — labels: `{tier}` (L1/L2a/L2b)
  - [ ] `cache_misses_total` counter — labels: `{tier}`
  - [ ] `backend_calls_total` counter — labels: `{domain}`
  - [ ] `l2b_confidence_score` histogram — distribution of confidence scores
  - [ ] `policy_rejection_total` counter — labels: `{reason}` (low_confidence/temporal_mismatch/expired)
  - [ ] `cache_latency_seconds` histogram — labels: `{tier}`
  - [ ] `embedding_duration_seconds` histogram
  - [ ] `backend_cost_usd_total` counter
  - [ ] `cache_cost_saved_usd_total` counter
- [ ] Instrument `coordinator.go` with all above metrics
- [ ] Instrument `scoring.go` with confidence histogram + rejection counter
- [ ] Grafana dashboard JSON (`configs/grafana_dashboard.json`):
  - [ ] Panel 1: Cache hit rate by tier (line chart)
  - [ ] Panel 2: Request latency P50/P95/P99 (heatmap)
  - [ ] Panel 3: Backend calls per minute (stat)
  - [ ] Panel 4: Cost savings (gauge)
  - [ ] Panel 5: Confidence score distribution (histogram)
  - [ ] Panel 6: Policy rejection reasons (pie chart)
  - [ ] Panel 7: False positive rate (line chart)
  - [ ] Panel 8: SLO burn rate (graph)

#### Sprint 8 Gate ✅
- [ ] All 8 Grafana panels render with real data from a running system
- [ ] Prometheus scrapes `/metrics` successfully

---

### Sprint 9 — Embedding Versioning, Maintenance & SLOs

- [ ] Startup embedding version check:
  - [ ] Query Postgres for distinct `(embed_model, embed_version)` values in `cache_entries`
  - [ ] If any mismatch with current Ollama model → log error and refuse to start
- [ ] `scripts/maintenance.sql`:
  - [ ] `DELETE FROM cache_entries WHERE NOW() > created_at + (ttl_seconds * INTERVAL '1 second')`
  - [ ] `REINDEX CONCURRENTLY cache_entries_embedding_idx`
  - [ ] `VACUUM ANALYZE cache_entries`
- [ ] Maintenance cron job (pg_cron or external scheduler in docker-compose) — run weekly
- [ ] `configs/alerts.yaml` — Prometheus alerting rules for all 6 SLOs:
  - [ ] SLO 1: P99 cache hit latency < 20ms
  - [ ] SLO 2: P99 backend latency < 700ms
  - [ ] SLO 3: Cache hit rate > 70%
  - [ ] SLO 4: Error rate < 0.1%
  - [ ] SLO 5: False positive rate < 2%
  - [ ] SLO 6: (SLO burn rate alert)
- [ ] Fix benchmark: single canonical run config, reconcile P99 (benchmarks=12ms vs summary=8ms — pick one)
- [ ] Fix cost savings calculation: deduct Ollama + Redis + Postgres infra costs from gross savings
- [ ] Load test — **1000 QPS sustained × 10 minutes**:
  - [ ] P99 cache-hit latency < 20ms ✅
  - [ ] Cache hit rate > 70% ✅
  - [ ] Error rate < 0.1% ✅

#### Sprint 9 Gate ✅
- [ ] All 6 SLO alerting rules defined and fire correctly on synthetic failures
- [ ] `/readyz` returns 503 within 5 seconds of postgres being stopped
- [ ] HNSW maintenance SQL runs without error on populated database
- [ ] Load test passes all SLO checks

---

## Phase 5 — Hardening & Advanced Features
**Weeks 11–14 · Final sprint · Production deployment + best-in-class capabilities**

> ⚠️ Do not begin Phase 5 until all Phase 4 SLOs are green.

### Sprint 10 — Production Deployment & Advanced Features

#### Kubernetes
- [ ] `k8s/deployment.yaml` — `replicas: 3`, resource requests/limits
- [ ] `k8s/hpa.yaml` — HPA: `minReplicas: 3`, `maxReplicas: 20`
- [ ] `k8s/pdb.yaml` — PodDisruptionBudget
- [ ] `k8s/service.yaml` — Service manifest
- [ ] Update README: label `docker-compose` as dev-only, document `kubectl apply` steps

#### Adaptive Threshold Tuning
- [ ] `internal/policy/tuner.go` — `PolicyTuner`:
  - [ ] 7-day rolling FPR (false positive rate) window
  - [ ] Raise threshold by `+0.005` if FPR > target
  - [ ] Lower threshold by `-0.01` if FPR well below target
  - [ ] Wire to `/feedback` signal stream
  - [ ] Run nightly via `time.Ticker` goroutine

#### A/B Testing Infrastructure
- [ ] `internal/policy/experiment.go` — Shadow A/B mode:
  - [ ] Evaluate both `control` and `candidate` policy on every request
  - [ ] Log candidate decision to audit log (never use candidate for actual serving)
  - [ ] Metrics tagged with `{experiment_id}` for offline analysis

#### Hardening
- [ ] Security review:
  - [ ] Pen-test auth endpoints (JWT forgery, missing claims)
  - [ ] Cross-tenant isolation test (Tenant A token cannot access Tenant B data)
  - [ ] Verify no hardcoded secrets (`grep -r "secret\|password\|key" --include="*.go"`)
- [ ] Final load test — **1000 QPS × 10 minutes — all 6 SLOs must pass (GO / NO-GO gate)**:
  - [ ] SLO 1: P99 cache hit < 20ms ✅
  - [ ] SLO 2: P99 backend path < 700ms ✅
  - [ ] SLO 3: Hit rate > 70% ✅
  - [ ] SLO 4: Error rate < 0.1% ✅
  - [ ] SLO 5: FPR < 2% ✅

#### Release
- [ ] `go vet ./...` — zero errors
- [ ] `golangci-lint run ./...` — zero errors
- [ ] Unit test coverage ≥ 80% for `internal/policy/` and `internal/cache/normalize.go`
- [ ] Integration tests pass with real Docker containers (testcontainers-go)
- [ ] All environment variables documented in README with example values
- [ ] Tag `v1.0.0`
- [ ] Write `CHANGELOG.md`
- [ ] Update README with canonical benchmark metrics (P99, hit rate, cost savings)

#### Sprint 10 Gate = MVP ✅
- [ ] All 10 functional requirements from Build_plan.md §10.1 pass
- [ ] All 5 SLO requirements from §10.2 pass under load
- [ ] All operational requirements from §10.3 pass
- [ ] All code quality requirements from §10.4 pass

---

## Build Dependency Order (never violate this)

```
synonyms.yaml
    └── normalize.go
            └── temporal.go ──────────────────────────────────┐
policy/types.go                                               │
    └── scoring.go                                            │
cache/entry.go                                                │
    └── l1.go                                                 │
resilience/circuit_breaker.go                                 │
    └── ollama.go (pkg)                                       │
            └── l2b.go (+ schema migrated)                    │
                    └──────────────────────────────┐          │
l2a.go                                            │          │
    └──────────────────────────────────────────────┤          │
                                                   ▼          ▼
                                           coordinator.go ←──────
                                                   │
JWT library configured                             │
    └── middleware.go (auth)                        │
            └── handlers.go ◄─────────────────────┘
                    └── metrics/prometheus.go
                            └── tuner.go (+ feedback endpoint)
```

---

## Notes & Decisions Log

| Date | Note |
|---|---|
| 2026-03-18 | Project scaffolding done. `internal/cache/entry.go` is the only file written so far. Starting Sprint 1 remaining tasks. |
| 2026-03-18 | Docker & infra phase complete. Created: `docker-compose.yml`, `Dockerfile`, `Makefile`, `migrations/001`, `migrations/002`, `db/migrate.go`, `configs/prometheus.yml`, `configs/policies.yaml`, `configs/synonyms.yaml`, Grafana datasource provisioning. Sprint 1 gate pending live verification (`docker-compose up --build`). |

---

*Last updated: 2026-03-18*
