package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gigliofr/mana-wise/usecase"
)

// MatchupSimulationRequest is the API payload for matchup simulation.
type MatchupSimulationRequest struct {
	Decklist          string             `json:"decklist"`
	SideboardDecklist string             `json:"sideboard_decklist,omitempty"`
	Format            string             `json:"format,omitempty"`
	PlayerArchetype   string             `json:"player_archetype,omitempty"`
	Opponents         []string           `json:"opponents,omitempty"`
	OnPlay            bool               `json:"on_play,omitempty"`
	MetaShares        map[string]float64 `json:"meta_shares,omitempty"`
}

// MatchupHandler serves matchup simulation requests.
type MatchupHandler struct {
	uc *usecase.MatchupSimulatorUseCase
}

// NewMatchupHandler creates a new matchup handler.
func NewMatchupHandler(uc *usecase.MatchupSimulatorUseCase) *MatchupHandler {
	return &MatchupHandler{uc: uc}
}

// ServeHTTP handles POST /matchup/simulate.
func (h *MatchupHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if h.uc == nil {
		jsonError(w, "matchup simulator unavailable", http.StatusServiceUnavailable)
		return
	}

	var req MatchupSimulationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	req.Decklist = strings.TrimSpace(req.Decklist)
	if req.Decklist == "" {
		jsonError(w, "decklist is required", http.StatusBadRequest)
		return
	}

	req.PlayerArchetype = normalizeArchetypeInput(req.PlayerArchetype)
	if req.PlayerArchetype != "" && !isPlayableArchetype(req.PlayerArchetype) {
		jsonError(w, "invalid player_archetype: supported values are aggro, midrange, control, combo, ramp", http.StatusBadRequest)
		return
	}

	req.Opponents = normalizeArchetypeList(req.Opponents)
	for _, opponent := range req.Opponents {
		if !isPlayableArchetype(opponent) {
			jsonError(w, "invalid opponent archetype: supported values are aggro, midrange, control, combo, ramp", http.StatusBadRequest)
			return
		}
	}

	res, err := h.uc.Execute(r.Context(), usecase.MatchupSimulationRequest{
		Decklist:          req.Decklist,
		SideboardDecklist: strings.TrimSpace(req.SideboardDecklist),
		Format:            req.Format,
		PlayerArchetype:   req.PlayerArchetype,
		Opponents:         req.Opponents,
		OnPlay:            req.OnPlay,
		MetaShares:        req.MetaShares,
	})
	if err != nil {
		jsonError(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	jsonOK(w, res)
}
