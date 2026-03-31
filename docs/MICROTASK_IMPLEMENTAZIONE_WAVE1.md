# Microtask Implementazione - Wave 1 (Alta Priorita)

Data: 2026-03-31

## Obiettivo Wave 1
Consolidare le feature ad alta priorita in endpoint deck-centric riusabili dal frontend.

## Feature 2 - Format legality checker real-time
- [x] MT-2.1 Definire response envelope `deck_id + formats + checked_at`
- [x] MT-2.2 Implementare `GET /api/v1/decks/{id}/legality`
- [x] MT-2.3 Risolvere carte dal deck (by `card_id`, fallback by `card_name`)
- [x] MT-2.4 Riutilizzare legality engine esistente (`DetermineDeckLegalityAllFormats`)
- [x] MT-2.5 Registrare route nel router principale
- [x] MT-2.6 Aggiungere test handler per caso OK e card missing
- [x] MT-2.7 Integrare badge per-carta nel frontend deck editor/library

## Feature 1 - Mana curve analyzer avanzato (deck endpoint)
- [x] MT-1.1 Aggiungere endpoint `GET /api/v1/decks/{id}/analysis`
- [x] MT-1.2 Estrarre breakdown CMC per tipo (creature/instant/sorcery/enchantment/artifact/planeswalker)
- [ ] MT-1.3 Esportare `meta_fit_score` e `deviation_from_meta`
- [ ] MT-1.4 Test integrazione endpoint

## Feature 3 - Mulligan simulator avanzato
- [x] MT-3.1 Esporre endpoint deck-centric `POST /api/v1/decks/{id}/simulate`
- [ ] MT-3.2 Parametro simulazioni (default 10k, max guardrail)
- [ ] MT-3.3 Esporre metriche richieste (`p_two_lands_t2`, `p_one_drop`, `curve_out_t4`)
- [ ] MT-3.4 Motivazione keep/mulligan strutturata

## Note
- L'implementazione mantiene retrocompatibilita con endpoint legacy (`/mulligan/simulate`, `/analyze`).
- I microtask rimanenti verranno implementati nei prossimi commit incrementali.
