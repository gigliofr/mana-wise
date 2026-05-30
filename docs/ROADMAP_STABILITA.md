# Roadmap di stabilita' e sviluppo

Obiettivo: ridurre il rischio operativo globale del progetto e costruire nuove feature senza aumentare la fragilita' del sistema.

## Principi operativi

- Prima stabilita', poi feature.
- Ogni nuova feature deve avere test, fallback e osservabilita' minimi.
- Le funzionalita' costose vanno rese cacheable o degradabili.
- Nessun endpoint critico deve dipendere da un solo provider esterno.

## Fase 1: stabilita' di base

### Task 1.1 - Osservabilita' minima

- Aggiungere correlation id per request.
- Rendere i log strutturati e uniformi.
- Esporre metriche base: latenza, errori, fallback, cache hit/miss.
- Verifica: dashboard o endpoint di metriche consultabile.

### Task 1.2 - Timeout e fallback

- Uniformare i timeout sui provider esterni.
- Introdurre retry limitato solo dove serve davvero.
- Rendere il fallback leggibile per l'utente.
- Verifica: ogni flusso critico ha un fallback noto.

### Task 1.3 - Cache dei path costosi

- Cache per analisi deck.
- Cache per lookup card e legality.
- Cache per rendering PDF/share dove possibile.
- Verifica: riduzione chiamate ripetute su richieste identiche.

### Task 1.4 - Test di contratto

- Snapshot test sugli endpoint principali.
- Test di regressione per PDF, share, mulligan, sideboard, synergies.
- Verifica: `go test ./...` verde e snapshot stabili.

## Fase 2: affidabilita' funzionale

### Task 2.1 - Legality real-time robusta

- Estrarre servizio legality dedicato.
- Memorizzare ban list con TTL.
- Aggiungere endpoint deck-centric coerente.
- Verifica: formato e ban note sempre disponibili.

### Task 2.2 - Mulligan simulator stabile

- Consolidare simulazione Monte Carlo batched.
- Rendere deterministico il seed nei test.
- Aggiungere raccomandazione keep/mulligan con motivazione.
- Verifica: output coerente e ripetibile.

### Task 2.3 - Synergy detector piu' solido

- Tenere una knowledge base locale delle combo note.
- Riconoscere package sinergici comuni.
- Separare rule-based e ranking semantico.
- Verifica: output spiegabile e non vuoto su deck campione.

## Fase 3: nuove feature ad alto valore

### Task 3.1 - Upgrade assistant

- Suggerire sostituzioni per budget, formato e archetipo.
- Ordinare i consigli per impatto.
- Verifica: suggerimenti pratici e motivati.

### Task 3.2 - Deck diff e versioning utile

- Mostrare differenze tra versioni del mazzo.
- Evidenziare impatto su curva, terra, interaction e legality.
- Verifica: confronto leggibile e utile.

### Task 3.3 - Meta alerts e trend

- Segnalare quando il meta cambia in modo rilevante.
- Notificare impatto su deck salvati o preferiti.
- Verifica: alert comprensibili e limitati al necessario.

### Task 3.4 - Sideboard coach avanzato

- Sviluppare piani per matchup e meta snapshot.
- Generare sideboard da 15 carte con rationale.
- Verifica: sideboard completa e orientata ai pairing.

## Fase 4: UX e collaborazione

### Task 4.1 - Stato AI trasparente

- Mostrare la sorgente AI usata.
- Segnalare fallback e modalita' interna.
- Verifica: nessun errore tecnico grezzo visibile all'utente.

### Task 4.2 - Condivisione avanzata

- Rafforzare PDF e link condivisibili.
- Aggiungere report sintetici per deck e analisi.
- Verifica: share sempre stabile e leggibile.

### Task 4.3 - Note e collaborazione

- Aggiungere note sui deck.
- Consentire commenti o tag interni.
- Verifica: metadati utili senza rompere il flusso principale.

## Ordine di esecuzione consigliato

1. Osservabilita' minima.
2. Timeout, fallback e cache.
3. Test di contratto e regressione.
4. Legality, mulligan e synergy.
5. Upgrade assistant e deck diff.
6. Meta alerts, sideboard avanzato e collaborazione.

## Definizione di done per ogni task

- Il codice compila.
- I test toccati passano.
- Il comportamento e' osservabile.
- Il fallback e' chiaro.
- La documentazione minima e' aggiornata.

## Regola di lavoro

- Ogni task deve essere abbastanza piccolo da stare in una singola PR o commit logico.
- Se un task supera una giornata, va spezzato ulteriormente.
- Se una feature tocca piu' di tre moduli, prima si crea un task tecnico di stabilizzazione.
