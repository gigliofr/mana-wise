package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/manawise/api/usecase"
)

// SideboardCoachRequest is the payload for matchup sideboard planning.
type SideboardCoachRequest struct {
	MainDecklist      string `json:"main_decklist"`
	SideboardDecklist string `json:"sideboard_decklist"`
	OpponentArchetype string `json:"opponent_archetype"`
	Format            string `json:"format,omitempty"`
}

// SideboardCoachHandler serves sideboard planning requests.
type SideboardCoachHandler struct {
	uc *usecase.SideboardCoachUseCase
}

// NewSideboardCoachHandler creates a sideboard handler.
func NewSideboardCoachHandler(uc *usecase.SideboardCoachUseCase) *SideboardCoachHandler {
	return &SideboardCoachHandler{uc: uc}
}

// ServeHTTP handles POST /sideboard/plan.
func (h *SideboardCoachHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.uc == nil {
		jsonError(w, "sideboard coach unavailable", http.StatusServiceUnavailable)
		return
	}

	var req SideboardCoachRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	req.MainDecklist = strings.TrimSpace(req.MainDecklist)
	req.SideboardDecklist = strings.TrimSpace(req.SideboardDecklist)
	req.OpponentArchetype = strings.TrimSpace(req.OpponentArchetype)

	if req.MainDecklist == "" {
		jsonError(w, "main_decklist is required", http.StatusBadRequest)
		return
	}
	if req.SideboardDecklist == "" {
		jsonError(w, "sideboard_decklist is required", http.StatusBadRequest)
		return
	}
	if req.OpponentArchetype == "" {
		jsonError(w, "opponent_archetype is required", http.StatusBadRequest)
		return
	}

	plan, err := h.uc.Execute(r.Context(), usecase.SideboardPlanRequest{
		MainDecklist:      req.MainDecklist,
		SideboardDecklist: req.SideboardDecklist,
		OpponentArchetype: req.OpponentArchetype,
		Format:            req.Format,
	})
	if err != nil {
		jsonError(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	jsonOK(w, plan)
}
