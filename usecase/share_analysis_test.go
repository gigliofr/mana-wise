package usecase

import (
    "context"
    "testing"
    "time"

    "github.com/gigliofr/mana-wise/domain"
)

type memSharedRepo struct {
    store map[string]*domain.SharedAnalysisLink
}

func newMemSharedRepo() *memSharedRepo {
    return &memSharedRepo{store: make(map[string]*domain.SharedAnalysisLink)}
}

func (m *memSharedRepo) Create(ctx context.Context, link *domain.SharedAnalysisLink) error {
    m.store[link.ID] = link
    return nil
}
func (m *memSharedRepo) FindByID(ctx context.Context, id string) (*domain.SharedAnalysisLink, error) {
    if l, ok := m.store[id]; ok {
        return l, nil
    }
    return nil, nil
}
func (m *memSharedRepo) Delete(ctx context.Context, id string) error { delete(m.store, id); return nil }
func (m *memSharedRepo) DeleteExpired(ctx context.Context, now time.Time) (int64, error) { return 0, nil }
func (m *memSharedRepo) IncrementVisit(ctx context.Context, id string, at time.Time) error { return nil }

func TestShareAnalysisCreatesLink(t *testing.T) {
    repo := newMemSharedRepo()
    req := ShareAnalysisRequest{
        DeckID:  "deck123",
        Channel: "link",
        TTL:     2 * time.Hour,
    }
    baseURL := "https://example.test"
    resp, err := ShareAnalysis(context.Background(), repo, req, baseURL)
    if err != nil {
        t.Fatalf("ShareAnalysis returned error: %v", err)
    }
    if resp == nil || resp.ShareURL == "" {
        t.Fatalf("expected non-empty share url, got %#v", resp)
    }
    // check repo stored an entry
    found := false
    for _, v := range repo.store {
        if v.DeckID == req.DeckID && v.Channel == req.Channel {
            found = true
            break
        }
    }
    if !found {
        t.Fatalf("expected repo to contain stored link for deck %s", req.DeckID)
    }
}
