# Piano Unificazione Libreria Mazzi + Visual Deck Builder

Data: 2026-04-02

## 1) Analisi requisiti

Obiettivi richiesti:
- Portare la gestione piani in top bar e rendere downgrade sicuro per utenti con entitlement Pro attivo.
- Ridurre sovrapposizione tra Libreria Mazzi e Visual Deck Builder.
- Rendere il Builder una funzione Pro, con selezione/filtri carta per rarita e edizione.

Vincoli:
- Mantenere compatibilita con API e decklist testuale esistente.
- Evitare regressioni su UX mobile.
- Conservare sincronizzazione in tempo reale con deck text.

## 2) Utility MTG (prodotto)

Per utenti MTG, Libreria e Builder risolvono due bisogni distinti:
- Libreria: persistenza, scelta deck, overview legale/slot.
- Builder: iterazione veloce, tuning per meta, side/maybe management.

Filtri per rarita/edizione sono utili per:
- budget tuning (es. preferire uncommon/common)
- costruzione in limiti di pool (es. set specifici di standard)
- scouting sostituzioni in sideboard.

## 3) Soluzione architetturale proposta

### Fase A (subito)
- Builder Pro-only lato UI (gating + CTA verso piani).
- Libreria resta entrypoint principale.

### Fase B (unificazione modello)
- Introdurre `DeckWorkspace` client-side:
  - deck base da Libreria
  - stato editing (main/side/maybe)
  - filtri attivi (rarity, set)
  - dirty state e undo/redo
- Libreria e Builder diventano due viste dello stesso workspace.

### Fase C (filtri rarita/edizione)
- Aggiungere endpoint query card metadata ottimizzato per batch nomi deck.
- In builder:
  - facet filter rarita (common/uncommon/rare/mythic)
  - facet filter edizione/set code
  - toggle: filtro visivo vs filtro operativo (solo carte selezionabili)

### Fase D (persistenza avanzata)
- Salvare preferenze filtri per utente.
- Aggiungere preset: Budget, Pioneer pool, Standard set window.

## 4) Piano implementativo

Sprint 1:
- Gating Pro + UX piani conferma downgrade
- Spostamento Piani in top bar

Sprint 2:
- Workspace unificato Libreria/Builder
- Batch card metadata endpoint

Sprint 3:
- Filtri rarita/edizione completi
- Telemetria uso filtri e conversione Free->Pro

## 5) Criteri di accettazione

- Free: può vedere Builder ma non usare funzioni avanzate (CTA Pro).
- Pro: può filtrare carte per rarita e set senza perdere sincronizzazione deck text.
- Nessun downgrade anticipato se `pro_until` nel futuro.
- Nessuna regressione su salvataggio deck e legalita.
