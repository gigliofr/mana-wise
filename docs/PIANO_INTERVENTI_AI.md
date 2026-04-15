# Piano Interventi AI - ManaWise

Versione: 1.0  
Data: 2026-03-16  
Stato: In esecuzione avanzata (A+B+C+D+E quasi completate)

## 1) Obiettivo generale
Realizzare una pipeline AI robusta e sostenibile che:
- continui a funzionare quando un provider esterno e' indisponibile o in quota limit,
- riduca la dipendenza da Gemini/OpenAI,
- introduca una modalita' interna senza chiamate esterne,
- mantenga UX chiara e prevedibile lato frontend.

## 2) Scope degli interventi
Interventi inclusi:
1. Fallback multi-provider (Gemini -> Provider secondario).
2. Motore interno rule-based (nessuna chiamata LLM esterna).
3. Modalita' runtime configurabile (external, internal, hybrid).
4. Messaggistica UX per errori/quota e trasparenza del risultato.
5. Test, monitoraggio e runbook operativo.

Interventi opzionali (fase successiva):
1. LLM locale self-hosted (es. Ollama) come terzo livello.
2. Ottimizzazione qualitativa con A/B test sui suggerimenti.

## 3) Timeline stimata
Stime conservative con 1 sviluppatore full-time.

### Fase A - Hardening provider + fallback
Durata: 0.5-1.5 giorni

Output:
- Config estesa per provider secondario.
- Strategia failover su errori 429/5xx/timeout/provider unavailable.
- Mapping errori umani (gia' iniziato) completato con codici standard interni.
- Test di integrazione su chain fallback.

### Fase B - Motore interno rule-based (V1)
Durata: 1-3 giorni

Output:
- Regole di raccomandazione basate su analisi deterministica.
- Ranking priorita' (Critical/Warning/Info).
- Template suggerimenti in IT/EN.
- Endpoint/API invariati per compatibilita' frontend.

### Fase C - Modalita' operative e routing
Durata: 0.5-1 giorno

Output:
- Modalita':
  - external_only
  - internal_only
  - hybrid_prefer_external
  - hybrid_prefer_internal
- Router decisionale unico lato backend.
- Config env e documentazione.

### Fase D - UX + osservabilita'
Durata: 0.5-1 giorno

Output:
- Banner/stato AI chiaro in UI (es. quota esaurita, fallback usato).
- Tracciamento eventi analytics (provider usato, fallback attivato, tempi risposta).
- Report base qualita' suggerimenti.

### Fase E - QA, rollout, runbook
Durata: 0.5-1 giorno

Output:
- Test end-to-end e regressione.
- Piano di rilascio graduale.
- Runbook operativo (incidenti quota/provider down).

## 4) Totale effort
- MVP robusto (A+B+C): 2-5.5 giorni.
- Completo con UX/ops (A+B+C+D+E): 3-7.5 giorni.

## 5) Requisiti dati/download
### Necessita' di scaricare molti dati?
- Fallback provider: NO.
- Motore interno rule-based: NO.
- LLM locale self-hosted (opzionale): SI, 4-20+ GB in base al modello.

### Dati necessari per partire subito
- Nessun dataset esterno obbligatorio.
- Si usano le metriche gia' presenti nell'analisi deck (mana, interaction, archetype, ecc.).

## 6) Architettura target (MVP)
### Flusso decisionale suggerimenti
1. Backend calcola analisi deterministica.
2. Router AI valuta AI_MODE:
- external_only: usa provider primario, fallback su secondario se configurato.
- internal_only: usa solo motore interno.
- hybrid_prefer_external: prova esterno, poi interno.
- hybrid_prefer_internal: prova interno, poi esterno.
3. API restituisce sempre:
- ai_suggestions (stringa o lista coerente al contratto),
- ai_error (messaggio leggibile se utile),
- ai_source (gemini/openrouter/internal_rules, ecc.) [nuovo campo consigliato].

## 7) Piano tecnico dettagliato
## 7.1 Configurazione
Aggiunte env previste:
- AI_MODE=hybrid_prefer_external
- LLM_SECONDARY_PROVIDER=openai_compatible
- LLM_SECONDARY_API_KEY=
- LLM_SECONDARY_BASE_URL=
- LLM_SECONDARY_MODEL=
- AI_INTERNAL_RULES_ENABLED=true
- AI_FALLBACK_ON_STATUS=429,500,502,503,504
- AI_FALLBACK_ON_TIMEOUT_MS=7000

Accettazione:
- Avvio server ok con combinazioni minime.
- Messaggi di config invalid chiari a startup.

## 7.2 Provider failover
Implementazione:
- Interfaccia comune per generatori suggerimenti.
- Primary adapter + Secondary adapter.
- Policy fallback con retry limitato (max 1 tentativo per provider).
- Circuit breaker semplice (cooldown breve su provider fallito).

Accettazione:
- Con 429 dal primario, secondary risponde entro SLA.
- Se entrambi falliscono, entra internal (in hybrid) oppure errore umano coerente.

## 7.3 Motore interno rule-based
Implementazione V1 regole:
1. Mana base:
- terre troppo basse/alte rispetto a curva e CMC medio.
2. Interaction profile:
- rimozioni insufficienti,
- stack interaction carente,
- gestione board wipe.
3. Curva:
- eccesso slot CMC alti,
- buchi turni 1-2-3.
4. Coerenza archetype:
- suggerimenti orientati all'archetipo stimato.

Scoring:
- Punteggio impatto per regola.
- Top 3 suggerimenti con motivazione e azione.

Accettazione:
- Output sempre disponibile in internal_only.
- Coerenza lessicale IT/EN.
- Nessun riferimento a carte non presenti se non espressamente richiesto.

## 7.4 Frontend/UX
Implementazione:
- Stato sorgente AI visibile (es. Suggerimenti da: Internal Rules).
- Messaggi utente:
- Quota provider esaurita,
- Fallback attivato,
- Suggerimenti locali utilizzati.

Accettazione:
- Nessun errore tecnico grezzo mostrato all'utente.
- Test visuale desktop/mobile.

## 7.5 Testing
Test minimi:
1. Unit test policy fallback.
2. Unit test ranking regole interne.
3. Integration test endpoint /analyze con scenari:
- primario OK,
- primario 429 + secondary OK,
- entrambi KO + internal OK,
- internal_only.
4. Snapshot test messaggi UX principali.

Accettazione:
- go test ./... verde.
- build frontend verde.

## 8) Milestone e gate
Milestone 1:
- Fallback primario-secondario operativo.
Gate:
- test integrazione fallback passati.

Milestone 2:
- Internal rules in produzione dietro feature flag.
Gate:
- output stabile su set deck di riferimento.

Milestone 3:
- Modalita' hybrid default + monitoraggio.
Gate:
- error rate AI ridotto e latenza entro target.

## 9) KPI di successo
- Disponibilita' suggerimenti AI > 99% in orario operativo.
- Riduzione errori AI lato utente >= 90%.
- Tempo medio risposta suggerimenti <= +20% rispetto baseline attuale.
- Percentuale fallback attivati monitorata e in calo.

## 10) Rischi e mitigazioni
1. Rischio: differenze contratto tra provider compatibili OpenAI.
Mitigazione: adapter normalizzatore response/error.

2. Rischio: qualita' suggerimenti interni inizialmente inferiore.
Mitigazione: tuning regole su deck campione e feedback loop.

3. Rischio: complessita' operativa in hybrid.
Mitigazione: runbook semplice + metriche minime obbligatorie.

## 11) Piano di rollout
1. Dev: attiva hybrid con internal fallback.
2. Staging: test carico leggero + scenari errore simulati.
3. Produzione canary: 10-20% traffico.
4. Full rollout: 100% con monitoraggio prime 48h.

Rollback:
- impostare AI_MODE=internal_only (safe mode immediato).

## 12) Checklist esecutiva
- [x] Definire env e parsing config nuovi.
- [x] Implementare adapter provider secondario.
- [x] Implementare policy fallback + retry/cooldown.
- [x] Implementare motore internal rules V1.
- [x] Integrare AI_MODE router.
- [x] Aggiornare risposta API con ai_source.
- [x] Aggiornare UI con stato sorgente e messaggi fallback.
- [x] Scrivere test unit/integration/snapshot.
- [x] Aggiornare README e runbook operativo.
- [ ] Eseguire rollout graduale.

### Aggiornamento stato (2026-04-16)
- Completato hardening AI multi-tier con fallback configurabile (`AI_FALLBACK_ON_STATUS`, `AI_FALLBACK_ON_TIMEOUT_MS`).
- Completati test unitari fallback chain primario/secondario/interno e snapshot UI fallback.
- Completato runbook operativo fallback AI e allineamento documentazione principale.
- Rimane operativo solo il rollout graduale su ambienti con monitoraggio KPI post-rilascio.

## 13) Allegato operativo (sottomissione)
Per sottomissioni future allegare questo file insieme a:
1. changelog tecnico,
2. evidenze test (output CI),
3. report KPI prima/dopo,
4. eventuale decision log su provider scelto.

---
Autore: Team ManaWise / Copilot support
