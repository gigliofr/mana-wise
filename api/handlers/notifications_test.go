package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/manawise/api/api/middleware"
	"github.com/manawise/api/domain"
)

type notifMockDeckRepo struct {
	decks []*domain.Deck
}

func (r *notifMockDeckRepo) FindByID(ctx context.Context, id string) (*domain.Deck, error) {
	for _, d := range r.decks {
		if d != nil && d.ID == id {
			return d, nil
		}
	}
	return nil, nil
}
func (r *notifMockDeckRepo) FindByUserID(ctx context.Context, userID string) ([]*domain.Deck, error) {
	out := make([]*domain.Deck, 0)
	for _, d := range r.decks {
		if d != nil && d.UserID == userID {
			out = append(out, d)
		}
	}
	return out, nil
}
func (r *notifMockDeckRepo) Create(ctx context.Context, deck *domain.Deck) error { return nil }
func (r *notifMockDeckRepo) Update(ctx context.Context, deck *domain.Deck) error { return nil }
func (r *notifMockDeckRepo) Delete(ctx context.Context, id string) error          { return nil }

func runNotificationsFeedRequest(t *testing.T, h *NotificationHandler, path string, withAuth bool) *httptest.ResponseRecorder {
	t.Helper()
	r := chi.NewRouter()
	if withAuth {
		r.Use(func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
				ctx := context.WithValue(req.Context(), middleware.ContextKeyUserID, "u-1")
				next.ServeHTTP(w, req.WithContext(ctx))
			})
		})
	}
	r.Get("/api/v1/users/me/notifications", h.Feed)

	req := httptest.NewRequest(http.MethodGet, path, nil)
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	return rr
}

func runWebhookRequest(t *testing.T, h *NotificationHandler, body string) *httptest.ResponseRecorder {
	t.Helper()
	r := chi.NewRouter()
	r.Post("/api/v1/webhooks/scryfall", h.IngestScryfallWebhook)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/webhooks/scryfall", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	return rr
}

func TestNotificationsFeed_Unauthorized(t *testing.T) {
	h := NewNotificationHandler(&notifMockDeckRepo{}, &legalityMockCardRepo{byName: map[string]*domain.Card{}})
	rr := runNotificationsFeedRequest(t, h, "/api/v1/users/me/notifications", false)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rr.Code)
	}
}

func TestNotificationsFeed_IllegalCardDetected(t *testing.T) {
	now := time.Now().UTC()
	deck := &domain.Deck{
		ID:     "d-n1",
		UserID: "u-1",
		Format: "modern",
		Cards:  []domain.DeckCard{{CardID: "c-fury", CardName: "Fury", Quantity: 2}},
	}
	cardRepo := &legalityMockCardRepo{
		byID: map[string]*domain.Card{
			"c-fury": {
				ID:         "c-fury",
				Name:       "Fury",
				Legalities: map[string]string{"modern": "banned"},
				UpdatedAt:  now,
			},
		},
		byName: map[string]*domain.Card{
			"Fury": {ID: "c-fury", Name: "Fury", Legalities: map[string]string{"modern": "banned"}, UpdatedAt: now},
		},
	}
	h := NewNotificationHandler(&notifMockDeckRepo{decks: []*domain.Deck{deck}}, cardRepo)

	rr := runNotificationsFeedRequest(t, h, "/api/v1/users/me/notifications", true)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}

	var resp notificationsFeedResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(resp.Items) == 0 {
		t.Fatalf("expected at least one notification item")
	}
	if resp.Items[0].Type != "banlist" {
		t.Fatalf("expected type banlist, got %s", resp.Items[0].Type)
	}
}

func TestWebhookIngestAndFeed_MatchedEvent(t *testing.T) {
	now := time.Now().UTC()
	deck := &domain.Deck{ID: "d-n2", UserID: "u-1", Format: "modern", Cards: []domain.DeckCard{{CardID: "c-rag", CardName: "Ragavan, Nimble Pilferer", Quantity: 1}}}
	cardRepo := &legalityMockCardRepo{
		byID: map[string]*domain.Card{"c-rag": {ID: "c-rag", Name: "Ragavan, Nimble Pilferer", Legalities: map[string]string{"modern": "legal"}, UpdatedAt: now}},
		byName: map[string]*domain.Card{"Ragavan, Nimble Pilferer": {ID: "c-rag", Name: "Ragavan, Nimble Pilferer", Legalities: map[string]string{"modern": "legal"}, UpdatedAt: now}},
	}
	h := NewNotificationHandler(&notifMockDeckRepo{decks: []*domain.Deck{deck}}, cardRepo)

	body := `{"events":[{"card":"Ragavan, Nimble Pilferer","format":"modern","action":"banned","message":"Card banned in Modern","replacement_suggestion":"Monastery Swiftspear"}]}`
	rw := runWebhookRequest(t, h, body)
	if rw.Code != http.StatusOK {
		t.Fatalf("expected 200 from webhook, got %d body=%s", rw.Code, rw.Body.String())
	}

	rr := runNotificationsFeedRequest(t, h, "/api/v1/users/me/notifications", true)
	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
	var resp notificationsFeedResponse
	_ = json.Unmarshal(rr.Body.Bytes(), &resp)
	if len(resp.Items) == 0 {
		t.Fatalf("expected notifications from webhook event")
	}
	found := false
	for _, it := range resp.Items {
		if strings.Contains(strings.ToLower(it.Message), "banned in modern") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected matched webhook message in notifications feed")
	}
}

func TestWebhookIngest_BadBody(t *testing.T) {
	h := NewNotificationHandler(&notifMockDeckRepo{}, &legalityMockCardRepo{byName: map[string]*domain.Card{}})
	rr := runWebhookRequest(t, h, `{`)
	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rr.Code)
	}
}
