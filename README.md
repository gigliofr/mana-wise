| POST   | `/api/v1/decks/import` | JWT | Import decklist from Arena/MTGO/Moxfield/text format with card resolution |
| GET    | `/api/v1/decks/{id}/export?format=arena|mtgo|moxfield|text` | JWT | Export deck to specified format |
# ManaWise AI

AI-powered Magic: The Gathering deck analyzer.  
Go backend · MongoDB · OpenAI · React frontend · Railway-ready.

## Quick start

```bash
# 1. Copy env
cp .env.example .env
# Edit .env: fill MONGODB_URI and JWT_SECRET at minimum.

# 2. Run backend
go run ./cmd/server

# 3. Run frontend (dev)
cd web && npm install && npm run dev
```

API available at `http://localhost:8080/api/v1`.  
Frontend at `http://localhost:5173` (proxies API).

## Architecture

```
domain/           → Card, Deck, User, AnalysisResult + Repository interfaces
usecase/          → AnalyzeDeck, ManaCurve, Interaction, WorkerPool
infrastructure/   → MongoDB repositories, Scryfall client, LLM connector, Cache
api/              → Chi router, JWT middleware, Freemium gate, HTTP handlers
cmd/server/       → Entry point, dependency injection
web/              → Vite + React SPA
```

## Key endpoints

| Method | Path | Auth | Description |
|--------|------|------|-------------|
| GET    | `/api/v1/health` | — | Health check |
| GET    | `/api/v1/meta/{format}` | — | Meta snapshot with archetype distribution and trend data (formats: modern, legacy, pioneer, standard) |
| POST   | `/api/v1/auth/register` | — | Register |
| POST   | `/api/v1/auth/login` | — | Login |
| POST   | `/api/v1/auth/forgot-password` | — | Request password reset email (always returns accepted) |
| POST   | `/api/v1/auth/reset-password` | — | Reset password using one-time token |
| GET    | `/api/v1/auth/me` | JWT | Current user |
| POST   | `/api/v1/analyze` | JWT + Freemium | Analyze decklist |
| POST   | `/api/v1/matchup/simulate` | JWT | Simulate matchup matrix (play/draw aware, meta-weighted, weakness diagnosis) |
| POST   | `/api/v1/sideboard/plan` | JWT | Build matchup-specific sideboard ins/outs |
| POST   | `/api/v1/mulligan/simulate` | JWT | Monte Carlo keep-rate simulation by hand size |
| POST   | `/api/v1/deck/classify` | JWT | Classify deck fingerprint (archetype, color identity, curve, confidence) |
| GET    | `/api/v1/decks/{id}/summary` | JWT | Aggregated deck snapshot (cards count, legality map, estimated prices, missing prices) |
| GET    | `/api/v1/decks/{id}/price` | JWT | Deck-centric price aggregation (USD/EUR totals + per-card lines, main/sideboard split) |
| GET    | `/api/v1/decks/{id}/budget?target=200` | JWT | Budget optimizer with replacement suggestions to reach target USD |
| GET    | `/api/v1/decks/{id}/analysis` | JWT | Deterministic analysis + optional fingerprint for a saved deck |
| POST   | `/api/v1/decks/{id}/sideboard/suggest` | JWT | Deck-centric sideboard suggestions with complete 15-card meta-oriented generation (works with or without saved sideboard) |
| POST   | `/api/v1/decks/{id}/simulate` | JWT | Deck-centric mulligan simulation (keep probability, P(2 lands T2), P(1-drop), curve-out T1-T4, structured keep/mulligan reasoning) |
| GET    | `/api/v1/decks/{id}/synergies` | JWT | Deck-centric combo and synergy package detection (hybrid rule+embedding ranking, includes ranking metadata) |
| GET    | `/api/v1/decks/{id}/history` | JWT | Deck version timeline with per-version card diff and snapshot |
| POST   | `/api/v1/decks/{id}/restore/{version}` | JWT | Restore deck to a previous version and append restore event |
| GET    | `/api/v1/decks/{id}/legality` | JWT | Real-time multi-format legality report for a saved deck |
| GET    | `/api/v1/users/me/collection/gaps/{deck_id}?owned=CardA:2,CardB:1` | JWT | Collection gap analysis versus target deck (missing copies + acquisition USD total) |
| GET    | `/api/v1/users/me/notifications` | JWT | User notification feed for banlist/rotation events with replacement suggestions |
| POST   | `/api/v1/webhooks/scryfall` | — | Ingest external card legality events (banlist/rotation) into notification store |
| GET    | `/api/v1/cards/search?name=...` | JWT | Resolve card by name with fuzzy fallback |
| POST   | `/api/v1/cards/metadata/batch` | JWT | Resolve metadata batch (rarity, set, collector number) by card names |
| GET    | `/api/v1/cards/by-name/price-trend?name=...` | JWT | Price trend by card name |
| GET    | `/api/v1/cards/by-name/synergies?name=...&n=10` | JWT | Synergies by card name |
| GET    | `/api/v1/cards/{id}` | JWT | Card detail |
| GET    | `/api/v1/cards/{id}/price-trend` | JWT | Price trend |
| GET    | `/api/v1/cards/{id}/synergies?n=10` | JWT | Semantic synergies |
| POST   | `/api/v1/embed/batch` | JWT | Generate and store card embeddings |
| POST   | `/api/v1/analytics/upgrade-click` | JWT | Track upgrade CTA click |
| POST   | `/api/v1/ota/release` | JWT | Publish OTA firmware release (checksum-verified) |
| POST   | `/api/v1/ota/report-boot` | JWT | Report boot status; failed triggers rollback |
| GET    | `/api/v1/ota/manifest` | JWT | Current/previous OTA release metadata |
| GET    | `/api/v1/admin/metrics/funnel` | `X-Admin-Secret` | Runtime funnel snapshot (event counters, AI fallback counters, forwarding errors) |

`POST /api/v1/embed/batch` body:

```json
{
	"limit": 200,
	"force": false
}
```

Query options for synergies:

- `n` (max 100)
- `min_score` (optional cosine threshold, e.g. `0.35`)

`GET /api/v1/cards/search?name=lightning%20bolt` resolves a card from local DB first, then falls back to Scryfall fuzzy matching and persists the result.

## Freemium limits

| Plan | Analyses/day |
|------|-------------|
| Free | 3 |
| Pro  | Unlimited |

For local testing you can switch a user plan quickly:

```powershell
./scripts/set_user_plan.ps1 -Email "you@example.com" -Plan pro
```

## AI pipeline

ManaWise uses a multi-tier AI routing system controlled by environment variables.

### Modes (`AI_MODE`)

| Value | Behaviour |
|-------|-----------|
| `hybrid_prefer_external` | Try primary provider → secondary → internal rules (default) |
| `hybrid_prefer_internal` | Try internal rules first → external chain as fallback |
| `external_only` | Only external providers; falls back to internal rules if no provider is reachable |
| `internal_only` | Only deterministic internal rules — no external calls |

### Internal rules engine

When `AI_INTERNAL_RULES_ENABLED=true`, a deterministic rule-based engine generates ranked suggestions from the deck's analysis result (land delta, average CMC, interaction score, per-category breakdowns). Zero external calls; always available.

### Secondary provider

Set `LLM_SECONDARY_PROVIDER` + credentials to chain a second LLM after the primary fails (quota / timeout / 5xx). Same OpenAI-compatible interface.

### Response field `ai_source`

Every `/api/v1/analyze` response includes `"ai_source"` indicating which tier produced the suggestions (e.g. `"gemini"`, `"openai"`, `"internal_rules"`).

## Environment variables

See `.env.example` for all variables.  
Required: `MONGODB_URI`, `JWT_SECRET`.  
Optional: `OPENAI_API_KEY` or `GEMINI_API_KEY` (AI suggestions fall back to internal rules if absent).

### Coolify / single-service deployment

If you deploy ManaWise as a single service in Coolify, set at least:

```text
PORT=8080
ENVIRONMENT=production
LOG_LEVEL=INFO
APP_TIMEZONE=Europe/Rome
MONGODB_URI=...
MONGODB_DB_NAME=manawise
JWT_SECRET=...
JWT_EXPIRY_HOURS=72
MANAWISE_ALLOWED_ORIGINS=https://your-frontend-domain
FRONTEND_RESET_PASSWORD_URL=https://your-frontend-domain
PASSWORD_RESET_TOKEN_TTL_MINUTES=30
SMTP_HOST=...
SMTP_PORT=587
SMTP_USER=...
SMTP_KEY=...
MAIL_FROM=...
```

Email handling uses the same SMTP variables as the previous deployment flow, so reset and transactional mails keep working with the same provider settings.

### Runtime / CORS variables

| Variable | Default | Description |
|----------|---------|-------------|
| `MANAWISE_ALLOWED_ORIGINS` | `*` in development, empty in non-development | Comma-separated allowlist of origins for CORS. In non-development, when empty, cross-origin requests are blocked by default. |
| `SMTP_HOST` | — | SMTP host for transactional auth emails |
| `SMTP_PORT` | — | SMTP port |
| `SMTP_USER` | — | SMTP username |
| `SMTP_KEY` | — | SMTP password/API key |
| `MAIL_FROM` | — | Sender email address |
| `FRONTEND_RESET_PASSWORD_URL` | `/reset-password` fallback | Optional absolute frontend URL used in reset email links |
| `PASSWORD_RESET_TOKEN_TTL_MINUTES` | `30` | Password reset token expiry in minutes |

### Key AI variables

| Variable | Default | Description |
|----------|---------|-------------|
| `AI_MODE` | `hybrid_prefer_external` | Routing mode (see above) |
| `AI_INTERNAL_RULES_ENABLED` | `true` | Enable deterministic fallback engine |
| `LLM_PROVIDER` | `gemini` | Primary LLM provider (`gemini` / `openai_compatible`) |
| `LLM_API_KEY` | — | Primary provider API key |
| `LLM_BASE_URL` | — | Primary provider base URL (optional for Gemini) |
| `LLM_MODEL` | — | Primary model name |
| `LLM_SECONDARY_PROVIDER` | — | Secondary provider (optional) |
| `LLM_SECONDARY_API_KEY` | — | Secondary API key |
| `LLM_SECONDARY_BASE_URL` | — | Secondary base URL |
| `LLM_SECONDARY_MODEL` | — | Secondary model name |

## Tests

```bash
go test ./... -cover
```

## Discord Bot (`cmd/bot`)

Run:

```bash
go run ./cmd/bot
```

Required env vars:

- `DISCORD_BOT_TOKEN`
- `MANAWISE_BOT_JWT` (JWT for a service/user account allowed to call `/api/v1/analyze`)

Optional:

- `MANAWISE_API_URL` (default `http://localhost:8080`)
- `BOT_DEFAULT_FORMAT` (default `commander`)

## Meta Snapshot ETL (optional v2)

`GET /api/v1/meta/{format}` now supports optional external ETL sources with automatic fallback to built-in snapshots.

Optional query parameters:

- `refresh=1` or `force_refresh=true` to bypass cache and force a source refresh.

Optional environment variables:

- `MANAWISE_META_SOURCE_MODERN`
- `MANAWISE_META_SOURCE_LEGACY`
- `MANAWISE_META_SOURCE_PIONEER`
- `MANAWISE_META_SOURCE_STANDARD`
- `MANAWISE_META_CACHE_TTL_SECONDS` (default `900`)

Behavior:

- if a `MANAWISE_META_SOURCE_*` URL is configured and reachable, response uses external payload (`data_source` preserved or set to `external-etl-v1`)
- if external source fails/unavailable, response falls back to deterministic internal snapshot (`data_source=hardcoded-v1-fallback`)
- response includes `cache_status` (`hit`, `miss-external`, `miss-fallback`, `bypass-external`, `bypass-fallback`)

## Analytics (optional)

Set these environment variables to track funnel events:

- `ANALYTICS_PROVIDER=posthog`
- `POSTHOG_API_KEY=...`
- `POSTHOG_HOST=https://app.posthog.com`

Tracked events:

- `analysis_completed`
- `daily_limit_reached`
- `upgrade_clicked`
- `deck_saved`
- `sideboard_suggest_generated`

Runtime metrics endpoint (`GET /api/v1/admin/metrics/funnel`) returns an in-memory snapshot with:

- `total_events`
- `event_counts`
- `analysis_fallbacks`
- `analysis_by_ai_source`
- `forwarding_errors`
- `last_event_at_unix_ms`

Use header `X-Admin-Secret` with the value of `ADMIN_SECRET`.

## OTA (secure update flow)

Publish endpoint payload example:

```json
{
	"version": "1.2.3",
	"platform": "esp32",
	"binary_base64": "...",
	"sha256": "hexchecksum"
}
```

Boot report payload example:

```json
{
	"device_id": "kiosk-001",
	"version": "1.2.3",
	"status": "failed"
}
```

If status is `failed`, the backend rolls back manifest pointers to the previous release.

Command:

```text
!analizza modern
4 Lightning Bolt
4 Goblin Guide
...

!prezzo Black Lotus
!sinergie Rhystic Study
```

## Deploy to Railway

1. Add env vars from `.env.example` in Railway Dashboard > Variables.
2. Push to `main` — Railway auto-builds via `Dockerfile`.
