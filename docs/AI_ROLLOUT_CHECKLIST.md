# AI Rollout Checklist

Updated: 2026-04-16
Scope: Gradual rollout for AI suggestions pipeline (`/api/v1/analyze`)

## 1) Pre-flight gate (all environments)
1. Ensure backend and frontend are on a tagged commit with green CI.
2. Verify required environment variables are set.
3. Confirm admin metrics endpoint is reachable with `X-Admin-Secret`.
4. Run smoke tests:
- `go test ./...`
- `cd web && npm test`
- `cd web && npm run build`

## 2) Environment baseline
Recommended baseline before progressive rollout:

```text
AI_MODE=hybrid_prefer_external
AI_INTERNAL_RULES_ENABLED=true
AI_FALLBACK_ON_STATUS=429,500,502,503,504
AI_FALLBACK_ON_TIMEOUT_MS=15000
```

Optional secondary provider:

```text
LLM_SECONDARY_PROVIDER=openai
LLM_SECONDARY_API_KEY=...
LLM_SECONDARY_BASE_URL=...
LLM_SECONDARY_MODEL=gpt-4o-mini
```

## 3) Dev rollout (100%)
1. Deploy with baseline config.
2. Execute 10-20 manual `/api/v1/analyze` calls across at least 2 formats.
3. Validate responses include `ai_source` for every call.
4. Simulate provider failure and validate fallback:
- response still contains suggestions
- `ai_source=internal_rules`
- `ai_error` populated
5. Validate frontend AI tab shows source and fallback warning state.

Exit criteria:
- No 5xx increase on analyze endpoint.
- Fallback path works deterministically.

## 4) Staging rollout (100%)
1. Replay representative traffic (or scripted deck set).
2. Collect metrics for at least 30-60 minutes:
- `analysis_fallbacks`
- `analysis_by_ai_source`
- `forwarding_errors`
3. Compare with baseline latency and error rate.
4. Verify no UI regressions on desktop/mobile.

Exit criteria:
- Analyze availability >= 99% during test window.
- Mean latency regression <= +20% versus baseline.
- No unhandled frontend errors.

## 5) Production canary (10-20%)
1. Enable rollout on 10% traffic first.
2. Monitor for 30 minutes minimum.
3. If stable, increase to 20% and monitor another 30-60 minutes.
4. Keep incident channel open and owner on-call.

Abort criteria:
- Analyze 5xx spikes above baseline +2%
- Sustained fallback spike with degraded suggestion quality
- Critical user-facing errors in AI panel

## 6) Full production rollout (100%)
1. Promote to 100% traffic.
2. Monitor for first 48 hours:
- fallback rate trend
- provider source distribution
- analyze latency and error rate
3. Publish brief post-rollout report with KPI deltas.

## 7) Safe rollback procedure
Immediate safe mode:

```text
AI_MODE=internal_only
AI_INTERNAL_RULES_ENABLED=true
```

Then:
1. Confirm `ai_source=internal_rules` for fresh analyze requests.
2. Continue monitoring error budget and latency.
3. Re-enable hybrid mode only after root cause is mitigated.

## 8) Evidence bundle for completion
1. CI links and test outputs.
2. Metrics snapshots before/after rollout.
3. Sample API responses showing `ai_source` and fallback case.
4. Decision log (provider/model changes, threshold adjustments).
