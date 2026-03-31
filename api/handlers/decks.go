package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/manawise/api/api/middleware"
	"github.com/manawise/api/domain"
	"github.com/manawise/api/usecase"
)

// DeckHandler handles CRUD operations for saved decks.
type DeckHandler struct {
	repo     domain.DeckRepository
	userRepo domain.UserRepository
	cardRepo domain.CardRepository
	analyze  *usecase.AnalyzeDeckUseCase
	classify *usecase.DeckClassifierUseCase
}

// NewDeckHandler creates a DeckHandler.
func NewDeckHandler(repo domain.DeckRepository, userRepo domain.UserRepository, cardRepo domain.CardRepository, analyze *usecase.AnalyzeDeckUseCase, classify *usecase.DeckClassifierUseCase) *DeckHandler {
	return &DeckHandler{repo: repo, userRepo: userRepo, cardRepo: cardRepo, analyze: analyze, classify: classify}
}

type deckLegalityResponse struct {
	DeckID       string                               `json:"deck_id"`
	Formats      map[string]usecase.DeckLegalityResult `json:"formats"`
	CheckedAtUTC string                               `json:"checked_at"`
}

type deckAnalysisResponse struct {
	DeckID        string                                `json:"deck_id"`
	Deterministic domain.AnalysisResult                 `json:"deterministic"`
	Fingerprint   *usecase.DeckClassifyResult           `json:"fingerprint,omitempty"`
	Legality      map[string]usecase.DeckLegalityResult `json:"legality"`
	CheckedAtUTC  string                                `json:"checked_at"`
}

// saveDeckRequest is the JSON body for deck save/update.
type saveDeckRequest struct {
	Name        string            `json:"name"`
	Format      string            `json:"format"`
	Cards       []domain.DeckCard `json:"cards"`
	Description string            `json:"description,omitempty"`
	IsPublic    bool              `json:"is_public"`
}

// List handles GET /api/v1/decks — returns all decks owned by the authenticated user.
func (h *DeckHandler) List(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	decks, err := h.repo.FindByUserID(r.Context(), userID)
	if err != nil {
		jsonError(w, "failed to retrieve decks", http.StatusInternalServerError)
		return
	}

	jsonOK(w, decks)
}

// Get handles GET /api/v1/decks/{id}.
func (h *DeckHandler) Get(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	id := chi.URLParam(r, "id")

	deck, err := h.repo.FindByID(r.Context(), id)
	if err != nil {
		jsonError(w, "failed to retrieve deck", http.StatusInternalServerError)
		return
	}
	if deck == nil {
		jsonError(w, "deck not found", http.StatusNotFound)
		return
	}
	// Only the owner (or a public deck) can be viewed.
	if deck.UserID != userID && !deck.IsPublic {
		jsonError(w, "deck not found", http.StatusNotFound)
		return
	}

	jsonOK(w, deck)
}

// Legality handles GET /api/v1/decks/{id}/legality.
func (h *DeckHandler) Legality(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	id := chi.URLParam(r, "id")

	if h.cardRepo == nil {
		jsonError(w, "card repository unavailable", http.StatusServiceUnavailable)
		return
	}

	deck, err := h.repo.FindByID(r.Context(), id)
	if err != nil {
		jsonError(w, "failed to retrieve deck", http.StatusInternalServerError)
		return
	}
	if deck == nil {
		jsonError(w, "deck not found", http.StatusNotFound)
		return
	}
	if deck.UserID != userID && !deck.IsPublic {
		jsonError(w, "deck not found", http.StatusNotFound)
		return
	}

	cards, quantities, err := h.resolveDeckCards(r, deck)
	if err != nil {
		jsonError(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	jsonOK(w, deckLegalityResponse{
		DeckID:       deck.ID,
		Formats:      usecase.DetermineDeckLegalityAllFormats(cards, quantities),
		CheckedAtUTC: time.Now().UTC().Format(time.RFC3339),
	})
}

// Analysis handles GET /api/v1/decks/{id}/analysis.
func (h *DeckHandler) Analysis(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	id := chi.URLParam(r, "id")

	if h.analyze == nil {
		jsonError(w, "analysis use case unavailable", http.StatusServiceUnavailable)
		return
	}

	deck, err := h.repo.FindByID(r.Context(), id)
	if err != nil {
		jsonError(w, "failed to retrieve deck", http.StatusInternalServerError)
		return
	}
	if deck == nil {
		jsonError(w, "deck not found", http.StatusNotFound)
		return
	}
	if deck.UserID != userID && !deck.IsPublic {
		jsonError(w, "deck not found", http.StatusNotFound)
		return
	}

	decklist := deckToDecklist(deck)
	if strings.TrimSpace(decklist) == "" {
		jsonError(w, "deck has no mainboard cards", http.StatusUnprocessableEntity)
		return
	}

	analysisResult, err := h.analyze.Execute(r.Context(), usecase.AnalyzeDeckRequest{
		Decklist: decklist,
		Format:   deck.Format,
	})
	if err != nil {
		jsonError(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	var fingerprint *usecase.DeckClassifyResult
	if h.classify != nil {
		if fp, fpErr := h.classify.Execute(r.Context(), usecase.DeckClassifyRequest{Decklist: decklist, Format: deck.Format}); fpErr == nil {
			fingerprint = &fp
		}
	}

	legality := usecase.DetermineDeckLegalityAllFormats(analysisResult.RawCards, analysisResult.Quantities)

	jsonOK(w, deckAnalysisResponse{
		DeckID:        deck.ID,
		Deterministic: analysisResult.Result,
		Fingerprint:   fingerprint,
		Legality:      legality,
		CheckedAtUTC:  time.Now().UTC().Format(time.RFC3339),
	})
}

func (h *DeckHandler) resolveDeckCards(r *http.Request, deck *domain.Deck) ([]*domain.Card, map[string]int, error) {
	mainCards := deck.MainboardCards()
	cards := make([]*domain.Card, 0, len(mainCards))
	quantities := make(map[string]int, len(mainCards))
	seen := make(map[string]bool, len(mainCards))

	for _, entry := range mainCards {
		if entry.Quantity <= 0 {
			continue
		}

		var card *domain.Card
		var err error
		if strings.TrimSpace(entry.CardID) != "" {
			card, err = h.cardRepo.FindByID(r.Context(), entry.CardID)
			if err != nil {
				return nil, nil, err
			}
		}
		if card == nil {
			card, err = h.cardRepo.FindByName(r.Context(), entry.CardName)
			if err != nil {
				return nil, nil, err
			}
		}
		if card == nil {
			return nil, nil, &deckCardResolveError{name: entry.CardName}
		}

		if !seen[card.ID] {
			cards = append(cards, card)
			seen[card.ID] = true
		}
		quantities[card.ID] += entry.Quantity
	}

	return cards, quantities, nil
}

type deckCardResolveError struct {
	name string
}

func (e *deckCardResolveError) Error() string {
	return "card not found in catalog: " + strings.TrimSpace(e.name)
}

func deckToDecklist(deck *domain.Deck) string {
	if deck == nil {
		return ""
	}
	var b strings.Builder
	for _, c := range deck.MainboardCards() {
		if c.Quantity <= 0 {
			continue
		}
		name := strings.TrimSpace(c.CardName)
		if name == "" {
			continue
		}
		b.WriteString(strconv.Itoa(c.Quantity))
		b.WriteString(" ")
		b.WriteString(name)
		b.WriteString("\n")
	}
	return strings.TrimSpace(b.String())
}

// Create handles POST /api/v1/decks.
func (h *DeckHandler) Create(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	if userID == "" {
		jsonError(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	plan := strings.ToLower(strings.TrimSpace(middleware.PlanFromContext(r.Context())))
	if h.userRepo != nil {
		if u, err := h.userRepo.FindByID(r.Context(), userID); err == nil && u != nil {
			plan = strings.ToLower(strings.TrimSpace(string(u.Plan)))
		}
	}
	if plan == "" {
		plan = "free"
	}
	if plan != "pro" {
		decks, err := h.repo.FindByUserID(r.Context(), userID)
		if err != nil {
			jsonError(w, "failed to check deck limit", http.StatusInternalServerError)
			return
		}
		if len(decks) >= 1 {
			jsonError(w, "free plan allows only 1 saved deck. upgrade to pro for unlimited decks", http.StatusForbidden)
			return
		}
	}

	var req saveDeckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	req.Format = domain.NormalizeFormat(strings.TrimSpace(req.Format))
	if req.Name == "" {
		jsonError(w, "name is required", http.StatusBadRequest)
		return
	}
	if !domain.IsValidFormat(req.Format) {
		jsonError(w, "unsupported format", http.StatusBadRequest)
		return
	}

	now := time.Now().UTC()
	deck := &domain.Deck{
		ID:          uuid.New().String(),
		UserID:      userID,
		Name:        req.Name,
		Format:      req.Format,
		Cards:       req.Cards,
		Description: req.Description,
		IsPublic:    req.IsPublic,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := h.repo.Create(r.Context(), deck); err != nil {
		jsonError(w, "failed to save deck", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusCreated)
	jsonOK(w, deck)
}

// Update handles PUT /api/v1/decks/{id}.
func (h *DeckHandler) Update(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	id := chi.URLParam(r, "id")

	existing, err := h.repo.FindByID(r.Context(), id)
	if err != nil {
		jsonError(w, "failed to retrieve deck", http.StatusInternalServerError)
		return
	}
	if existing == nil || existing.UserID != userID {
		jsonError(w, "deck not found", http.StatusNotFound)
		return
	}

	var req saveDeckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	req.Name = strings.TrimSpace(req.Name)
	req.Format = domain.NormalizeFormat(strings.TrimSpace(req.Format))
	if req.Name == "" {
		jsonError(w, "name is required", http.StatusBadRequest)
		return
	}
	if !domain.IsValidFormat(req.Format) {
		jsonError(w, "unsupported format", http.StatusBadRequest)
		return
	}

	existing.Name = req.Name
	existing.Format = req.Format
	existing.Cards = req.Cards
	existing.Description = req.Description
	existing.IsPublic = req.IsPublic
	existing.UpdatedAt = time.Now().UTC()

	if err := h.repo.Update(r.Context(), existing); err != nil {
		jsonError(w, "failed to update deck", http.StatusInternalServerError)
		return
	}

	jsonOK(w, existing)
}

// Delete handles DELETE /api/v1/decks/{id}.
func (h *DeckHandler) Delete(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	id := chi.URLParam(r, "id")

	existing, err := h.repo.FindByID(r.Context(), id)
	if err != nil {
		jsonError(w, "failed to retrieve deck", http.StatusInternalServerError)
		return
	}
	if existing == nil || existing.UserID != userID {
		jsonError(w, "deck not found", http.StatusNotFound)
		return
	}

	if err := h.repo.Delete(r.Context(), id); err != nil {
		jsonError(w, "failed to delete deck", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
