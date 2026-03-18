# Semantic Cache Proxy

A Redis-like caching proxy for **probabilistic queries** that safely reuses responses using
**time-aware, domain-aware, and observable decisions**.

---

## Project Status

🚧 **Active Development**

Current Phase: **Phase 1 — Core Server**

---

## What This Is

This service sits between clients and expensive backends (LLMs, APIs, DBs) and decides:

> “Is it safe to reuse a previous answer, or should we recompute?”

Correctness always wins over optimization.

---

## High-Level Flow
### Request
- L1 (in-memory exact cache)
- L2a (Redis normalized cache)
- L2b (semantic cache + policy)
- Backend
- Write-through + metrics

## Tech Stack

- Language: Go
- HTTP: net/http
- L1: In-process LRU
- L2a: Redis
- L2b: Ollama + Postgres (pgvector)
- Metrics: Prometheus
- Config: YAML
