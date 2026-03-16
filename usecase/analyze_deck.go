package usecase

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/manawise/api/domain"
	"github.com/manawise/api/infrastructure/scryfall"
)

// CardFetcher can resolve cards by name (Scryfall or DB).
type CardFetcher interface {
	GetCardByName(ctx context.Context, name string) (*scryfall.ScryfallCard, error)
	GetCardByFuzzyName(ctx context.Context, name string) (*scryfall.ScryfallCard, error)
}

// AnalyzeDeckRequest is the input for the AnalyzeDeck use case.
type AnalyzeDeckRequest struct {
	Decklist string // raw decklist, one card per line: "4 Lightning Bolt"
	Format   string
}

// AnalyzeDeckResponse is the output of the AnalyzeDeck use case.
type AnalyzeDeckResponse struct {
	Result     domain.AnalysisResult
	RawCards   []*domain.Card // resolved domain cards
	Quantities map[string]int // cardID -> total quantity in decklist
	Commander  *CommanderInfo // populated for "commander" format decks
	Sideboard  *SideboardInfo // populated when decklist contains a Sideboard section
}

// SideboardInfo is populated when a Sideboard / SB: section is detected in the decklist.
// SideboardInfo is included in AnalyzeDeckResponse when a sideboard is present.
type SideboardInfo struct {
	Quantities map[string]int `json:"quantities"` // cardID -> quantity
	TotalCards int            `json:"total_cards"`
}

// CommanderCards holds the resolved commander cards (up to 2 for Partner pairs).
// Only populated for "commander" format.
type CommanderInfo struct {
	Cards []*domain.Card `json:"cards"`
}

// AnalyzeDeckUseCase orchestrates card resolution + deterministic analysis.
type AnalyzeDeckUseCase struct {
	fetcher  CardFetcher
	cardRepo domain.CardRepository
	poolSize int
}

// NewAnalyzeDeckUseCase creates a new AnalyzeDeckUseCase.
func NewAnalyzeDeckUseCase(fetcher CardFetcher, cardRepo domain.CardRepository, poolSize int) *AnalyzeDeckUseCase {
	if poolSize <= 0 {
		poolSize = 20
	}
	return &AnalyzeDeckUseCase{fetcher: fetcher, cardRepo: cardRepo, poolSize: poolSize}
}

// Execute runs the full deterministic analysis pipeline and returns an AnalyzeDeckResponse.
// It respects ctx deadlines so callers can impose an overall timeout.
func (uc *AnalyzeDeckUseCase) Execute(ctx context.Context, req AnalyzeDeckRequest) (*AnalyzeDeckResponse, error) {
	if !domain.IsValidFormat(req.Format) {
		return nil, fmt.Errorf("unsupported format: %s", req.Format)
	}

	start := time.Now()

	entries, err := parseDecklist(req.Decklist)
	if err != nil {
		return nil, fmt.Errorf("parse decklist: %w", err)
	}
	if len(entries) == 0 {
		return nil, fmt.Errorf("decklist is empty")
	}

	// Collect unique card names.
	names := make([]string, 0, len(entries))
	nameSet := make(map[string]bool)
	for _, e := range entries {
		if !nameSet[e.name] {
			names = append(names, e.name)
			nameSet[e.name] = true
		}
	}

	// Separate entries by zone.
	var mainEntries, commanderEntries, sideboardEntries []deckEntry
	for _, e := range entries {
		switch {
		case e.isCommander:
			commanderEntries = append(commanderEntries, e)
		case e.isSideboard:
			sideboardEntries = append(sideboardEntries, e)
		default:
			mainEntries = append(mainEntries, e)
		}
	}

	// Validate sideboard size for non-commander formats.
	const maxSideboardSize = 15
	sideboardTotal := 0
	for _, e := range sideboardEntries {
		sideboardTotal += e.qty
	}
	if sideboardTotal > maxSideboardSize {
		return nil, fmt.Errorf("sideboard exceeds maximum of %d cards (found %d)", maxSideboardSize, sideboardTotal)
	}

	// Step 1: Try to resolve from local DB first (batch).
	dbCards, err := uc.cardRepo.FindByNames(ctx, names)
	if err != nil {
		return nil, fmt.Errorf("DB lookup: %w", err)
	}

	dbIndex := make(map[string]*domain.Card, len(dbCards))
	for _, c := range dbCards {
		dbIndex[c.Name] = c
	}

	// Step 2: Fetch missing cards from Scryfall using Worker Pool.
	var missing []string
	for _, name := range names {
		if _, found := dbIndex[name]; !found {
			missing = append(missing, name)
		}
	}

	if len(missing) > 0 {
		results := WorkerPool(ctx, uc.poolSize, missing,
			func(ctx context.Context, name string) (*domain.Card, error) {
				sc, err := uc.fetcher.GetCardByName(ctx, name)
				if err != nil {
					// Fallback to fuzzy matching for localized names or small typos.
					sc, err = uc.fetcher.GetCardByFuzzyName(ctx, name)
					if err != nil {
						return nil, err
					}
				}
				card := scryfall.ToDomainCard(sc)
				// Persist to DB in background; ignore error for performance.
				_ = uc.cardRepo.Upsert(ctx, card)
				return card, nil
			},
		)

		for _, r := range results {
			if r.Err != nil {
				return nil, fmt.Errorf("resolve card %q: %w", r.Input, r.Err)
			}
			// Keep lookup keyed by the original decklist name to support fuzzy/localized matches.
			dbIndex[r.Input] = r.Output
			// Also store by canonical Scryfall name for future direct lookups.
			dbIndex[r.Output.Name] = r.Output
		}
	}

	// Build ordered card slice + quantity map for analysis.
	// Build mainboard + commander card slice and quantity map for analysis.
	mainCards := make([]*domain.Card, 0, len(mainEntries)+len(commanderEntries))
	quantities := make(map[string]int)
	seenMain := make(map[string]bool)
	for _, e := range append(mainEntries, commanderEntries...) {
		card, ok := dbIndex[e.name]
		if !ok {
			return nil, fmt.Errorf("card not found: %q", e.name)
		}
		if !seenMain[card.ID] {
			mainCards = append(mainCards, card)
			seenMain[card.ID] = true
		}
		quantities[card.ID] += e.qty
	}

	// Build sideboard quantities (cards resolved independently above).
	var sideboardInfo *SideboardInfo
	if len(sideboardEntries) > 0 {
		sbQty := make(map[string]int, len(sideboardEntries))
		for _, e := range sideboardEntries {
			card, ok := dbIndex[e.name]
			if !ok {
				return nil, fmt.Errorf("sideboard card not found: %q", e.name)
			}
			sbQty[card.ID] += e.qty
		}
		sideboardInfo = &SideboardInfo{Quantities: sbQty, TotalCards: sideboardTotal}
	}

	// Build commander info.
	var commanderInfo *CommanderInfo
	if len(commanderEntries) > 0 {
		cmdCards := make([]*domain.Card, 0, len(commanderEntries))
		seen := make(map[string]bool)
		for _, e := range commanderEntries {
			card, ok := dbIndex[e.name]
			if !ok {
				return nil, fmt.Errorf("commander card not found: %q", e.name)
			}
			if !seen[card.ID] {
				cmdCards = append(cmdCards, card)
				seen[card.ID] = true
			}
		}
		commanderInfo = &CommanderInfo{Cards: cmdCards}
	}

	// Step 3: Deterministic analyses.
	manaResult := AnalyzeManaCurve(mainCards, quantities, req.Format)
	interactionResult := AnalyzeInteraction(mainCards, quantities, req.Format)

	resp := &AnalyzeDeckResponse{
		Result: domain.AnalysisResult{
			Format:      req.Format,
			Mana:        manaResult,
			Interaction: interactionResult,
			LatencyMs:   time.Since(start).Milliseconds(),
		},
		RawCards:   mainCards,
		Quantities: quantities,
		Commander:  commanderInfo,
		Sideboard:  sideboardInfo,
	}
	return resp, nil
}

// deckEntry is one parsed line from a decklist.
type deckEntry struct {
	qty         int
	name        string
	isCommander bool
	isSideboard bool
}

// parseDecklist converts a raw decklist string into entries.
// Supports section headers: "Deck" / "Mazzo", "Commander", "Sideboard" / "SB:".
// Arena export format example:
//
//	Commander
//	1 Atraxa, Praetors' Voice
//
//	Deck
//	4 Lightning Bolt
//
//	Sideboard
//	2 Surgical Extraction
func parseDecklist(raw string) ([]deckEntry, error) {
	var entries []deckEntry
	section := "deck" // default section
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		lower := strings.ToLower(line)

		// Detect section headers (with or without trailing colon).
		if lower == "deck" || lower == "mazzo" || lower == "deck:" || lower == "mazzo:" {
			section = "deck"
			continue
		}
		if lower == "commander" || lower == "commander:" {
			section = "commander"
			continue
		}
		if lower == "sideboard" || lower == "sideboard:" || lower == "sb:" {
			section = "sideboard"
			continue
		}
		// Skip any other header-style lines (e.g. "About", "Land:")
		if strings.HasSuffix(lower, ":") {
			continue
		}

		// Strip inline comments "//..."
		if idx := strings.Index(line, "//"); idx != -1 {
			line = strings.TrimSpace(line[:idx])
		}

		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue // not enough tokens — skip (could be a tag or annotation)
		}

		isCmd := section == "commander"
		isSB := section == "sideboard"

		qtyStr := strings.TrimSuffix(parts[0], "x")
		qty, err := strconv.Atoi(qtyStr)
		if err != nil || qty <= 0 {
			// First token is not a number; treat whole line as name with qty=1.
			entries = append(entries, deckEntry{qty: 1, name: strings.Join(parts, " "), isCommander: isCmd, isSideboard: isSB})
			continue
		}

		name := sanitizeCardName(strings.Join(parts[1:], " "))
		if name == "" {
			continue
		}
		entries = append(entries, deckEntry{qty: qty, name: name, isCommander: isCmd, isSideboard: isSB})
	}
	return entries, nil
}

// sanitizeCardName strips MTG Arena trailing metadata like "(SET) 123".
func sanitizeCardName(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return ""
	}
	parts := strings.Fields(name)
	if len(parts) >= 2 {
		last := parts[len(parts)-1]
		prev := parts[len(parts)-2]
		if isAllDigits(last) && strings.HasPrefix(prev, "(") && strings.HasSuffix(prev, ")") {
			parts = parts[:len(parts)-2]
		}
	}
	return strings.TrimSpace(strings.Join(parts, " "))
}

func isAllDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}
