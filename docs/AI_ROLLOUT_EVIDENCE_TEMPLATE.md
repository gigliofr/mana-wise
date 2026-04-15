# AI Rollout Evidence Template

Date:
Owner:
Environment sequence: dev -> staging -> canary -> prod
Release/commit:

## 1) Scope
- Feature set enabled:
- AI mode at rollout start:
- Primary provider/model:
- Secondary provider/model:

## 2) Configuration snapshot
- AI_MODE:
- AI_INTERNAL_RULES_ENABLED:
- AI_FALLBACK_ON_STATUS:
- AI_FALLBACK_ON_TIMEOUT_MS:
- LLM_PROVIDER:
- LLM_MODEL:
- LLM_SECONDARY_PROVIDER:
- LLM_SECONDARY_MODEL:

## 3) Pre-flight checks
- [ ] CI green
- [ ] go test ./... passed
- [ ] web npm test passed
- [ ] web npm run build passed
- [ ] admin metrics endpoint reachable

Evidence links:
- CI run:
- Test output artifact:

## 4) Dev validation (100%)
Time window:
Manual/scripted analyze calls:

Observed values:
- ai_source present on all calls: yes/no
- fallback case validated (internal_rules + ai_error): yes/no
- average latency (ms):
- hard failures:

Notes:

## 5) Staging validation (100%)
Time window:
Traffic profile/replay method:

Metrics snapshot:
- analysis_fallbacks:
- analysis_by_ai_source:
- forwarding_errors:
- analyze error rate:
- mean latency change vs baseline:

Exit criteria met: yes/no
Notes:

## 6) Canary rollout (10-20%)
10% window:
- start/end:
- issues:
- rollback required: yes/no

20% window:
- start/end:
- issues:
- rollback required: yes/no

Abort criteria triggered: yes/no
Details:

## 7) Full rollout (100%)
Start time:
48h monitoring summary:
- fallback trend:
- source distribution trend:
- error rate trend:
- latency trend:

Final status: success/partial/rolled back

## 8) Rollback details (if any)
- Trigger reason:
- Safe mode applied (AI_MODE=internal_only): yes/no
- Recovery actions:
- Re-enable criteria:

## 9) KPI before/after
| KPI | Baseline | During rollout | 48h post-rollout | Delta |
|-----|----------|----------------|------------------|-------|
| Analyze availability | | | | |
| Analyze mean latency (ms) | | | | |
| Fallback rate | | | | |
| AI user-facing errors | | | | |

## 10) Decision log
- Provider/model changes:
- Timeout/policy changes:
- Follow-up actions:

## 11) Sign-off
- Engineering:
- Product:
- Date:
