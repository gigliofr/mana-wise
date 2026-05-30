package usecase

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/gigliofr/mana-wise/domain"
	"github.com/gigliofr/mana-wise/infrastructure/cache"
	"github.com/gigliofr/mana-wise/infrastructure/scryfall"
)

type fakeCardFetcher struct {
	exact      map[string]*scryfall.ScryfallCard
	fuzzy      map[string]*scryfall.ScryfallCard
	bySet      map[string]*scryfall.ScryfallCard
	collection map[string]*scryfall.ScryfallCard
	exactCalls int
	fuzzyCalls int
	bySetCalls int
	collectionCalls int
}

func (f *fakeCardFetcher) GetCardByName(ctx context.Context, name string) (*scryfall.ScryfallCard, error) {
	f.exactCalls++
	if c, ok := f.exact[name]; ok {
		return c, nil
	}
	return nil, errors.New("not found")
}

func (f *fakeCardFetcher) GetCardByFuzzyName(ctx context.Context, name string) (*scryfall.ScryfallCard, error) {
	f.fuzzyCalls++
	if c, ok := f.fuzzy[name]; ok {
		return c, nil
	}
	return nil, errors.New("not found")
}

func (f *fakeCardFetcher) GetCardBySetCollector(ctx context.Context, setCode, collectorNumber string) (*scryfall.ScryfallCard, error) {
	f.bySetCalls++
	key := setCode + "#" + collectorNumber
	if c, ok := f.bySet[key]; ok {
		return c, nil
	}
	return nil, errors.New("not found")
}

func (f *fakeCardFetcher) FetchCardsByCollection(ctx context.Context, identifiers []scryfall.CollectionIdentifier) ([]scryfall.ScryfallCard, error) {
	f.collectionCalls++
	out := make([]scryfall.ScryfallCard, 0, len(identifiers))
	for _, id := range identifiers {
		if c, ok := f.collection[id.Name]; ok {
			out = append(out, *c)
		}
	}
	return out, nil
}

type fakeCardRepo struct {
	byName            map[string]*domain.Card
	upsert            []*domain.Card
	findByNamesResult []*domain.Card
	findByNamesCalls  int
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
	r.findByNamesCalls++
	if r.findByNamesResult != nil {
		return r.findByNamesResult, nil
	}
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

func TestAnalyzeDeckExecute_BatchCollectionResolutionUsesCollectionAPI(t *testing.T) {
	fetcher := &fakeCardFetcher{
		exact: map[string]*scryfall.ScryfallCard{},
		fuzzy: map[string]*scryfall.ScryfallCard{},
		bySet: map[string]*scryfall.ScryfallCard{},
		collection: map[string]*scryfall.ScryfallCard{
			"Lightning Bolt": {
				ID:         "bolt-1",
				Name:       "Lightning Bolt",
				CMC:        1,
				TypeLine:   "Instant",
				OracleText: "Deal 3 damage to any target.",
				Colors:     []string{"R"},
				Legalities: map[string]string{"modern": "legal"},
			},
			"Mountain": {
				ID:         "mountain-1",
				Name:       "Mountain",
				CMC:        0,
				TypeLine:   "Basic Land - Mountain",
				OracleText: "{T}: Add {R}.",
				Colors:     []string{},
				Legalities: map[string]string{"modern": "legal"},
			},
		},
	}
	repo := &fakeCardRepo{byName: map[string]*domain.Card{}}
	uc := NewAnalyzeDeckUseCase(fetcher, repo, 2)

	resp, err := uc.Execute(context.Background(), AnalyzeDeckRequest{
		Decklist: "4 Lightning Bolt\n20 Mountain",
		Format:   "modern",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.RawCards) != 2 {
		t.Fatalf("expected 2 resolved cards, got %d", len(resp.RawCards))
	}
	if fetcher.collectionCalls == 0 {
		t.Fatal("expected collection API path to be used")
	}
	if fetcher.exactCalls != 0 || fetcher.fuzzyCalls != 0 || fetcher.bySetCalls != 0 {
		t.Fatalf("expected batch path to avoid per-card fetches, got exact=%d fuzzy=%d bySet=%d", fetcher.exactCalls, fetcher.fuzzyCalls, fetcher.bySetCalls)
	}
	if repo.findByNamesCalls != 1 {
		t.Fatalf("expected one DB prefetch call, got %d", repo.findByNamesCalls)
	}
}

func TestAnalyzeDeckExecute_ReturnsErrorWhenResolverReturnsNilCard(t *testing.T) {
	fetcher := &fakeCardFetcher{
		exact: map[string]*scryfall.ScryfallCard{
			"Ghost Card": nil,
		},
		fuzzy: map[string]*scryfall.ScryfallCard{},
		bySet: map[string]*scryfall.ScryfallCard{},
	}
	repo := &fakeCardRepo{byName: map[string]*domain.Card{}}
	uc := NewAnalyzeDeckUseCase(fetcher, repo, 2)

	_, err := uc.Execute(context.Background(), AnalyzeDeckRequest{
		Decklist: "4 Ghost Card",
		Format:   "standard",
	})
	if err == nil {
		t.Fatal("expected error when resolver returns nil card")
	}
}

func TestAnalyzeDeckExecute_SkipsUnresolvedCardsAndReturnsWarnings(t *testing.T) {
	fetcher := &fakeCardFetcher{
		exact: map[string]*scryfall.ScryfallCard{
			"Lightning Bolt": {
				ID:         "bolt-1",
				Name:       "Lightning Bolt",
				CMC:        1,
				TypeLine:   "Instant",
				OracleText: "Deal 3 damage to any target.",
				Colors:     []string{"R"},
				Legalities: map[string]string{"modern": "legal"},
			},
		},
		fuzzy: map[string]*scryfall.ScryfallCard{},
		bySet: map[string]*scryfall.ScryfallCard{},
	}
	repo := &fakeCardRepo{byName: map[string]*domain.Card{}}
	uc := NewAnalyzeDeckUseCase(fetcher, repo, 2)

	resp, err := uc.Execute(context.Background(), AnalyzeDeckRequest{
		Decklist: "4 Lightning Bolt\n1 Missing Card",
		Format:   "modern",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.RawCards) != 1 {
		t.Fatalf("expected 1 resolved card, got %d", len(resp.RawCards))
	}
	if len(resp.Warnings) == 0 {
		t.Fatal("expected warnings for unresolved cards")
	}
	if got := resp.Warnings[0]; got == "" || !strings.Contains(got, "Missing Card") {
		t.Fatalf("expected warning mentioning missing card, got %q", got)
	}
}

func TestAnalyzeDeckExecute_DBReturnsNilCardEntry_DoesNotPanic(t *testing.T) {
	fetcher := &fakeCardFetcher{
		exact: map[string]*scryfall.ScryfallCard{
			"Lightning Bolt": {
				ID:         "bolt-1",
				Name:       "Lightning Bolt",
				CMC:        1,
				TypeLine:   "Instant",
				OracleText: "Deal 3 damage to any target.",
				Colors:     []string{"R"},
				Legalities: map[string]string{"standard": "not_legal", "modern": "legal"},
			},
		},
		fuzzy: map[string]*scryfall.ScryfallCard{},
		bySet: map[string]*scryfall.ScryfallCard{},
	}
	repo := &fakeCardRepo{
		byName:            map[string]*domain.Card{},
		findByNamesResult: []*domain.Card{nil},
	}
	uc := NewAnalyzeDeckUseCase(fetcher, repo, 2)

	resp, err := uc.Execute(context.Background(), AnalyzeDeckRequest{
		Decklist: "4 Lightning Bolt",
		Format:   "modern",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp == nil || len(resp.RawCards) == 0 {
		t.Fatalf("expected resolved cards, got %+v", resp)
	}
}

func TestAnalyzeDeckExecute_UsesCacheForRepeatedDecklist(t *testing.T) {
	fetcher := &fakeCardFetcher{
		exact: map[string]*scryfall.ScryfallCard{
			"Lightning Bolt": {
				ID:         "bolt-1",
				Name:       "Lightning Bolt",
				CMC:        1,
				TypeLine:   "Instant",
				OracleText: "Deal 3 damage to any target.",
				Colors:     []string{"R"},
				Legalities: map[string]string{"modern": "legal"},
			},
		},
	}
	repo := &fakeCardRepo{byName: map[string]*domain.Card{}}
	uc := NewAnalyzeDeckUseCase(fetcher, repo, 2).WithCache(cache.New(), time.Hour)
	req := AnalyzeDeckRequest{Decklist: "4 Lightning Bolt", Format: "modern"}

	first, err := uc.Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("first execution failed: %v", err)
	}
	if first == nil {
		t.Fatal("expected first response")
	}
	first.Result.LatencyMs = 999

	second, err := uc.Execute(context.Background(), req)
	if err != nil {
		t.Fatalf("second execution failed: %v", err)
	}
	if second == nil {
		t.Fatal("expected second response")
	}
	if fetcher.exactCalls != 1 || repo.findByNamesCalls != 1 {
		t.Fatalf("expected cached second execution, got exactCalls=%d findByNamesCalls=%d", fetcher.exactCalls, repo.findByNamesCalls)
	}
	if second.Result.LatencyMs == 999 {
		t.Fatal("expected cached response to be cloned, not reused by reference")
	}
}
