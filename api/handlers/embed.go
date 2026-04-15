package handlers

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/gigliofr/mana-wise/usecase"
)

// EmbedBatchHandler handles POST /embed/batch.
type EmbedBatchHandler struct {
	uc *usecase.EmbedBatchUseCase
}

// NewEmbedBatchHandler creates a new EmbedBatchHandler.
func NewEmbedBatchHandler(uc *usecase.EmbedBatchUseCase) *EmbedBatchHandler {
	return &EmbedBatchHandler{uc: uc}
}

// ServeHTTP runs the embedding pipeline on a batch of cards.
func (h *EmbedBatchHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.uc == nil {
		jsonError(w, "embedding pipeline is not configured", http.StatusServiceUnavailable)
		return
	}

	var req usecase.EmbedBatchRequest
	if r.Body != nil {
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil && !errors.Is(err, io.EOF) {
			jsonError(w, "invalid request body", http.StatusBadRequest)
			return
		}
	}

	result, err := h.uc.Execute(r.Context(), req)
	if err != nil {
		jsonError(w, err.Error(), http.StatusInternalServerError)
		return
	}
	jsonOK(w, result)
}
