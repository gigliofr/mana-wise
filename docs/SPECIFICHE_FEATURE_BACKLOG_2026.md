# Specifiche Tecniche Feature Backlog 2026

Versione: 1.0  
Data: 2026-03-31  
Ambito: ManaWise API + Web

## Obiettivo
Tradurre il backlog funzionale in specifiche implementabili nel codice attuale, mantenendo coerenza con architettura esistente:
- base path API: `/api/v1`
- router: Chi (`api/router.go`)
- backend: Go + MongoDB
- frontend: React/Vite

## Scala priorita
- Alta: impatta core value e retention
- Media: aumenta conversione e depth of use
- Bassa/UX: migliora experience e stickiness

## Stato corrente sintetico
- Parzialmente presenti: 1, 2, 3, 5, 11
- Mancanti da zero o incompleti: 4, 6, 7, 8, 9, 10, 12, 13

---

## 1) Mana Curve Analyzer Avanzato
Priorita: Alta  
Stato: Parziale (gia presenti curve, archetype, confidence; manca breakdown completo per tipo dentro bucket CMC)

### Descrizione funzionale
Visualizzazione interattiva curva mana con breakdown per tipo carta per ciascun CMC:
- creature
- instant
- sorcery
- enchantment
- artifact
- planeswalker

Include classificazione archetipo e score aderenza meta.

### Contratto API
`GET /api/v1/decks/{id}/analysis`

Risposta 200:
```json
{
  "deck_id": "uuid",
  "curve": [
    {
      "cmc": 0,
      "creatures": 0,
      "instants": 2,
      "sorceries": 1,
      "enchantments": 0,
      "artifacts": 1,
      "planeswalkers": 0,
      "total": 4
    }
  ],
  "archetype": "aggro",
  "confidence": 0.87,
  "avg_cmc": 1.8,
  "deviation_from_meta": 0.12,
  "meta_fit_score": 88
}
```

Errori:
- 401 unauthorized
- 404 deck not found
- 422 deck not analyzable

### Dipendenze tecniche
- Scryfall API (normalizzazione tipi)
- Classificatore leggero (gia presente nel progetto, da estendere su metriche meta-fit)

### Note implementative
- Riusare pipeline `AnalyzeDeckUseCase`
- Aggiungere modello `CurveBucketByType` in domain
- Persistenza snapshot analisi opzionale per caching

---

## 2) Format Legality Checker Real-Time
Priorita: Alta  
Stato: Parziale (legality gia presente in analisi; manca endpoint dedicato real-time con badge per carta)

### Descrizione funzionale
Validazione immediata per carta su ban list di:
- standard
- pioneer
- modern
- legacy
- vintage
- commander

### Contratto API
`GET /api/v1/decks/{id}/legality`

Risposta 200:
```json
{
  "formats": {
    "standard": { "legal": true, "banned": [] },
    "modern": { "legal": false, "banned": ["Hogaak, Arisen Necropolis"] },
    "commander": { "legal": true, "banned": [] }
  },
  "checked_at": "2026-03-31T10:30:00Z"
}
```

Sync ban list:
- `POST /api/v1/webhooks/scryfall/banlist` (opzionale)
- fallback cron giornaliero interno

### Dipendenze tecniche
- Scryfall legality/ban metadata
- Scheduler cron

### Note implementative
- Estrarre legality checker in service dedicato
- Caching ban list in Mongo + TTL

---

## 3) Mulligan Simulator & Hand Evaluator
Priorita: Alta  
Stato: Parziale (endpoint `POST /api/v1/mulligan/simulate` esiste; da completare con 10k simulazioni e metriche target)

### Descrizione funzionale
Simulatore London Mulligan su Monte Carlo + ipergeometrica server-side:
- P(2 terre a T2)
- P(1-drop disponibile)
- P(curve-out T1-T4)
- raccomandazione keep/mulligan con reason testuale

### Contratto API
`POST /api/v1/decks/{id}/simulate`

Body:
```json
{
  "simulations": 10000,
  "format": "modern"
}
```

Risposta 200:
```json
{
  "keep_probability": 0.73,
  "avg_lands_t1": 1.2,
  "p_two_lands_t2": 0.69,
  "p_one_drop": 0.58,
  "curve_out_t4": 0.41,
  "recommendation": "keep",
  "reason": "Mano bilanciata: 3 terre, curva 1-2-3"
}
```

### Dipendenze tecniche
- Algoritmo ipergeometrico backend (gia introdotto)
- Monte Carlo batched (con seed controllato per test)

### Note implementative
- Mantenere endpoint legacy `/mulligan/simulate` per backward compatibility
- Introdurre endpoint deck-centric come facciata nuova

---

## 4) Synergy & Combo Detector
Priorita: Alta  
Stato: Parziale (endpoint deck-centric v1 rule-based implementato)

### Descrizione funzionale
Rileva combo note e pacchetti sinergici nel deck, con spiegazione e turn-kill stimato.

### Contratto API
`GET /api/v1/decks/{id}/synergies`

Risposta 200:
```json
{
  "combos": [
    {
      "cards": ["Thassa's Oracle", "Demonic Consultation"],
      "type": "two_card_win",
      "description": "Win condition immediata con libreria vuota",
      "turn_kill": 3
    }
  ],
  "synergy_score": 74,
  "packages": [
    {
      "name": "draw engine",
      "cards": ["The One Ring", "Orcish Bowmasters"],
      "score": 68
    }
  ]
}
```

### Dipendenze tecniche
- Dataset combo (EDHREC/public)
- Similarita semantica embedding gia presenti nel progetto

### Note implementative
- Primo step rule-based con knowledge base locale
- Secondo step ranking con embedding similarity

Stato implementazione corrente:
- endpoint `GET /api/v1/decks/{id}/synergies` disponibile
- output include `combos`, `synergy_score`, `packages`
- detection su combo note + package tags (draw/interaction/early pressure/mana engine)
- ranking ibrido rule+embedding con metadati (`ranking_mode`, `embedding_coverage`, `combo.score`)

---

## 5) Sideboard Builder AI
Priorita: Media  
Stato: Parziale (`POST /api/v1/sideboard/plan` e `POST /api/v1/decks/{id}/sideboard/suggest` presenti; manca generazione 15-card completa orientata meta snapshot)

### Descrizione funzionale
Genera sideboard da 15 carte ottimizzata per formato e meta corrente, con rationale per matchup.

### Contratto API
`POST /api/v1/decks/{id}/sideboard/suggest`

Body:
```json
{
  "format": "modern",
  "meta_snapshot": "2026-Q1"
}
```

Risposta 200:
```json
{
  "suggestions": [
    {
      "card": "Rest in Peace",
      "qty": 2,
      "reason": "Hate graveyard vs Dredge/Living End",
      "matchups": ["dredge", "living_end"]
    }
  ],
  "total_cards": 15
}
```

### Dipendenze tecniche
- Meta dataset (MTGGoldfish/MTGTOP8)
- Modulo matchup simulator gia esistente

Stato implementazione corrente:
- endpoint deck-centric `POST /api/v1/decks/{id}/sideboard/suggest` disponibile
- genera una sideboard completa da 15 carte orientata matchup/meta (`total_cards=15`)
- funziona sia con sideboard salvata (modalita ibrida) sia senza sideboard (fallback meta template)
- produce `suggestions`, `total_cards`, `generation_mode`, `plan`
- supporta `opponent_archetype` e `meta_snapshot` nel payload

---

## 6) Price Tracker & Budget Optimizer
Priorita: Media  
Stato: Parziale (endpoint deck-centric prezzo implementato)

### Descrizione funzionale
Prezzo deck real-time + suggerimenti di replacement per budget target.

### Contratto API
- `GET /api/v1/decks/{id}/price`
- `GET /api/v1/decks/{id}/budget?target=200`

Risposta esempio:
```json
{
  "total_usd": 842.5,
  "total_eur": 780.2,
  "cards": [
    { "name": "Ragavan, Nimble Pilferer", "price_usd": 52.0, "price_eur": 47.0 }
  ]
}
```

### Dipendenze tecniche
- TCGPlayer API
- Cardmarket API
- Cache prezzi con TTL breve

Stato implementazione corrente:
- endpoint `GET /api/v1/decks/{id}/price` disponibile
- endpoint `GET /api/v1/decks/{id}/budget?target=...` disponibile
- output include `total_usd`, `total_eur`, split main/sideboard e line items per carta
- budget response include `replacements`, `estimated_savings_usd`, `estimated_new_total_usd`, `achievable`
- source prezzo per carta: `current_prices` con fallback a `latest_snapshot`

---

## 7) Meta Dashboard per Formato
Priorita: Media  
Stato: Parziale (v1: endpoint `/api/v1/meta/{format}` disponibile con distribuzione archetype staticamente configurata + trend dati; manca ETL MTGTOP8/MTGGoldfish integrato per dati real-time)

### Descrizione funzionale
Endpoint snapshot meta con archetype distribution, trend percentuale (up/down/stable), sideboard samples, e popular cards. Versione v1 con hardcoded meta realistic per Modern/Legacy/Pioneer/Standard. Future v2 integrate live ETL.

### Contratto API
`GET /api/v1/meta/{format}`

Risposta 200:
```json
{
  "format": "modern",
  "archetypes": [
    {
      "name": "Scam",
      "percentage": 18.5,
      "description": "Rakdos tempo with Fury + Counterspell interactive shell",
      "trend_direction": "stable",
      "trend_percentage": 0.2,
      "sideboard_sample": ["Temporary Lockdown", "Zealous Persecution"],
      "popular_cards": ["Fury", "Murktide", "Dress Down"]
    }
  ],
  "last_updated_at": "2026-03-31T16:45:00Z",
  "data_source": "hardcoded-v1-placeholder",
  "sample_size": 1000
}
```

### Dipendenze tecniche
- Handler: `api/handlers/meta.go`
- Route: `GET /api/v1/meta/{format}` (public, no JWT)
- Supportati: modern, legacy, pioneer, standard
- V1: Hardcoded realistic meta distribution (Modern: Scam 18.5%, Rhinos 16.2%, Murktide 14.8%, etc.)
- Future v1: Scraping ETL MTGTOP8 + MTGGoldfish con job settimanale + storage storico

---

## 8) Import/Export Universale
Priorita: Media  
Stato: Parziale (v1: endpoints stabiliti + parser per Arena/MTGO/Moxfield/text con card resolution)

### Descrizione funzionale
Import/export multi formato: Arena text, MTGO, Moxfield, Archidekt, MTGGoldfish.

### Contratto API
- `POST /api/v1/decks/import`
- `GET /api/v1/decks/{id}/export?format=arena|mtgo|moxfield|text`

Import body:
```json
{
  "format": "arena",
  "data": "..."
}
```

Risposta 200:
```json
{
  "deck_id": "uuid",
  "cards_parsed": 60,
  "warnings": []
}
```

### Dipendenze tecniche
- Parser dedicati per sorgente
- Normalizzazione nomi carta su resolver gia presente

---

## 9) Collection Gap Analysis
Priorita: Bassa  
Stato: Parziale (v1 endpoint disponibile con inventory via query `owned`; manca persistenza inventory utente su DB)

### Descrizione funzionale
Confronta collezione utente vs deck target e mostra missing + costo totale.

### Contratto API
`GET /api/v1/users/me/collection/gaps/{deck_id}`

Query opzionale v1:
- `owned=Card Name:Qty,Other Card:Qty` (ripetibile)

Risposta 200:
```json
{
  "deck_id": "uuid",
  "completion_pct": 73,
  "missing": [
    { "card": "Force of Will", "qty": 2, "price_usd": 94.0, "line_total_usd": 188.0 }
  ],
  "total_to_acquire_usd": 210.5,
  "inventory_source": "query_owned_v1"
}
```

### Dipendenze tecniche
- Handler: `DeckHandler.CollectionGaps()`
- Route protetta JWT: `GET /api/v1/users/me/collection/gaps/{deck_id}`
- Price service: riuso `extractCardUnitPrices` (feature 6)
- Inventory v1: query parsing (`owned`) in `parseOwnedInventoryFromQuery`
- Future v2: inventory utente persistito (Mongo collection dedicata) con sync da scanner/import

---

## 10) Deck Versioning & Changelog
Priorita: Bassa  
Stato: Parziale (v1 disponibile con versioning embedded nel deck document + restore endpoint)

### Descrizione funzionale
Snapshot versioni deck, diff add/remove, restore versione precedente, note.

### Contratto API
- `GET /api/v1/decks/{id}/history`
- `POST /api/v1/decks/{id}/restore/{version}`

Risposta history:
```json
{
  "versions": [
    {
      "v": 3,
      "date": "2026-03-28",
      "changes": [
        { "op": "add", "card": "Ragavan, Nimble Pilferer", "qty": 2 },
        { "op": "remove", "card": "Goblin Guide", "qty": 2 }
      ],
      "note": "swap after FNM"
    }
  ]
}
```

### Dipendenze tecniche
- Implementato v1 snapshot diff-based embedded in `domain.Deck`:
  - `version` corrente
  - `history[]` con `v`, `date`, `changes[]`, `note`, `snapshot[]`
- Diff calcolato lato handler su update/restore (`add`/`remove` con qty)
- Restore crea sempre una nuova versione append-only (audit trail)
- Future v2: collection storica dedicata/event sourcing per deck molto grandi

---

## 11) Card Hover Preview HD
Priorita: UX  
Stato: Parziale avanzato (esteso su DeckLibrary, SideboardCoach, Analyzer legality, ScoreDetail; DFC flip + cache LRU/TTL implementati)

### Descrizione funzionale
Hover su nome carta mostra preview HD Scryfall, supporto DFC con flip animato, disponibile in ogni area UI.

### Contratto tecnico frontend
- Nessun endpoint backend obbligatorio
- Fetch diretto:
  - `https://api.scryfall.com/cards/named?exact={name}`
  - usare `image_uris` / `card_faces[].image_uris`

### Dipendenze tecniche
- LRU cache in-memory lato client (TTL 24h + eviction)
- Supporto DFC/multi-face con flip UI (front/back)
- Future v2: Service Worker cache per immagini Scryfall cross-session

---

## 12) Visual Deck Builder Drag & Drop
Priorita: UX  
Stato: Parziale (v1 frontend disponibile con DnD Main/Side/Maybe e sync realtime deck text)

### Descrizione funzionale
Builder visuale con tile immagini carta, DnD tra main/side/maybeboard, sync realtime con deck text view.

### Contratto API
`PATCH /api/v1/decks/{id}`

Body:
```json
{
  "cards": [
    { "name": "Lightning Bolt", "qty": 4, "board": "main" }
  ]
}
```

### Dipendenze tecniche
- DnD Kit (`@dnd-kit/core`, `@dnd-kit/sortable`, `@dnd-kit/utilities`) implementato
- Componente: `web/src/components/VisualDeckBuilder.jsx`
- Integrazione App: tab `Builder` con sync diretto su `sharedDecklist`
- Hover preview su card tiles (riuso `CardHoverPreview`)
- V1: sync realtime deck text; persistenza via endpoint save deck già esistente
- Future v2: patch endpoint dedicato board-aware + reorder persistito per deck salvati

---

## 13) Notifiche Ban List & Rotation
Priorita: UX  
Stato: Parziale

### Descrizione funzionale
Notifica utenti quando carta del deck viene bannata/ruota fuori formato, con suggerimento replacement.

### Contratto API
- `POST /api/v1/webhooks/scryfall`
- `GET /api/v1/users/me/notifications`

Risposta feed:
```json
{
  "items": [
    {
      "type": "banlist",
      "deck_id": "uuid",
      "card": "Card Name",
      "message": "Card banned in Modern",
      "replacement_suggestion": "Alternative Card",
      "created_at": "2026-03-31T10:20:00Z"
    }
  ]
}
```

### Dipendenze tecniche
- Ingestion webhook
- Notification store
- Email provider (Resend/SendGrid) [non implementato in v1]

### Stato implementazione v1 (backend)
- Endpoint `POST /api/v1/webhooks/scryfall` implementato con ingest JSON (evento singolo o batch).
- Endpoint `GET /api/v1/users/me/notifications` implementato con feed utente JWT-protetto.
- Notification store v1 in-memory (ring cap a 500 eventi) con dedup logica lato feed.
- Generazione notifiche da doppia fonte:
  - eventi webhook compatibili per carta/formato
  - rilevazione real-time carte non legali nei deck utente
- Suggerimento replacement basilare incluso nel payload.

---

## Piano delivery consigliato (incrementale)

### Wave 1 (2-3 settimane)
- Feature 1 hardening endpoint deck-centric
- Feature 2 endpoint legality dedicato + cache ban list
- Feature 3 completion simulatore 10k con benchmark deterministic

### Wave 2 (2-4 settimane)
- Feature 4 synergy detector v1 rule-based
- Feature 5 sideboard suggest 15-card meta-aware
- Feature 11 rollout preview HD globale + SW cache

### Wave 3 (4-6 settimane)
- Feature 6 price + budget optimizer
- Feature 7 meta dashboard ETL
- Feature 8 import/export universale

### Wave 4 (3-5 settimane)
- Feature 9 collection gaps
- Feature 10 versioning + restore
- Feature 12 visual builder DnD
- Feature 13 notifiche ban/rotation

## Criteri di accettazione trasversali
- Tutti i nuovi endpoint sotto `/api/v1`
- Test unit e integration per ogni handler/usecase nuovo
- SLA p95 endpoint read <= 400ms con cache warm
- Eventuali dipendenze esterne protette da timeout, retry e fallback
- Frontend mobile-first e i18n IT/EN coerente
