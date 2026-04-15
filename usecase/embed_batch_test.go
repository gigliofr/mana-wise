package usecase_test

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/gigliofr/mana-wise/domain"
	"github.com/gigliofr/mana-wise/usecase"
)

type mockCardRepo struct {
	mu         sync.Mutex
	cards      []*domain.Card
	updatedIDs []string
	vectors    map[string][]float64
	failID     string
}

func (m *mockCardRepo) FindByID(ctx context.Context, id string) (*domain.Card, error) {
	return nil, nil
}
func (m *mockCardRepo) FindByScryfallID(ctx context.Context, scryfallID string) (*domain.Card, error) {
	return nil, nil
}
func (m *mockCardRepo) FindByName(ctx context.Context, name string) (*domain.Card, error) {
	return nil, nil
}
func (m *mockCardRepo) FindByNames(ctx context.Context, names []string) ([]*domain.Card, error) {
	return nil, nil
}
func (m *mockCardRepo) FindForEmbedding(ctx context.Context, limit int, onlyMissing bool) ([]*domain.Card, error) {
	if len(m.cards) > limit {
		return m.cards[:limit], nil
	}
	return m.cards, nil
}
func (m *mockCardRepo) Upsert(ctx context.Context, card *domain.Card) error        { return nil }
func (m *mockCardRepo) UpsertMany(ctx context.Context, cards []*domain.Card) error { return nil }
func (m *mockCardRepo) UpdateEmbedding(ctx context.Context, id string, vector []float64) error {
	if id == m.failID {
		return errors.New("update failed")
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.vectors == nil {
		m.vectors = map[string][]float64{}
	}
	m.updatedIDs = append(m.updatedIDs, id)
	m.vectors[id] = vector
	return nil
}
func (m *mockCardRepo) FindWithEmbeddings(ctx context.Context, limit int) ([]*domain.Card, error) {
	return nil, nil
}
func (m *mockCardRepo) CountAll(ctx context.Context) (int64, error) { return int64(len(m.cards)), nil }

type mockEmbedder struct {
	failFor string
}

func (m *mockEmbedder) EmbedText(ctx context.Context, input string) ([]float64, error) {
	if m.failFor != "" && input != "" && contains(input, m.failFor) {
		return nil, errors.New("embed failed")
	}
	return []float64{0.1, 0.2, 0.3}, nil
}

func contains(s, sub string) bool {
	return len(sub) == 0 || (len(s) >= len(sub) && indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func TestEmbedBatchUseCase_ExecuteSuccess(t *testing.T) {
	repo := &mockCardRepo{cards: []*domain.Card{
		{ID: "c1", Name: "Lightning Bolt", TypeLine: "Instant", OracleText: "Deal 3 damage to any target"},
		{ID: "c2", Name: "Counterspell", TypeLine: "Instant", OracleText: "Counter target spell"},
	}}
	uc := usecase.NewEmbedBatchUseCase(repo, &mockEmbedder{}, 2)

	res, err := uc.Execute(context.Background(), usecase.EmbedBatchRequest{Limit: 10})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Processed != 2 || res.Updated != 2 || res.Failed != 0 {
		t.Fatalf("unexpected result: %+v", res)
	}
	if len(repo.updatedIDs) != 2 {
		t.Fatalf("expected 2 updates, got %d", len(repo.updatedIDs))
	}
}

func TestEmbedBatchUseCase_SkipsExistingWhenNotForced(t *testing.T) {
	repo := &mockCardRepo{cards: []*domain.Card{
		{ID: "c1", Name: "Card A", EmbeddingVector: []float64{0.9}},
		{ID: "c2", Name: "Card B", OracleText: "Draw a card"},
	}}
	uc := usecase.NewEmbedBatchUseCase(repo, &mockEmbedder{}, 2)

	res, err := uc.Execute(context.Background(), usecase.EmbedBatchRequest{Limit: 10, Force: false})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.Skipped != 1 || res.Updated != 1 {
		t.Fatalf("unexpected result: %+v", res)
	}
}

func TestEmbedBatchUseCase_ReportsFailures(t *testing.T) {
	repo := &mockCardRepo{cards: []*domain.Card{
		{ID: "ok", Name: "Card OK", OracleText: "Ramp"},
		{ID: "bad", Name: "Card Bad", OracleText: "Fail me"},
	}}
	embedder := &mockEmbedder{failFor: "Fail me"}
	uc := usecase.NewEmbedBatchUseCase(repo, embedder, 2)

	res, err := uc.Execute(context.Background(), usecase.EmbedBatchRequest{Limit: 10, Force: true})
	if err != nil {
		t.Fatalf("unexpected top-level error: %v", err)
	}
	if res.Failed != 1 || res.Updated != 1 {
		t.Fatalf("unexpected result: %+v", res)
	}
}

func TestEmbedBatchUseCase_NoProvider(t *testing.T) {
	repo := &mockCardRepo{cards: []*domain.Card{{ID: "c1", Name: "Card"}}}
	uc := usecase.NewEmbedBatchUseCase(repo, nil, 1)
	_, err := uc.Execute(context.Background(), usecase.EmbedBatchRequest{Limit: 1})
	if err == nil {
		t.Fatal("expected error when embedder is nil")
	}
}
