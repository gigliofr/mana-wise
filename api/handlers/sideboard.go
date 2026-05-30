package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gigliofr/mana-wise/usecase"
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
		WriteAPIErrorFromMsg(w, "sideboard coach unavailable", http.StatusServiceUnavailable)
		return
	}

	var req SideboardCoachRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteAPIErrorFromMsg(w, "invalid request body", http.StatusBadRequest)
		return
	}

	req.MainDecklist = strings.TrimSpace(req.MainDecklist)
	req.SideboardDecklist = strings.TrimSpace(req.SideboardDecklist)
	req.OpponentArchetype = normalizeArchetypeInput(req.OpponentArchetype)

	if req.MainDecklist == "" {
		WriteAPIErrorFromMsg(w, "main_decklist is required", http.StatusBadRequest)
		return
	}
	if req.SideboardDecklist == "" {
		WriteAPIErrorFromMsg(w, "sideboard_decklist is required", http.StatusBadRequest)
		return
	}
	if req.OpponentArchetype == "" {
		WriteAPIErrorFromMsg(w, "opponent_archetype is required", http.StatusBadRequest)
		return
	}
	if !isValidSideboardOpponentArchetype(req.OpponentArchetype) {
		WriteAPIErrorFromMsg(w, "invalid opponent_archetype: supported values are aggro, midrange, control, combo, ramp, graveyard, artifacts, enchantments", http.StatusBadRequest)
		return
	}

	plan, err := h.uc.Execute(r.Context(), usecase.SideboardPlanRequest{
		MainDecklist:      req.MainDecklist,
		SideboardDecklist: req.SideboardDecklist,
		OpponentArchetype: req.OpponentArchetype,
		Format:            req.Format,
	})
	if err != nil {
		WriteAPIErrorFromMsg(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	jsonOK(w, plan)
}
