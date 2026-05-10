package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gigliofr/mana-wise/domain"
)

type memorySharedAnalysisRepo struct {
	links map[string]*domain.SharedAnalysisLink
}

func newMemorySharedAnalysisRepo() *memorySharedAnalysisRepo {
	return &memorySharedAnalysisRepo{links: make(map[string]*domain.SharedAnalysisLink)}
}

func (r *memorySharedAnalysisRepo) Create(ctx context.Context, link *domain.SharedAnalysisLink) error {
	r.links[link.ID] = link
	return nil
}

func (r *memorySharedAnalysisRepo) FindByID(ctx context.Context, id string) (*domain.SharedAnalysisLink, error) {
	if link, ok := r.links[id]; ok {
		return link, nil
	}
	return nil, nil
}

func (r *memorySharedAnalysisRepo) Delete(ctx context.Context, id string) error { delete(r.links, id); return nil }
func (r *memorySharedAnalysisRepo) DeleteExpired(ctx context.Context, now time.Time) (int64, error) { return 0, nil }
func (r *memorySharedAnalysisRepo) IncrementVisit(ctx context.Context, id string, at time.Time) error { return nil }

func TestShareAnalysisHandler_UsesForwardedHostAndProto(t *testing.T) {
	repo := newMemorySharedAnalysisRepo()
	h := NewShareAnalysisHandler(repo, nil)

	body := `{"deck_id":"deck-123","channel":"link","ttl_hours":24}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/analysis/share", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Forwarded-Proto", "https")
	req.Header.Set("X-Forwarded-Host", "mana-wise.geniuscrafters.it")
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}

	var resp ShareAnalysisAPIResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode response: %v body=%s", err, rr.Body.String())
	}
	const expectedPrefix = "https://mana-wise.geniuscrafters.it/share/"
	if !strings.HasPrefix(resp.ShareURL, expectedPrefix) {
		t.Fatalf("expected forwarded host in share url, got %q", resp.ShareURL)
	}
	if token := strings.TrimPrefix(resp.ShareURL, expectedPrefix); token == "" {
		t.Fatalf("expected non-empty share token in url, got %q", resp.ShareURL)
	}
}
