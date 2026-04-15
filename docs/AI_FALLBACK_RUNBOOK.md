# AI Fallback Runbook

Updated: 2026-04-15
Scope: `/api/v1/analyze` AI suggestion pipeline

## 1) Goal
Keep AI suggestions available even when external providers fail, and make fallback behavior observable and predictable.

## 2) Runtime modes
Configured through `AI_MODE`:

- `hybrid_prefer_external`: primary -> secondary -> internal rules (default)
- `hybrid_prefer_internal`: internal rules first, then external chain
- `external_only`: external providers only, with last-resort internal fallback only if no provider is reachable
- `internal_only`: deterministic local rules only

## 3) Fallback policy knobs
- `AI_FALLBACK_ON_STATUS`: comma-separated status codes that trigger fallback to internal rules in `hybrid_prefer_external`
- `AI_FALLBACK_ON_TIMEOUT_MS`: external call timeout in milliseconds; when > 0, timeout also triggers fallback

Recommended production baseline:

- `AI_MODE=hybrid_prefer_external`
- `AI_INTERNAL_RULES_ENABLED=true`
- `AI_FALLBACK_ON_STATUS=429,500,502,503,504`
- `AI_FALLBACK_ON_TIMEOUT_MS=15000`

## 4) Operational signals
### API response
- `ai_source`: provider that produced final suggestions (`openai:...`, `gemini:...`, `internal_rules`)
- `ai_error`: present when an external provider failed and warnings are surfaced

### Frontend UX
In AI tab:
- Source label always visible
- Status pill (`External AI`, `Local rules`, `Fallback active`)
- Warning banner shown when internal suggestions are used after provider failure

### Metrics endpoint
Use admin endpoint:
- `GET /api/v1/admin/metrics/funnel`
- Header: `X-Admin-Secret: <ADMIN_SECRET>`

Watch especially:
- `analysis_fallbacks`
- `analysis_by_ai_source`
- `forwarding_errors`

## 5) Incident playbook
### Case A: primary provider quota exhausted (429)
1. Confirm spikes in `analysis_fallbacks` and source shift to internal/secondary.
2. Verify `AI_FALLBACK_ON_STATUS` includes `429`.
3. If quality impact is acceptable, keep service in hybrid mode.
4. If external quality is critical, rotate key or switch primary model/provider.

### Case B: provider latency or timeout spikes
1. Confirm timeout-related errors in logs.
2. Tune `AI_FALLBACK_ON_TIMEOUT_MS` (typically 10000-20000).
3. Keep internal fallback enabled to preserve availability.

### Case C: both external providers unstable
1. Switch to safe mode: `AI_MODE=internal_only`.
2. Keep collecting metrics and error counts.
3. Restore hybrid mode only after external stability is confirmed.

## 6) Rollback and safe mode
Immediate safe mode:
- `AI_MODE=internal_only`
- `AI_INTERNAL_RULES_ENABLED=true`

This removes dependency on external providers and guarantees deterministic suggestions.

## 7) Verification checklist after changes
1. API response includes `ai_source` on every analyze request.
2. In forced-failure scenario, response still returns suggestions (from `internal_rules`) plus `ai_error`.
3. Frontend AI panel displays fallback warning and source correctly.
4. Admin metrics show expected increments in fallback counters.
5. CI passes (`go test ./...`, frontend `npm test`, `npm run build`).

For progressive deployment across dev/staging/canary/prod, follow [docs/AI_ROLLOUT_CHECKLIST.md](docs/AI_ROLLOUT_CHECKLIST.md).
