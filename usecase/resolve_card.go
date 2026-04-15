package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/gigliofr/mana-wise/domain"
	"github.com/gigliofr/mana-wise/infrastructure/scryfall"
)

// CardNameFetcher resolves cards by exact and fuzzy name from external APIs.
type CardNameFetcher interface {
	GetCardByName(ctx context.Context, name string) (*scryfall.ScryfallCard, error)
	GetCardByFuzzyName(ctx context.Context, name string) (*scryfall.ScryfallCard, error)
}

// ResolveCardByNameUseCase resolves a card by name, preferring local DB and falling back to Scryfall fuzzy matching.
type ResolveCardByNameUseCase struct {
	fetcher  CardNameFetcher
	cardRepo domain.CardRepository
}

// NewResolveCardByNameUseCase creates a ResolveCardByNameUseCase.
func NewResolveCardByNameUseCase(fetcher CardNameFetcher, cardRepo domain.CardRepository) *ResolveCardByNameUseCase {
	return &ResolveCardByNameUseCase{fetcher: fetcher, cardRepo: cardRepo}
}

// Execute resolves a card by name using exact DB lookup first, then Scryfall fuzzy resolution.
func (uc *ResolveCardByNameUseCase) Execute(ctx context.Context, name string) (*domain.Card, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return nil, fmt.Errorf("card name is required")
	}

	card, err := uc.cardRepo.FindByName(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("find card by name: %w", err)
	}
	if card != nil {
		return card, nil
	}

	sc, err := uc.fetcher.GetCardByFuzzyName(ctx, name)
	if err != nil {
		return nil, fmt.Errorf("fuzzy resolve card %q: %w", name, err)
	}
	if sc == nil {
		return nil, fmt.Errorf("fuzzy resolve card %q: resolver returned nil card", name)
	}
	card = scryfall.ToDomainCard(sc)
	if card == nil {
		return nil, fmt.Errorf("fuzzy resolve card %q: failed to convert card", name)
	}
	if err = uc.cardRepo.Upsert(ctx, card); err != nil {
		return nil, fmt.Errorf("persist resolved card: %w", err)
	}
	return card, nil
}
