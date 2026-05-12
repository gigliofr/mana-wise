# Enhanced Stability Roadmap – Phase 2 & Beyond

**Last Updated:** May 12, 2026  
**Status:** Planning (Phase 1 partially complete)

---

## Executive Summary

The ManaWise backend has foundational stability measures in place:
- **RequestID tracking** per request (chi middleware)
- **Cache layers** for analysis & legality (30-min TTL, metrics tracking)
- **Panic recovery** middleware with structured logging
- **AI fallback policy** on LLM provider errors

However, critical gaps remain in **provider resilience, validation rigor, and observability**.  
This roadmap prioritizes **high-impact, low-risk** improvements to prevent cascading failures and improve MTTR.

---

## Phase 1 Status ✅

### ✅ Task 1.1 – Observability Minimum
- [x] Request ID correlation (chi middleware.RequestID)
- [x] Panic logging with req_id
- [x] AI fallback policy startup logs
- [x] Cache metrics (hits/misses in `/admin/metrics/funnel`)

### ✅ Task 1.3 – Cache for Expensive Paths
- [x] Analysis deck cache (30-min TTL, keyed by decklist+format hash)
- [x] Legality evaluator cache (30-min TTL, keyed by cards+quantities hash)
- [x] Cache janitor goroutines (1-min eviction interval, graceful shutdown)

### ⏳ Task 1.2 – Timeouts & Fallbacks (Partial)
- [x] AI fallback policy (LLM → internal rules)
- [x] Scryfall retry with exponential backoff
- [ ] **Missing:** Circuit breaker for MongoDB, uniform DB query timeouts
- [ ] **Missing:** Timeout propagation via context.WithTimeout

### ⏳ Task 1.4 – Contract Tests (Minimal)
- [x] Handler panic recovery tests
- [ ] **Missing:** Snapshot tests for analyze, score, legality endpoints
- [ ] **Missing:** Regression tests for PDF, share, sideboard payloads

---

## Phase 2 – Resilience & Reliability (NEW)

### 2.1 – Circuit Breaker for External Providers

**Problem:** MongoDB/Scryfall outages cascade to all clients; no fast-fail.

**Solution:**
```
1. Create infrastructure/circuitbreaker/circuit_breaker.go
   - States: Closed (normal) → Open (failing fast) → Half-Open (testing)
   - Configurable thresholds: failures=5, timeout=30s
   - Expose state in /admin/metrics/stability

2. Wrap MongoDB repository with circuit breaker
   - On Open: return ErrCircuitOpen (cached or fallback data)
   - On Half-Open: allow 1 request, transition based on result

3. Expose Scryfall circuit state (rate limit tracking already builtin)

4. Add metrics: state transitions, recovery success rate
```

**Definition of Done:**
- [ ] infrastructure/circuitbreaker implemented & tested
- [ ] MongoDB repository wrapped with circuit breaker
- [ ] State exposed in admin metrics
- [ ] Test: simulate provider outage, verify fast-fail behavior

**Estimated Effort:** 1 day

---

### 2.2 – Uniform Database Query Timeouts

**Problem:** Queries can hang indefinitely; no resource exhaustion protection.

**Solution:**
```
1. Wrap MongoDB client with default timeout:
   - Read queries: 10s (most queries should complete in <2s)
   - Write queries: 5s (bulk inserts/updates)
   - Configurable via env: DB_QUERY_TIMEOUT_READ_MS, DB_QUERY_TIMEOUT_WRITE_MS

2. Audit critical query paths:
   - CardRepository.FindByNames
   - DeckRepository.FindByID, Update
   - UserRepository.FindByEmail
   - SharedAnalysisLinkRepository queries
   - Add context.WithTimeout(timeout) to each

3. Add test: simulate slow query, verify timeout error

4. Telemetry: log query name, duration, timeout status
```

**Definition of Done:**
- [ ] MongoDB wrapper with timeout default implemented
- [ ] All critical queries wrapped
- [ ] Test timeout behavior
- [ ] Timeout values documented

**Estimated Effort:** 1 day

---

### 2.3 – Structured Logging with Log Levels

**Problem:** All logs go to stdout; no filtering; hard to troubleshoot.

**Solution:**
```
1. Integrate Go 1.21+ slog (or zap for compatibility):
   - Levels: DEBUG, INFO, WARN, ERROR, FATAL
   - Structured fields: req_id, endpoint, duration_ms, error, etc.

2. Config:
   - LOG_LEVEL env var (default: INFO)
   - Output format: JSON in prod, human in dev

3. Key log points:
   - ERROR: DB query fail, LLM fail, cache eviction, circuit breaker state change
   - INFO: Request start/end, cache hit/miss rate, config loaded
   - DEBUG: SQL queries, provider retries, timeout events

4. Backward compat: redirect old log.Printf to slog
```

**Definition of Done:**
- [ ] slog/zap integrated
- [ ] Core log points migrated
- [ ] LOG_LEVEL configurable
- [ ] Documentation: what to log and at which level

**Estimated Effort:** 0.5 day

---

### 2.4 – Rate Limiting for Expensive Endpoints

**Problem:** POST /analyze not rate-limited; possible DOS/abuse.

**Solution:**
```
1. Create middleware/rate_limiter.go:
   - Implement token bucket per (user_id, ip)
   - Differentiated by plan: free=5 req/min, pro=50 req/min

2. Apply to:
   - POST /analyze (expensive card resolution)
   - POST /score (deterministic analysis)

3. Response headers:
   - X-RateLimit-Limit: 5
   - X-RateLimit-Remaining: 3
   - X-RateLimit-Reset: 1715520000 (unix timestamp)

4. Return 429 on limit exceeded with Retry-After header

5. Telemetry: track 429 responses, alert if high rate
```

**Definition of Done:**
- [ ] Rate limiter middleware implemented
- [ ] Applied to /analyze and /score
- [ ] Response headers correct
- [ ] Test: verify limits enforced, plan differentiation works

**Estimated Effort:** 0.5 day

---

### 2.5 – Validation Layer

**Problem:** Input validation scattered in handlers; easy to miss edge cases.

**Solution:**
```
1. Create domain/validation/validator.go:
   - Composable validators: isNonEmpty, isValidFormat, isValidPlan, etc.
   - Return struct: field, code, message

2. Apply to request DTOs:
   - AnalyzeRequest: format must be known, decklist non-empty
   - DeckUpdateRequest: format valid, name non-empty
   - ShareAnalysisRequest: deck_id matches auth user

3. Validator runs in handler before usecase.Execute()

4. Error response:
   {
     "error": "validation failed",
     "code": "ERR_VALIDATION",
     "issues": [
       { "field": "format", "code": "ERR_UNKNOWN_FORMAT", "message": "format 'foo' is not supported" }
     ]
   }
```

**Definition of Done:**
- [ ] Validator framework implemented
- [ ] Applied to all request DTOs
- [ ] Test: validation errors return correct response
- [ ] Documentation: how to add new validator

**Estimated Effort:** 1 day

---

### 2.6 – Standard Error Response Format

**Problem:** API errors inconsistent (some have `error`, `code`, `details`; others don't).

**Solution:**
```
1. Standardize all error responses:
   {
     "error": "human-readable message",
     "code": "ERR_CODE",
     "details": { "field": "value", ... }  // optional
   }

2. Error codes enum:
   - ERR_INVALID_FORMAT
   - ERR_CARD_NOT_FOUND
   - ERR_DECK_NOT_FOUND
   - ERR_UNAUTHORIZED
   - ERR_RATE_LIMITED
   - ERR_INTERNAL_ERROR
   - ERR_TIMEOUT
   - etc.

3. Migrate all jsonError() calls in handlers

4. Snapshot test for common error scenarios
```

**Definition of Done:**
- [ ] Error response struct standardized
- [ ] All handlers migrated
- [ ] Test: snapshot tests for common errors
- [ ] Documentation: error code reference

**Estimated Effort:** 0.5 day

---

### 2.7 – Transparent Retry Strategy

**Problem:** Retry logic varies per provider (Scryfall builtin, LLM policy-based, MongoDB none).

**Solution:**
```
1. Create infrastructure/retry/retry.go:
   - BackoffFunc: exponential with jitter
   - MaxRetries: configurable per provider
   - IsRetryable(error): predicate for transient errors

2. Standardize per provider:
   - Scryfall: retry on 429, 5xx (already done, standardize)
   - MongoDB: retry on transient errors (duplicate key, timeout) -- NEW
   - LLM: explicit fallback policy (already done)

3. Telemetry: track retry_count, delay_ms per call

4. Documentation: explain when/why we retry
```

**Definition of Done:**
- [ ] Retry framework implemented
- [ ] Applied to MongoDB wrapper
- [ ] Telemetry added
- [ ] Test: verify retry behavior under simulated failure

**Estimated Effort:** 1 day

---

### 2.8 – Context Deadline Propagation

**Problem:** Long-running operations (analyze, PDF render, share) might not respect client timeout.

**Solution:**
```
1. Modify handler root middleware:
   - Extract client deadline from r.Context().Deadline()
   - Or set server deadline: ctx, cancel = context.WithTimeout(r.Context(), 30s)

2. Pass ctx to all usecase methods:
   - Verify: AnalyzeDeckUseCase.Execute(ctx) already done ✓
   - Check: ScoreUseCase.Calculate(ctx)
   - Check: PublicShareHandler.ServeHTTP context passing

3. Add test:
   - Start long operation
   - Cancel context mid-operation
   - Verify operation stops gracefully

4. Telemetry: track context.DeadlineExceeded errors
```

**Definition of Done:**
- [ ] Context.WithTimeout applied at handler entry
- [ ] All usecase methods receive context
- [ ] Test: operation timeout behavior
- [ ] Documentation: timeout values per endpoint

**Estimated Effort:** 0.5 day

---

## Phase 3 – Observability Deep Dive

### 3.1 – Stability Metrics Endpoint

**New Endpoint:** `GET /admin/metrics/stability` (secret-key protected)

**Response:**
```json
{
  "timestamp_unix_ms": 1715520000000,
  "errors_24h": {
    "/analyze": 12,
    "/score": 5,
    "/decks/{id}": 3
  },
  "cache_stats": {
    "hits": 2450,
    "misses": 340,
    "hit_ratio": 0.878,
    "evictions_24h": 45
  },
  "circuit_breakers": {
    "mongodb": { "state": "closed", "failures": 0, "last_transition": 1715400000 },
    "scryfall": { "state": "closed", "failures": 2, "last_transition": 1715300000 }
  },
  "latency_percentiles_ms": {
    "p50": { "/analyze": 150, "/score": 200 },
    "p95": { "/analyze": 450, "/score": 650 },
    "p99": { "/analyze": 1200, "/score": 1800 }
  },
  "database": {
    "query_latency_p99_ms": 85,
    "connection_pool_used": 8,
    "connection_pool_max": 20
  },
  "rate_limit_violations_24h": 3,
  "request_timeout_rate": 0.0012,
  "ai_fallbacks_24h": 45,
  "ai_fallback_rate": 0.023
}
```

**Definition of Done:**
- [ ] Metrics aggregated and exposed
- [ ] Endpoint protected by admin secret
- [ ] Test snapshot
- [ ] Documentation: what each metric means

**Estimated Effort:** 0.5 day

---

## Phase 4 – Database Optimization

### 4.1 – Index Audit & Tuning

**Problem:** MongoDB queries might be slow due to missing indexes.

**Solution:**
```
1. Identify critical query paths:
   - CardRepository.FindByNames (batch lookup)
   - DeckRepository.FindByID (user_id + id)
   - UserRepository.FindByEmail
   - SharedAnalysisLinkRepository.FindByToken

2. Create indexes:
   db.cards.createIndex({ name: 1 })
   db.decks.createIndex({ user_id: 1, _id: 1 })
   db.users.createIndex({ email: 1 }, { unique: true })
   db.shared_analysis_links.createIndex({ token: 1 }, { unique: true })
   db.shared_analysis_links.createIndex({ user_id: 1 })

3. Benchmark query latency pre/post index

4. Document migration script (index creation in test setup or deployment)
```

**Definition of Done:**
- [ ] Indexes identified and created
- [ ] Benchmark (explain plan, latency)
- [ ] Migration script created
- [ ] Test: verify indexes used (EXPLAIN output)

**Estimated Effort:** 1 day

---

### 4.2 – Connection Pool Tuning

**Problem:** MongoDB connection pool might be undersized or oversized.

**Solution:**
```
1. Verify current pool config in connection URI
   - minPoolSize, maxPoolSize, maxIdleTimeMS

2. Recommended settings:
   - minPoolSize: 5
   - maxPoolSize: 50 (adjust based on concurrency)
   - maxIdleTimeMS: 60000

3. Add env config:
   - MONGODB_MIN_POOL_SIZE
   - MONGODB_MAX_POOL_SIZE

4. Telemetry: monitor pool usage (used connections, wait time)
```

**Definition of Done:**
- [ ] Config parameters added
- [ ] Documentation: how to tune pool
- [ ] Monitoring added to metrics endpoint
- [ ] Load test to verify pool sizing

**Estimated Effort:** 0.5 day

---

## Execution Timeline

### Week 1 (Sprint 1) – High Priority
- [x] Circuit Breaker framework (1 day)
- [x] Structured Logging (0.5 day)
- [ ] Rate Limiting middleware (0.5 day)
- [ ] Validation framework (1 day)

### Week 2 (Sprint 2) – Medium Priority
- [ ] Timeout Uniformity (1 day)
- [ ] Error Response Standardization (0.5 day)
- [ ] Retry Strategy (1 day)
- [ ] Context Deadline Propagation (0.5 day)

### Week 3 (Sprint 3) – Observability
- [ ] Stability Metrics endpoint (0.5 day)
- [ ] Database Index Audit (1 day)
- [ ] Connection Pool Tuning (0.5 day)

### Week 4+ – Feature Development
- Resume Phase 2–4 feature work (upgrade assistant, deck diff, meta alerts, etc.)

---

## Definition of Done (Per Task)

- [ ] Code compiles and all tests pass
- [ ] New test coverage added (unit or integration)
- [ ] Documentation updated (README, inline comments)
- [ ] Telemetry/observability added (logs, metrics, monitoring points)
- [ ] Backward compatibility preserved (or migration path clear)
- [ ] Performance impact analyzed (if applicable)
- [ ] Code review approved and merged

---

## Success Criteria

### Quantitative
- Error rate < 0.1% for non-transient errors
- Cache hit ratio > 80% for analysis/legality
- P99 latency for /analyze < 2s (non-cached)
- Circuit breaker prevents cascading failure (< 1s MTTR after provider recovery)
- Zero unhandled panics in production logs

### Qualitative
- On-call engineer can diagnose 95% of issues from logs within 5 minutes
- Error messages clearly indicate root cause (validation, provider unavailable, timeout, etc.)
- Fallback behavior transparent to user (clear why they got internal AI instead of primary LLM)

---

## Risk Mitigation

| Risk | Likelihood | Impact | Mitigation |
|------|------------|--------|-----------|
| Circuit breaker too aggressive (opens on transient spike) | Medium | Medium | Half-Open state with configurable thresholds; test with simulated spikes |
| Logging overhead reduces throughput | Low | Low | Async logging, sample verbose logs in DEBUG |
| Rate limit false positives | Medium | Low | Grace period (1-2 requests), monitor 429 rate, adjust limits per plan |
| Timeout too tight (kills valid requests) | High | High | Monitor P99 latency before deployment, add grace buffer (P99 + 500ms) |
| Database indexes bloat | Low | Low | Benchmark index creation, monitor disk usage |

---

## Next Steps

1. **Triage this roadmap** with team (prioritize tasks if different from suggested order)
2. **Create tasks** in your issue tracker (Jira, Linear, GitHub Issues) per section
3. **Assign owners** for parallel execution if possible
4. **Set weekly reviews** to track progress and blockers
5. **Integrate metrics** into dashboards/alerts once implemented

---

**Questions?** Refer to `/memories/repo/stability-analysis-plan.md` for analysis details.
