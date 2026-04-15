package usecase

import (
	"context"
	"errors"
	"testing"

	"github.com/gigliofr/mana-wise/domain"
	"github.com/gigliofr/mana-wise/infrastructure/scryfall"
)

type fakeCardFetcher struct {
	exact map[string]*scryfall.ScryfallCard
	fuzzy map[string]*scryfall.ScryfallCard
	bySet map[string]*scryfall.ScryfallCard
}

func (f *fakeCardFetcher) GetCardByName(ctx context.Context, name string) (*scryfall.ScryfallCard, error) {
	if c, ok := f.exact[name]; ok {
		return c, nil
	}
	return nil, errors.New("not found")
}

func (f *fakeCardFetcher) GetCardByFuzzyName(ctx context.Context, name string) (*scryfall.ScryfallCard, error) {
	if c, ok := f.fuzzy[name]; ok {
		return c, nil
	}
	return nil, errors.New("not found")
}

func (f *fakeCardFetcher) GetCardBySetCollector(ctx context.Context, setCode, collectorNumber string) (*scryfall.ScryfallCard, error) {
	key := setCode + "#" + collectorNumber
	if c, ok := f.bySet[key]; ok {
		return c, nil
	}
	return nil, errors.New("not found")
}

type fakeCardRepo struct {
	byName map[string]*domain.Card
	upsert []*domain.Card
}

func (r *fakeCardRepo) FindByID(ctx context.Context, id string) (*domain.Card, error) {
	return nil, errors.New("not implemented")
}

func (r *fakeCardRepo) FindByScryfallID(ctx context.Context, scryfallID string) (*domain.Card, error) {
	return nil, errors.New("not implemented")
}

func (r *fakeCardRepo) FindByName(ctx context.Context, name string) (*domain.Card, error) {
	if c, ok := r.byName[name]; ok {
		return c, nil
	}
	return nil, errors.New("not found")
}

func (r *fakeCardRepo) FindByNames(ctx context.Context, names []string) ([]*domain.Card, error) {
	out := make([]*domain.Card, 0, len(names))
	for _, n := range names {
		if c, ok := r.byName[n]; ok {
			out = append(out, c)
		}
	}
	return out, nil
}

func (r *fakeCardRepo) FindForEmbedding(ctx context.Context, limit int, onlyMissing bool) ([]*domain.Card, error) {
	return nil, errors.New("not implemented")
}

func (r *fakeCardRepo) Upsert(ctx context.Context, card *domain.Card) error {
	if r.byName == nil {
		r.byName = map[string]*domain.Card{}
	}
	r.byName[card.Name] = card
	r.upsert = append(r.upsert, card)
	return nil
}

func (r *fakeCardRepo) UpsertMany(ctx context.Context, cards []*domain.Card) error {
	for _, c := range cards {
		if err := r.Upsert(ctx, c); err != nil {
			return err
		}
	}
	return nil
}

func (r *fakeCardRepo) UpdateEmbedding(ctx context.Context, id string, vector []float64) error {
	return errors.New("not implemented")
}

func (r *fakeCardRepo) FindWithEmbeddings(ctx context.Context, limit int) ([]*domain.Card, error) {
	return nil, errors.New("not implemented")
}

func (r *fakeCardRepo) CountAll(ctx context.Context) (int64, error) {
	return 0, errors.New("not implemented")
}

func TestAnalyzeDeckExecute_LocalizedNameResolvedByFuzzy(t *testing.T) {
	fetcher := &fakeCardFetcher{
		exact: map[string]*scryfall.ScryfallCard{},
		fuzzy: map[string]*scryfall.ScryfallCard{
			"Elfi di Llanowar": {
				ID:         "card-llanowar-elves",
				Name:       "Llanowar Elves",
				CMC:        1,
				TypeLine:   "Creature - Elf Druid",
				OracleText: "{T}: Add {G}.",
				Colors:     []string{"G"},
				Legalities: map[string]string{"standard": "legal"},
			},
		},
	}
	repo := &fakeCardRepo{byName: map[string]*domain.Card{}}
	uc := NewAnalyzeDeckUseCase(fetcher, repo, 2)

	resp, err := uc.Execute(context.Background(), AnalyzeDeckRequest{
		Decklist: "Mazzo\n4 Elfi di Llanowar (FDN) 227",
		Format:   "standard",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.RawCards) != 1 {
		t.Fatalf("expected 1 resolved card entry, got %d", len(resp.RawCards))
	}
	if resp.RawCards[0].Name != "Llanowar Elves" {
		t.Fatalf("expected canonical English card name, got %q", resp.RawCards[0].Name)
	}
	if resp.Result.Mana.TotalCards != 4 {
		t.Fatalf("expected quantity-aware mana total 4, got %d", resp.Result.Mana.TotalCards)
	}
	if len(repo.upsert) != 1 {
		t.Fatalf("expected 1 upsert for fetched card, got %d", len(repo.upsert))
	}
}

func TestAnalyzeDeckExecute_ResolvesBySetCollectorBeforeName(t *testing.T) {
	fetcher := &fakeCardFetcher{
		exact: map[string]*scryfall.ScryfallCard{},
		fuzzy: map[string]*scryfall.ScryfallCard{},
		bySet: map[string]*scryfall.ScryfallCard{
			"eoe#276": {
				ID:              "forest-eoe-276",
				Name:            "Forest",
				TypeLine:        "Basic Land - Forest",
				Set:             "eoe",
				CollectorNumber: "276",
				Legalities:      map[string]string{"standard": "legal"},
			},
		},
	}
	repo := &fakeCardRepo{byName: map[string]*domain.Card{}}
	uc := NewAnalyzeDeckUseCase(fetcher, repo, 2)

	resp, err := uc.Execute(context.Background(), AnalyzeDeckRequest{
		Decklist: "Mazzo\n12 Foresta (EOE) 276",
		Format:   "standard",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.RawCards) != 1 {
		t.Fatalf("expected 1 resolved card entry, got %d", len(resp.RawCards))
	}
	if !resp.RawCards[0].IsLand() {
		t.Fatalf("expected resolved card to be identified as land, got type line %q", resp.RawCards[0].TypeLine)
	}
	if resp.Result.Mana.LandCount != 12 {
		t.Fatalf("expected land count 12, got %d", resp.Result.Mana.LandCount)
	}
}
