package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gigliofr/mana-wise/usecase"
)

// DeckClassifyRequest is the API payload for deck classification.
type DeckClassifyRequest struct {
	Decklist string `json:"decklist"`
	Format   string `json:"format,omitempty"`
}

// DeckClassifyHandler serves deck classification requests.
type DeckClassifyHandler struct {
	uc *usecase.DeckClassifierUseCase
}

// NewDeckClassifyHandler creates a new deck classify handler.
func NewDeckClassifyHandler(uc *usecase.DeckClassifierUseCase) *DeckClassifyHandler {
	return &DeckClassifyHandler{uc: uc}
}

// ServeHTTP handles POST /deck/classify.
func (h *DeckClassifyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.uc == nil {
		jsonError(w, "deck classifier unavailable", http.StatusServiceUnavailable)
		return
	}

	var req DeckClassifyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	req.Decklist = strings.TrimSpace(req.Decklist)
	if req.Decklist == "" {
		jsonError(w, "decklist is required", http.StatusBadRequest)
		return
	}

	res, err := h.uc.Execute(r.Context(), usecase.DeckClassifyRequest{
		Decklist: req.Decklist,
		Format:   req.Format,
	})
	if err != nil {
		jsonError(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	jsonOK(w, res)
}
