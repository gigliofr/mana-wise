# Commander Brackets Rollout Evidence Template

Date:
Owner:
Environment sequence: dev -> staging -> canary -> prod
Release/commit:

## 1) Scope
- Feature set enabled: Commander bracket evaluator + admin config endpoints
- Runtime config source: in-memory shared config (phase 1)
- Admin endpoints enabled:
  - GET /api/v1/admin/commander-brackets
  - PUT /api/v1/admin/commander-brackets

## 2) Configuration snapshot
- enabled:
- decisive_cards count:
- tutor_keywords count:
- extra_turn_keywords count:
- mass_land_denial_keywords count:
- combo_keywords count:
- fast_mana_keywords count:
- bracket1_max_signals:
- bracket2_max_signals:
- bracket3_max_decisive:
- bracket3_max_signals:
- bracket4_max_signals:
- cedh_tutor_threshold:
- cedh_combo_threshold:
- cedh_fast_mana_threshold:
- cedh_decisive_threshold:

## 3) Pre-flight checks
- [ ] go test ./... passed
- [ ] web npm run build passed
- [ ] /analyze returns commander_bracket for commander decks
- [ ] /score returns commander_bracket for commander decks
- [ ] admin commander-brackets GET/PUT reachable with X-Admin-Secret

Evidence links:
- CI run:
- Test output artifact:
- Smoke-check output artifact:

## 4) Dev validation (100%)
Time window:
Scripted checks:
- scripts/verify_commander_brackets_rollout.ps1

Observed values:
- commander_bracket present on all commander analyze calls: yes/no
- bracket range always 1..5: yes/no
- reason non-empty: yes/no
- admin config fetch/update succeeded: yes/no

Notes:

## 5) Staging validation (100%)
Time window:
Traffic profile/replay method:

Observed values:
- bracket distribution:
- invalid bracket responses:
- regressions in non-commander formats:
- average latency delta vs baseline:

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
- bracket endpoint error rate:
- admin config change count:
- support tickets/user reports:

Final status: success/partial/rolled back

## 8) Rollback details (if any)
- Trigger reason:
- Temporary fallback strategy applied:
- Recovery actions:
- Re-enable criteria:

## 9) KPI before/after
| KPI | Baseline | During rollout | 48h post-rollout | Delta |
|-----|----------|----------------|------------------|-------|
| Commander analyze success rate | | | | |
| commander_bracket presence rate | | | | |
| Invalid bracket responses | | | | |
| Mean analyze latency (ms) | | | | |

## 10) Decision log
- Rule/config updates applied:
- Threshold changes:
- Follow-up actions:

## 11) Sign-off
- Engineering:
- Product:
- Date:
