package handlers

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/gigliofr/mana-wise/api/middleware"
	"github.com/gigliofr/mana-wise/domain"
)

type notificationItem struct {
	Type                  string `json:"type"`
	DeckID                string `json:"deck_id,omitempty"`
	Card                  string `json:"card"`
	Message               string `json:"message"`
	ReplacementSuggestion string `json:"replacement_suggestion,omitempty"`
	CreatedAt             string `json:"created_at"`
}

type notificationsFeedResponse struct {
	Items []notificationItem `json:"items"`
}

type webhookBanEvent struct {
	Card                  string `json:"card"`
	Format                string `json:"format,omitempty"`
	Action                string `json:"action,omitempty"` // banned|unbanned|rotation
	Message               string `json:"message,omitempty"`
	ReplacementSuggestion string `json:"replacement_suggestion,omitempty"`
	CreatedAt             string `json:"created_at,omitempty"`
}

type webhookScryfallRequest struct {
	Events []webhookBanEvent `json:"events,omitempty"`
	Card   string            `json:"card,omitempty"`
	Format string            `json:"format,omitempty"`
	Action string            `json:"action,omitempty"`
	Message string           `json:"message,omitempty"`
	ReplacementSuggestion string `json:"replacement_suggestion,omitempty"`
}

type webhookScryfallResponse struct {
	Accepted int    `json:"accepted"`
	Status   string `json:"status"`
}

type NotificationHandler struct {
	deckRepo domain.DeckRepository
	cardRepo domain.CardRepository

	mu         sync.RWMutex
	webhookLog []webhookBanEvent
}

func NewNotificationHandler(deckRepo domain.DeckRepository, cardRepo domain.CardRepository) *NotificationHandler {
	return &NotificationHandler{deckRepo: deckRepo, cardRepo: cardRepo, webhookLog: make([]webhookBanEvent, 0, 64)}
}

// IngestScryfallWebhook handles POST /api/v1/webhooks/scryfall.
func (h *NotificationHandler) IngestScryfallWebhook(w http.ResponseWriter, r *http.Request) {
	var req webhookScryfallRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	events := req.Events
	if len(events) == 0 && strings.TrimSpace(req.Card) != "" {
		events = []webhookBanEvent{{
			Card:                  req.Card,
			Format:                req.Format,
			Action:                req.Action,
			Message:               req.Message,
			ReplacementSuggestion: req.ReplacementSuggestion,
		}}
	}

	now := time.Now().UTC().Format(time.RFC3339)
	accepted := 0

	h.mu.Lock()
	defer h.mu.Unlock()

	for _, ev := range events {
		card := strings.TrimSpace(ev.Card)
		if card == "" {
			continue
		}
		if strings.TrimSpace(ev.CreatedAt) == "" {
			ev.CreatedAt = now
		}
		ev.Card = card
		ev.Format = domain.NormalizeFormat(strings.TrimSpace(ev.Format))
		ev.Action = strings.ToLower(strings.TrimSpace(ev.Action))
		if ev.Action == "" {
			ev.Action = "banned"
		}
		if strings.TrimSpace(ev.Message) == "" {
			if ev.Format != "" {
				ev.Message = "Card " + ev.Card + " update in " + strings.Title(ev.Format)
			} else {
				ev.Message = "Card " + ev.Card + " update"
			}
		}
		h.webhookLog = append(h.webhookLog, ev)
		accepted++
	}

	const maxEvents = 500
	if len(h.webhookLog) > maxEvents {
		h.webhookLog = h.webhookLog[len(h.webhookLog)-maxEvents:]
	}

	jsonOK(w, webhookScryfallResponse{Accepted: accepted, Status: "accepted"})
}

// Feed handles GET /api/v1/users/me/notifications.
func (h *NotificationHandler) Feed(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}
	if h.deckRepo == nil || h.cardRepo == nil {
		jsonError(w, "notification dependencies unavailable", http.StatusServiceUnavailable)
		return
	}

	decks, err := h.deckRepo.FindByUserID(r.Context(), userID)
	if err != nil {
		jsonError(w, "failed to retrieve decks", http.StatusInternalServerError)
		return
	}

	h.mu.RLock()
	logSnapshot := append([]webhookBanEvent(nil), h.webhookLog...)
	h.mu.RUnlock()

	items := make([]notificationItem, 0)
	seen := map[string]bool{}

	for _, deck := range decks {
		if deck == nil {
			continue
		}
		format := domain.NormalizeFormat(deck.Format)
		for _, entry := range deck.Cards {
			cardName := strings.TrimSpace(entry.CardName)
			if cardName == "" {
				continue
			}

			card, resolveErr := h.resolveNotificationCardEntry(r, entry)
			if resolveErr != nil || card == nil {
				continue
			}
			resolvedName := card.Name
			if strings.TrimSpace(resolvedName) == "" {
				resolvedName = cardName
			}
			if !card.IsLegal(format) {
				msg := "Card banned or not legal in " + strings.Title(format)
				key := strings.ToLower(deck.ID + "|" + resolvedName + "|" + msg)
				if !seen[key] {
					items = append(items, notificationItem{
						Type:                  "banlist",
						DeckID:                deck.ID,
						Card:                  resolvedName,
						Message:               msg,
						ReplacementSuggestion: suggestReplacement(resolvedName),
						CreatedAt:             time.Now().UTC().Format(time.RFC3339),
					})
					seen[key] = true
				}
			}

			for i := len(logSnapshot) - 1; i >= 0; i-- {
				ev := logSnapshot[i]
				if !strings.EqualFold(strings.TrimSpace(ev.Card), resolvedName) {
					continue
				}
				if ev.Format != "" && ev.Format != format {
					continue
				}
				msg := strings.TrimSpace(ev.Message)
				if msg == "" {
					msg = "Card update received"
				}
				key := strings.ToLower(deck.ID + "|" + resolvedName + "|" + msg)
				if seen[key] {
					continue
				}
				items = append(items, notificationItem{
					Type:                  "banlist",
					DeckID:                deck.ID,
					Card:                  resolvedName,
					Message:               msg,
					ReplacementSuggestion: strings.TrimSpace(ev.ReplacementSuggestion),
					CreatedAt:             normalizeCreatedAt(ev.CreatedAt),
				})
				seen[key] = true
				break
			}
		}
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].CreatedAt > items[j].CreatedAt
	})

	jsonOK(w, notificationsFeedResponse{Items: items})
}

func suggestReplacement(cardName string) string {
	low := strings.ToLower(strings.TrimSpace(cardName))
	switch {
	case strings.Contains(low, "fury"):
		return "Consider Unholy Heat or Lightning Bolt package adjustments."
	case strings.Contains(low, "ragavan"):
		return "Consider Monastery Swiftspear for low-curve pressure."
	case strings.Contains(low, "expressive iteration"):
		return "Consider Preordain or Consider for card selection."
	default:
		return "Consider a legal alternative with similar role and mana curve."
	}
}

func normalizeCreatedAt(v string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return time.Now().UTC().Format(time.RFC3339)
	}
	if _, err := time.Parse(time.RFC3339, v); err != nil {
		return time.Now().UTC().Format(time.RFC3339)
	}
	return v
}

func (h *NotificationHandler) resolveNotificationCardEntry(r *http.Request, entry domain.DeckCard) (*domain.Card, error) {
	if strings.TrimSpace(entry.CardID) != "" {
		card, err := h.cardRepo.FindByID(r.Context(), entry.CardID)
		if err != nil {
			return nil, err
		}
		if card != nil {
			return card, nil
		}
	}
	return h.cardRepo.FindByName(r.Context(), entry.CardName)
}
