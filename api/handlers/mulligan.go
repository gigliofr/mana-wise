package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/manawise/api/usecase"
)

// MulliganRequest is the payload for mulligan simulation.
type MulliganRequest struct {
	Decklist   string `json:"decklist"`
	Format     string `json:"format,omitempty"`
	Archetype  string `json:"archetype,omitempty"`
	Iterations int    `json:"iterations,omitempty"`
	OnPlay     *bool  `json:"on_play,omitempty"`
}

// MulliganHandler serves mulligan simulation.
type MulliganHandler struct {
	uc *usecase.MulliganAssistantUseCase
}

// NewMulliganHandler creates a new mulligan handler.
func NewMulliganHandler(uc *usecase.MulliganAssistantUseCase) *MulliganHandler {
	return &MulliganHandler{uc: uc}
}

// ServeHTTP handles POST /mulligan/simulate.
func (h *MulliganHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.uc == nil {
		jsonError(w, "mulligan assistant unavailable", http.StatusServiceUnavailable)
		return
	}

	var req MulliganRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}
	req.Decklist = strings.TrimSpace(req.Decklist)
	if req.Decklist == "" {
		jsonError(w, "decklist is required", http.StatusBadRequest)
		return
	}

	onPlay := true
	if req.OnPlay != nil {
		onPlay = *req.OnPlay
	}

	res, err := h.uc.Execute(r.Context(), usecase.MulliganSimulationRequest{
		Decklist:   req.Decklist,
		Format:     req.Format,
		Archetype:  req.Archetype,
		Iterations: req.Iterations,
		OnPlay:     onPlay,
	})
	if err != nil {
		jsonError(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	jsonOK(w, res)
}
