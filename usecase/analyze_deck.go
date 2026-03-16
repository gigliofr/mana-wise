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
	RawCards   []*domain.Card   // resolved domain cards
	Quantities map[string]int   // cardID -> total quantity in decklist
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
	allCards := make([]*domain.Card, 0, len(names))
	quantities := make(map[string]int, len(entries))
	for _, e := range entries {
		card, ok := dbIndex[e.name]
		if !ok {
			return nil, fmt.Errorf("card not found: %q", e.name)
		}
		allCards = append(allCards, card)
		quantities[card.ID] += e.qty
	}

	// Step 3: Deterministic analyses.
	manaResult := AnalyzeManaCurve(allCards, quantities, req.Format)
	interactionResult := AnalyzeInteraction(allCards, quantities, req.Format)

	resp := &AnalyzeDeckResponse{
		Result: domain.AnalysisResult{
			Format:      req.Format,
			Mana:        manaResult,
			Interaction: interactionResult,
			LatencyMs:   time.Since(start).Milliseconds(),
		},
		RawCards:   allCards,
		Quantities: quantities,
	}
	return resp, nil
}

// deckEntry is one parsed line from a decklist.
type deckEntry struct {
	qty  int
	name string
}

// parseDecklist converts a raw decklist string into entries.
// Supported formats:
//
//	"4 Lightning Bolt"
//	"1x Birds of Paradise"
//	"Sideboard:" headers and blank lines are silently ignored.
func parseDecklist(raw string) ([]deckEntry, error) {
	var entries []deckEntry
	for _, line := range strings.Split(raw, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasSuffix(strings.ToLower(line), ":") || isDeckHeader(line) {
			continue
		}

		// Strip inline comments "//..."
		if idx := strings.Index(line, "//"); idx != -1 {
			line = strings.TrimSpace(line[:idx])
		}

		parts := strings.Fields(line)
		if len(parts) < 2 {
			continue // not enough tokens — skip (could be a tag)
		}

		qtyStr := strings.TrimSuffix(parts[0], "x")
		qty, err := strconv.Atoi(qtyStr)
		if err != nil || qty <= 0 {
			// First token is not a number; treat whole line as name with qty=1.
			entries = append(entries, deckEntry{qty: 1, name: strings.Join(parts, " ")})
			continue
		}

		name := sanitizeCardName(strings.Join(parts[1:], " "))
		if name == "" {
			continue
		}
		entries = append(entries, deckEntry{qty: qty, name: name})
	}
	return entries, nil
}

func isDeckHeader(line string) bool {
	header := strings.ToLower(strings.TrimSpace(line))
	return header == "deck" || header == "mazzo"
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
