# Commander Brackets Backoffice Rollout Plan

Version: 1.0
Date: 2026-04-18
Status: Ready for phase-1 rollout

## 1) Objective
Enable safe, auditable management of Commander bracket rules from an admin surface, without redeploys for every threshold/list update.

## 2) Current Phase (Implemented)
- Rule-based bracket evaluator in backend.
- Shared runtime config object for bracket rules.
- Admin endpoints with secret guard:
  - GET /api/v1/admin/commander-brackets
  - PUT /api/v1/admin/commander-brackets
- Hidden admin UI section for built-in admin users.
- Frontend analyzer consumes commander_bracket from backend response.

## 3) Known Limitations (Phase 1)
- Config is in-memory only and not persisted across server restarts.
- Access control is secret-header based and UI visibility is email-pattern based.
- No audit trail of config changes.

## 4) Rollout Strategy
### Phase A - Dev validation (100%)
- Run:
  - go test ./...
  - web npm run build
  - scripts/verify_commander_brackets_rollout.ps1 -Token <jwt> -AdminSecret <secret>
- Validate response contract:
  - analyze(commander) always returns commander_bracket with bracket in 1..5.
  - score(commander) returns commander_bracket.
  - admin GET/PUT works with valid secret and fails with invalid secret.

Exit criteria:
- All checks pass; no contract regression.

### Phase B - Staging validation (100%)
- Replay representative commander decklists (low, medium, high power).
- Compare bracket distribution before/after migration.
- Verify no impact on non-commander routes.

Exit criteria:
- No spike in analyze errors.
- Bracket output stable and deterministic for repeated inputs.

### Phase C - Canary production (10-20%)
- Enable for 10% traffic, monitor 2-4 hours.
- Increase to 20% if stable.
- Watch:
  - analyze error rate
  - latency delta
  - support signals/user reports for bracket misclassification

Abort criteria:
- Commander analyze errors > baseline + 2% absolute for 15 minutes.
- Invalid commander_bracket payload observed.

### Phase D - Full rollout (100%)
- Promote after successful canary.
- Monitor for 48h.

Success criteria:
- Contract stability maintained.
- No severe incidents.

## 5) Backoffice Evolution Plan
### Phase 2 - Persistence + versioning
- Store commander bracket config in MongoDB.
- Load persisted config at startup with fallback to defaults.
- Add config version and updated_at metadata.

### Phase 3 - Role-based admin
- Introduce explicit admin role on user model.
- Keep X-Admin-Secret as emergency break-glass path.
- Enforce RBAC on admin endpoints.

### Phase 4 - Audit and approvals
- Add append-only change log for each config update.
- Track actor, timestamp, old/new values, reason.
- Optional 4-eyes approval for production writes.

### Phase 5 - Safe experimentation
- Add dry-run mode for proposed config.
- Evaluate bracket output on sample decks before apply.
- Add import/export for config bundles.

## 6) Operational Checklist
- [ ] Dev validation complete
- [ ] Staging validation complete
- [ ] Canary 10% complete
- [ ] Canary 20% complete
- [ ] Full rollout complete
- [ ] Post-rollout evidence archived

## 7) Evidence Workflow
1. Generate evidence file:
  - scripts/create_ai_rollout_evidence.ps1 -TemplateKind commander-brackets
2. Run smoke checks:
   - scripts/verify_commander_brackets_rollout.ps1 -Token <jwt> -AdminSecret <secret>
3. Attach outputs and sign-off in evidence markdown.

## 8) Rollback Plan
Immediate rollback options:
- Revert to previous release.
- Reset commander bracket config to default safe values.
- Temporarily freeze admin PUT endpoint access if malformed updates are suspected.

Recovery criteria:
- Stable analyze responses.
- Valid commander_bracket payload for commander requests.
- Error rate and latency back to baseline range.
