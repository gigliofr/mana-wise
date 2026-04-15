package usecase_test

import (
	"context"
	"errors"
	"testing"

	"github.com/gigliofr/mana-wise/domain"
	"github.com/gigliofr/mana-wise/infrastructure/scryfall"
	"github.com/gigliofr/mana-wise/usecase"
)

type mockResolveRepo struct {
	byName    map[string]*domain.Card
	upserted  *domain.Card
	findErr   error
	upsertErr error
}

func (m *mockResolveRepo) FindByID(ctx context.Context, id string) (*domain.Card, error) {
	return nil, nil
}
func (m *mockResolveRepo) FindByScryfallID(ctx context.Context, scryfallID string) (*domain.Card, error) {
	return nil, nil
}
func (m *mockResolveRepo) FindByName(ctx context.Context, name string) (*domain.Card, error) {
	if m.findErr != nil {
		return nil, m.findErr
	}
	return m.byName[name], nil
}
func (m *mockResolveRepo) FindByNames(ctx context.Context, names []string) ([]*domain.Card, error) {
	return nil, nil
}
func (m *mockResolveRepo) FindForEmbedding(ctx context.Context, limit int, onlyMissing bool) ([]*domain.Card, error) {
	return nil, nil
}
func (m *mockResolveRepo) Upsert(ctx context.Context, card *domain.Card) error {
	if m.upsertErr != nil {
		return m.upsertErr
	}
	m.upserted = card
	return nil
}
func (m *mockResolveRepo) UpsertMany(ctx context.Context, cards []*domain.Card) error { return nil }
func (m *mockResolveRepo) UpdateEmbedding(ctx context.Context, id string, vector []float64) error {
	return nil
}
func (m *mockResolveRepo) FindWithEmbeddings(ctx context.Context, limit int) ([]*domain.Card, error) {
	return nil, nil
}
func (m *mockResolveRepo) CountAll(ctx context.Context) (int64, error) { return 0, nil }

type mockCardNameFetcher struct {
	card *scryfall.ScryfallCard
	err  error
}

func (m *mockCardNameFetcher) GetCardByName(ctx context.Context, name string) (*scryfall.ScryfallCard, error) {
	return nil, nil
}
func (m *mockCardNameFetcher) GetCardByFuzzyName(ctx context.Context, name string) (*scryfall.ScryfallCard, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.card, nil
}

func TestResolveCardByNameUseCase_UsesDBFirst(t *testing.T) {
	repo := &mockResolveRepo{byName: map[string]*domain.Card{"Lightning Bolt": {ID: "1", Name: "Lightning Bolt"}}}
	uc := usecase.NewResolveCardByNameUseCase(&mockCardNameFetcher{}, repo)

	card, err := uc.Execute(context.Background(), "Lightning Bolt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if card == nil || card.Name != "Lightning Bolt" {
		t.Fatalf("unexpected card: %+v", card)
	}
	if repo.upserted != nil {
		t.Fatal("should not upsert when card already exists locally")
	}
}

func TestResolveCardByNameUseCase_FuzzyFallback(t *testing.T) {
	repo := &mockResolveRepo{byName: map[string]*domain.Card{}}
	fetcher := &mockCardNameFetcher{card: &scryfall.ScryfallCard{ID: "abc", Name: "Lightning Bolt", TypeLine: "Instant"}}
	uc := usecase.NewResolveCardByNameUseCase(fetcher, repo)

	card, err := uc.Execute(context.Background(), "lightning boltt")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if card == nil || card.ID != "abc" {
		t.Fatalf("unexpected card: %+v", card)
	}
	if repo.upserted == nil || repo.upserted.ID != "abc" {
		t.Fatalf("expected upserted card, got %+v", repo.upserted)
	}
}

func TestResolveCardByNameUseCase_FuzzyError(t *testing.T) {
	repo := &mockResolveRepo{byName: map[string]*domain.Card{}}
	uc := usecase.NewResolveCardByNameUseCase(&mockCardNameFetcher{err: errors.New("not found")}, repo)

	_, err := uc.Execute(context.Background(), "missing card")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestResolveCardByNameUseCase_FuzzyNilCard(t *testing.T) {
	repo := &mockResolveRepo{byName: map[string]*domain.Card{}}
	uc := usecase.NewResolveCardByNameUseCase(&mockCardNameFetcher{card: nil}, repo)

	_, err := uc.Execute(context.Background(), "missing card")
	if err == nil {
		t.Fatal("expected error when resolver returns nil card")
	}
}
