package handlers

import (
	"encoding/json"
	"math"
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
	mulligan *usecase.MulliganAssistantUseCase
}

// NewDeckHandler creates a DeckHandler.
func NewDeckHandler(repo domain.DeckRepository, userRepo domain.UserRepository, cardRepo domain.CardRepository, analyze *usecase.AnalyzeDeckUseCase, classify *usecase.DeckClassifierUseCase, mulligan *usecase.MulliganAssistantUseCase) *DeckHandler {
	return &DeckHandler{repo: repo, userRepo: userRepo, cardRepo: cardRepo, analyze: analyze, classify: classify, mulligan: mulligan}
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
	Curve         []curveTypeBucket                     `json:"curve,omitempty"`
	Archetype     string                                `json:"archetype,omitempty"`
	Confidence    float64                               `json:"confidence,omitempty"`
	AvgCMC        float64                               `json:"avg_cmc,omitempty"`
	MetaFitScore  int                                   `json:"meta_fit_score,omitempty"`
	DeviationMeta float64                               `json:"deviation_from_meta,omitempty"`
	CheckedAtUTC  string                                `json:"checked_at"`
}

type deckSynergiesResponse struct {
	DeckID       string             `json:"deck_id"`
	Combos       []deckCombo        `json:"combos"`
	SynergyScore int                `json:"synergy_score"`
	Packages     []synergyPackage   `json:"packages"`
	RankingMode  string             `json:"ranking_mode,omitempty"`
	EmbeddingCoverage float64       `json:"embedding_coverage,omitempty"`
	CheckedAtUTC string             `json:"checked_at"`
}

type deckPriceCardLine struct {
	CardID       string  `json:"card_id,omitempty"`
	Name         string  `json:"name"`
	Quantity     int     `json:"quantity"`
	IsSideboard  bool    `json:"is_sideboard"`
	UnitUSD      float64 `json:"unit_usd"`
	UnitEUR      float64 `json:"unit_eur"`
	LineTotalUSD float64 `json:"line_total_usd"`
	LineTotalEUR float64 `json:"line_total_eur"`
	PriceSource  string  `json:"price_source"`
}

type deckPriceResponse struct {
	DeckID            string              `json:"deck_id"`
	TotalUSD          float64             `json:"total_usd"`
	TotalEUR          float64             `json:"total_eur"`
	MainboardTotalUSD float64             `json:"mainboard_total_usd"`
	MainboardTotalEUR float64             `json:"mainboard_total_eur"`
	SideboardTotalUSD float64             `json:"sideboard_total_usd"`
	SideboardTotalEUR float64             `json:"sideboard_total_eur"`
	Cards             []deckPriceCardLine `json:"cards"`
	MissingPrices     []string            `json:"missing_prices,omitempty"`
	CheckedAtUTC      string              `json:"checked_at"`
}

type deckCombo struct {
	Cards       []string `json:"cards"`
	Type        string   `json:"type"`
	Description string   `json:"description"`
	TurnKill    int      `json:"turn_kill"`
	Score       int      `json:"score,omitempty"`
}

type synergyPackage struct {
	Name  string   `json:"name"`
	Cards []string `json:"cards"`
	Score int      `json:"score"`
}

type curveTypeBucket struct {
	CMC          int `json:"cmc"`
	Creatures    int `json:"creatures"`
	Instants     int `json:"instants"`
	Sorceries    int `json:"sorceries"`
	Enchantments int `json:"enchantments"`
	Artifacts    int `json:"artifacts"`
	Planeswalkers int `json:"planeswalkers"`
	Total        int `json:"total"`
}

type deckSimulateRequest struct {
	Simulations int    `json:"simulations,omitempty"`
	Format      string `json:"format,omitempty"`
	Archetype   string `json:"archetype,omitempty"`
	OnPlay      *bool  `json:"on_play,omitempty"`
}

type deckSideboardSuggestRequest struct {
	Format           string `json:"format,omitempty"`
	OpponentArchetype string `json:"opponent_archetype,omitempty"`
	MetaSnapshot     string `json:"meta_snapshot,omitempty"`
}

type sideboardSuggestion struct {
	Card     string   `json:"card"`
	Qty      int      `json:"qty"`
	Reason   string   `json:"reason"`
	Matchups []string `json:"matchups"`
}

type deckSideboardSuggestResponse struct {
	DeckID       string                     `json:"deck_id"`
	Format       string                     `json:"format"`
	MetaSnapshot string                     `json:"meta_snapshot,omitempty"`
	Suggestions  []sideboardSuggestion      `json:"suggestions"`
	TotalCards   int                        `json:"total_cards"`
	GenerationMode string                   `json:"generation_mode"`
	Plan         usecase.SideboardPlanResult `json:"plan"`
	CheckedAtUTC string                     `json:"checked_at"`
}

type deckSimulateResponse struct {
	DeckID          string                      `json:"deck_id"`
	KeepProbability float64                     `json:"keep_probability"`
	AvgLandsT1      float64                     `json:"avg_lands_t1"`
	PTwoLandsT2     float64                     `json:"p_two_lands_t2"`
	POneDrop        float64                     `json:"p_one_drop"`
	CurveOutT4      float64                     `json:"curve_out_t4"`
	Recommendation  string                      `json:"recommendation"`
	Reasoning       usecase.MulliganReasoning   `json:"reasoning"`
	Raw             usecase.MulliganSimulationResult `json:"raw"`
	CheckedAtUTC    string                      `json:"checked_at"`
}

const targetSideboardSize = 15

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

// Price handles GET /api/v1/decks/{id}/price.
func (h *DeckHandler) Price(w http.ResponseWriter, r *http.Request) {
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

	lines := make([]deckPriceCardLine, 0, len(deck.Cards))
	missing := []string{}
	mainUSD := 0.0
	mainEUR := 0.0
	sideUSD := 0.0
	sideEUR := 0.0

	for _, entry := range deck.Cards {
		if strings.TrimSpace(entry.CardName) == "" || entry.Quantity <= 0 {
			continue
		}

		card, resolveErr := h.resolveDeckCardEntry(r, entry)
		if resolveErr != nil {
			jsonError(w, resolveErr.Error(), http.StatusUnprocessableEntity)
			return
		}

		usd, eur, source := extractCardUnitPrices(card)
		lineUSD := round4(usd * float64(entry.Quantity))
		lineEUR := round4(eur * float64(entry.Quantity))

		if usd == 0 && eur == 0 {
			missing = append(missing, strings.TrimSpace(entry.CardName))
		}

		if entry.IsSideboard {
			sideUSD += lineUSD
			sideEUR += lineEUR
		} else {
			mainUSD += lineUSD
			mainEUR += lineEUR
		}

		line := deckPriceCardLine{
			Name:         strings.TrimSpace(entry.CardName),
			Quantity:     entry.Quantity,
			IsSideboard:  entry.IsSideboard,
			UnitUSD:      usd,
			UnitEUR:      eur,
			LineTotalUSD: lineUSD,
			LineTotalEUR: lineEUR,
			PriceSource:  source,
		}
		if card != nil {
			line.CardID = card.ID
			if strings.TrimSpace(card.Name) != "" {
				line.Name = card.Name
			}
		}
		lines = append(lines, line)
	}

	jsonOK(w, deckPriceResponse{
		DeckID:            deck.ID,
		TotalUSD:          round4(mainUSD + sideUSD),
		TotalEUR:          round4(mainEUR + sideEUR),
		MainboardTotalUSD: round4(mainUSD),
		MainboardTotalEUR: round4(mainEUR),
		SideboardTotalUSD: round4(sideUSD),
		SideboardTotalEUR: round4(sideEUR),
		Cards:             lines,
		MissingPrices:     missing,
		CheckedAtUTC:      time.Now().UTC().Format(time.RFC3339),
	})
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
	archetype := ""
	confidence := 0.0
	if h.classify != nil {
		if fp, fpErr := h.classify.Execute(r.Context(), usecase.DeckClassifyRequest{Decklist: decklist, Format: deck.Format}); fpErr == nil {
			fingerprint = &fp
			archetype = fp.Archetype
			confidence = fp.Confidence
		}
	}

	legality := usecase.DetermineDeckLegalityAllFormats(analysisResult.RawCards, analysisResult.Quantities)
	curveBreakdown := buildCurveTypeBreakdown(analysisResult.RawCards, analysisResult.Quantities)
	deviation := deviationFromMeta(archetype, analysisResult.Result.Mana.AverageCMC)
	metaFit := metaFitFromDeviation(deviation)

	jsonOK(w, deckAnalysisResponse{
		DeckID:        deck.ID,
		Deterministic: analysisResult.Result,
		Fingerprint:   fingerprint,
		Legality:      legality,
		Curve:         curveBreakdown,
		Archetype:     archetype,
		Confidence:    confidence,
		AvgCMC:        analysisResult.Result.Mana.AverageCMC,
		MetaFitScore:  metaFit,
		DeviationMeta: deviation,
		CheckedAtUTC:  time.Now().UTC().Format(time.RFC3339),
	})
}

// Synergies handles GET /api/v1/decks/{id}/synergies.
func (h *DeckHandler) Synergies(w http.ResponseWriter, r *http.Request) {
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

	coverage := embeddingCoverage(cards)
	combos := detectDeckCombos(cards, quantities)
	pkgs := detectSynergyPackages(cards, quantities)
	score := computeSynergyScore(combos, pkgs)
	mode := "rule_based_v1"
	if coverage > 0 {
		mode = "hybrid_rule_embedding_v2"
	}

	jsonOK(w, deckSynergiesResponse{
		DeckID:       deck.ID,
		Combos:       combos,
		SynergyScore: score,
		Packages:     pkgs,
		RankingMode:  mode,
		EmbeddingCoverage: coverage,
		CheckedAtUTC: time.Now().UTC().Format(time.RFC3339),
	})
}

// Simulate handles POST /api/v1/decks/{id}/simulate.
func (h *DeckHandler) Simulate(w http.ResponseWriter, r *http.Request) {
	userID := middleware.UserIDFromContext(r.Context())
	id := chi.URLParam(r, "id")

	if h.mulligan == nil {
		jsonError(w, "mulligan assistant unavailable", http.StatusServiceUnavailable)
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

	var req deckSimulateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err.Error() != "EOF" {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	decklist := deckToDecklist(deck)
	if strings.TrimSpace(decklist) == "" {
		jsonError(w, "deck has no mainboard cards", http.StatusUnprocessableEntity)
		return
	}

	format := strings.TrimSpace(req.Format)
	if format == "" {
		format = deck.Format
	}

	onPlay := true
	if req.OnPlay != nil {
		onPlay = *req.OnPlay
	}

	res, err := h.mulligan.Execute(r.Context(), usecase.MulliganSimulationRequest{
		Decklist:   decklist,
		Format:     format,
		Archetype:  req.Archetype,
		Iterations: req.Simulations,
		OnPlay:     onPlay,
	})
	if err != nil {
		jsonError(w, err.Error(), http.StatusUnprocessableEntity)
		return
	}

	jsonOK(w, deckSimulateResponse{
		DeckID:          deck.ID,
		KeepProbability: res.KeepProbability,
		AvgLandsT1:      res.AvgLandsT1,
		PTwoLandsT2:     res.PTwoLandsT2,
		POneDrop:        res.POneDrop,
		CurveOutT4:      res.CurveOutT4,
		Recommendation:  res.Recommendation,
		Reasoning:       res.Reasoning,
		Raw:             res,
		CheckedAtUTC:    time.Now().UTC().Format(time.RFC3339),
	})
}

// SideboardSuggest handles POST /api/v1/decks/{id}/sideboard/suggest.
func (h *DeckHandler) SideboardSuggest(w http.ResponseWriter, r *http.Request) {
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

	var req deckSideboardSuggestRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil && err.Error() != "EOF" {
		jsonError(w, "invalid request body", http.StatusBadRequest)
		return
	}

	req.OpponentArchetype = normalizeArchetypeInput(req.OpponentArchetype)
	if req.OpponentArchetype == "" {
		req.OpponentArchetype = "midrange"
	}
	if !isValidSideboardOpponentArchetype(req.OpponentArchetype) {
		jsonError(w, "invalid opponent_archetype: supported values are aggro, midrange, control, combo, ramp, graveyard, artifacts, enchantments", http.StatusBadRequest)
		return
	}

	mainDecklist := deckToDecklist(deck)
	if strings.TrimSpace(mainDecklist) == "" {
		jsonError(w, "deck has no mainboard cards", http.StatusUnprocessableEntity)
		return
	}

	sideboardDecklist := deckToSideboardDecklist(deck)

	format := strings.TrimSpace(req.Format)
	if format == "" {
		format = deck.Format
	}

	plan := usecase.SideboardPlanResult{Matchup: req.OpponentArchetype}
	generationMode := "meta_fallback"
	if strings.TrimSpace(sideboardDecklist) != "" {
		uc := usecase.NewSideboardCoachUseCase(h.cardRepo)
		if p, err := uc.Execute(r.Context(), usecase.SideboardPlanRequest{
			MainDecklist:      mainDecklist,
			SideboardDecklist: sideboardDecklist,
			OpponentArchetype: req.OpponentArchetype,
			Format:            format,
		}); err == nil {
			plan = p
			generationMode = "hybrid_saved_sideboard_plus_meta"
		} else {
			plan.Notes = append(plan.Notes, "Sideboard plan fallback enabled: using matchup-oriented meta template.")
		}
	} else {
		plan.Notes = append(plan.Notes, "No saved sideboard found: generated full 15-card board from matchup-oriented template.")
	}

	suggestions := make([]sideboardSuggestion, 0, targetSideboardSize)
	slotByCard := map[string]int{}
	for _, in := range plan.Ins {
		if in.Qty <= 0 {
			continue
		}
		slotByCard[strings.ToLower(strings.TrimSpace(in.Card))] = len(suggestions)
		suggestions = append(suggestions, sideboardSuggestion{
			Card:     in.Card,
			Qty:      in.Qty,
			Reason:   in.Reason,
			Matchups: []string{req.OpponentArchetype},
		})
	}

	templates := metaSideboardTemplate(req.OpponentArchetype, strings.TrimSpace(req.MetaSnapshot))
	for _, cand := range templates {
		if cand.Qty <= 0 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(cand.Card))
		if idx, ok := slotByCard[key]; ok {
			suggestions[idx].Qty += cand.Qty
			if strings.TrimSpace(suggestions[idx].Reason) == "" {
				suggestions[idx].Reason = cand.Reason
			}
			continue
		}
		slotByCard[key] = len(suggestions)
		suggestions = append(suggestions, cand)
	}

	totalCards := 0
	for i := range suggestions {
		if suggestions[i].Qty < 0 {
			suggestions[i].Qty = 0
		}
		totalCards += suggestions[i].Qty
	}

	if totalCards > targetSideboardSize {
		extra := totalCards - targetSideboardSize
		for i := len(suggestions) - 1; i >= 0 && extra > 0; i-- {
			take := suggestions[i].Qty
			if take > extra {
				take = extra
			}
			suggestions[i].Qty -= take
			extra -= take
		}
	}

	totalCards = 0
	trimmed := make([]sideboardSuggestion, 0, len(suggestions))
	for _, s := range suggestions {
		if s.Qty <= 0 {
			continue
		}
		trimmed = append(trimmed, s)
		totalCards += s.Qty
	}
	suggestions = trimmed

	jsonOK(w, deckSideboardSuggestResponse{
		DeckID:       deck.ID,
		Format:       format,
		MetaSnapshot: strings.TrimSpace(req.MetaSnapshot),
		Suggestions:  suggestions,
		TotalCards:   totalCards,
		GenerationMode: generationMode,
		Plan:         plan,
		CheckedAtUTC: time.Now().UTC().Format(time.RFC3339),
	})
}

func metaSideboardTemplate(opponentArchetype, metaSnapshot string) []sideboardSuggestion {
	opponent := normalizeArchetypeInput(opponentArchetype)
	base := []sideboardSuggestion{}
	snapshot := strings.ToLower(strings.TrimSpace(metaSnapshot))

	switch opponent {
	case "combo":
		base = []sideboardSuggestion{
			{Card: "Damping Sphere", Qty: 3, Reason: "Slows mana engines and storm turns.", Matchups: []string{"combo", "ramp"}},
			{Card: "Thoughtseize", Qty: 2, Reason: "Pre-emptively strip key combo pieces.", Matchups: []string{"combo", "control"}},
			{Card: "Negate", Qty: 2, Reason: "Protect tempo while interacting on stack.", Matchups: []string{"combo", "control"}},
			{Card: "Unlicensed Hearse", Qty: 2, Reason: "Graveyard denial against recursion loops.", Matchups: []string{"combo", "graveyard"}},
			{Card: "Pithing Needle", Qty: 2, Reason: "Shuts down activated win enablers.", Matchups: []string{"combo", "artifacts"}},
			{Card: "Surgical Extraction", Qty: 2, Reason: "Removes deterministic line pieces from game.", Matchups: []string{"combo", "graveyard"}},
			{Card: "Flusterstorm", Qty: 2, Reason: "Stack-efficient answer in spell-heavy fights.", Matchups: []string{"combo", "control"}},
		}
	case "aggro":
		base = []sideboardSuggestion{
			{Card: "Anger of the Gods", Qty: 2, Reason: "Resets wide boards and exiles recursive threats.", Matchups: []string{"aggro", "graveyard"}},
			{Card: "Temporary Lockdown", Qty: 2, Reason: "Sweeps low-curve permanents efficiently.", Matchups: []string{"aggro", "artifacts", "enchantments"}},
			{Card: "Fatal Push", Qty: 3, Reason: "Cheap interaction for early pressure.", Matchups: []string{"aggro", "midrange"}},
			{Card: "Sunset Revelry", Qty: 2, Reason: "Stabilizes life total and board presence.", Matchups: []string{"aggro"}},
			{Card: "Path to Exile", Qty: 2, Reason: "Efficient creature answer at instant speed.", Matchups: []string{"aggro", "midrange"}},
			{Card: "Engineered Explosives", Qty: 2, Reason: "Flexible sweeper against token and low-CMC swarms.", Matchups: []string{"aggro", "artifacts"}},
			{Card: "Aether Gust", Qty: 2, Reason: "Tempo-positive answer versus red/green threats.", Matchups: []string{"aggro", "ramp"}},
		}
	case "control":
		base = []sideboardSuggestion{
			{Card: "Mystical Dispute", Qty: 3, Reason: "Mana-efficient counter war edge.", Matchups: []string{"control", "combo"}},
			{Card: "Thoughtseize", Qty: 2, Reason: "Disrupts reactive hands before big turns.", Matchups: []string{"control", "combo"}},
			{Card: "Teferi, Time Raveler", Qty: 2, Reason: "Forces sorcery-speed windows.", Matchups: []string{"control"}},
			{Card: "Duress", Qty: 2, Reason: "Low-cost interaction for non-creature spells.", Matchups: []string{"control", "combo"}},
			{Card: "Narset, Parter of Veils", Qty: 2, Reason: "Punishes cantrip/card-draw engines.", Matchups: []string{"control", "combo"}},
			{Card: "Negate", Qty: 2, Reason: "High coverage in non-creature exchanges.", Matchups: []string{"control", "combo"}},
			{Card: "Shark Typhoon", Qty: 2, Reason: "Uncounterable pressure in long games.", Matchups: []string{"control", "midrange"}},
		}
	default:
		base = []sideboardSuggestion{
			{Card: "Unlicensed Hearse", Qty: 2, Reason: "Graveyard interaction with board presence.", Matchups: []string{"graveyard", "midrange"}},
			{Card: "Negate", Qty: 2, Reason: "Stack interaction for spell-based plans.", Matchups: []string{"control", "combo"}},
			{Card: "Damping Sphere", Qty: 2, Reason: "Slows explosive mana curves.", Matchups: []string{"combo", "ramp"}},
			{Card: "Aether Gust", Qty: 2, Reason: "Tempo tool against red/green strategies.", Matchups: []string{"aggro", "ramp"}},
			{Card: "Disdainful Stroke", Qty: 2, Reason: "Catches expensive threats efficiently.", Matchups: []string{"ramp", "control"}},
			{Card: "Path to Exile", Qty: 3, Reason: "Efficient creature answer in broad metas.", Matchups: []string{"aggro", "midrange"}},
			{Card: "Pithing Needle", Qty: 2, Reason: "Versatile answer to planeswalker/engine activations.", Matchups: []string{"control", "artifacts"}},
		}
	}

	if strings.Contains(snapshot, "q1") && len(base) > 0 {
		base[0].Reason = base[0].Reason + " Tuned for early-season volatility."
	}

	return base
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

func (h *DeckHandler) resolveDeckCardEntry(r *http.Request, entry domain.DeckCard) (*domain.Card, error) {
	if strings.TrimSpace(entry.CardID) != "" {
		card, err := h.cardRepo.FindByID(r.Context(), entry.CardID)
		if err != nil {
			return nil, err
		}
		if card != nil {
			return card, nil
		}
	}

	card, err := h.cardRepo.FindByName(r.Context(), entry.CardName)
	if err != nil {
		return nil, err
	}
	if card == nil {
		return nil, &deckCardResolveError{name: entry.CardName}
	}
	return card, nil
}

func extractCardUnitPrices(card *domain.Card) (float64, float64, string) {
	if card == nil {
		return 0, 0, "missing"
	}
	if card.CurrentPrices != nil {
		usd := card.CurrentPrices["usd"]
		eur := card.CurrentPrices["eur"]
		if usd > 0 || eur > 0 {
			return round4(usd), round4(eur), "current_prices"
		}
	}
	if latest := card.LatestPrice(); latest != nil {
		if latest.USD > 0 || latest.EUR > 0 {
			return round4(latest.USD), round4(latest.EUR), "latest_snapshot"
		}
	}
	return 0, 0, "unpriced"
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

func deckToSideboardDecklist(deck *domain.Deck) string {
	if deck == nil {
		return ""
	}
	var b strings.Builder
	for _, c := range deck.Cards {
		if !c.IsSideboard || c.Quantity <= 0 {
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

func buildCurveTypeBreakdown(cards []*domain.Card, quantities map[string]int) []curveTypeBucket {
	buckets := map[int]*curveTypeBucket{}

	for _, card := range cards {
		if card == nil || card.IsLand() {
			continue
		}
		qty := quantities[card.ID]
		if qty <= 0 {
			qty = 1
		}
		cmc := int(math.Round(card.CMC))
		if cmc > 6 {
			cmc = 6
		}
		if cmc < 0 {
			cmc = 0
		}

		row, ok := buckets[cmc]
		if !ok {
			row = &curveTypeBucket{CMC: cmc}
			buckets[cmc] = row
		}

		tl := strings.ToLower(strings.TrimSpace(card.TypeLine))
		added := false
		if strings.Contains(tl, "creature") || strings.Contains(tl, "creatura") {
			row.Creatures += qty
			added = true
		}
		if strings.Contains(tl, "instant") || strings.Contains(tl, "istantaneo") {
			row.Instants += qty
			added = true
		}
		if strings.Contains(tl, "sorcery") || strings.Contains(tl, "stregoneria") {
			row.Sorceries += qty
			added = true
		}
		if strings.Contains(tl, "enchantment") || strings.Contains(tl, "incantesimo") {
			row.Enchantments += qty
			added = true
		}
		if strings.Contains(tl, "artifact") || strings.Contains(tl, "artefatto") {
			row.Artifacts += qty
			added = true
		}
		if strings.Contains(tl, "planeswalker") {
			row.Planeswalkers += qty
			added = true
		}
		if !added {
			row.Sorceries += qty
		}
		row.Total += qty
	}

	out := make([]curveTypeBucket, 0, 7)
	for cmc := 0; cmc <= 6; cmc++ {
		if row, ok := buckets[cmc]; ok {
			out = append(out, *row)
			continue
		}
		out = append(out, curveTypeBucket{CMC: cmc})
	}
	return out
}

func deviationFromMeta(archetype string, avgCMC float64) float64 {
	targets := map[string]float64{
		"aggro":    2.0,
		"midrange": 2.9,
		"control":  3.2,
		"combo":    2.6,
		"ramp":     3.6,
	}
	target, ok := targets[strings.ToLower(strings.TrimSpace(archetype))]
	if !ok || target <= 0 {
		return 0
	}
	return round4(math.Abs(avgCMC-target) / target)
}

func round4(v float64) float64 {
	return math.Round(v*10000) / 10000
}

func metaFitFromDeviation(deviation float64) int {
	if deviation <= 0 {
		return 100
	}
	score := int(math.Round((1 - math.Min(1, deviation)) * 100))
	if score < 0 {
		return 0
	}
	if score > 100 {
		return 100
	}
	return score
}

type knownCombo struct {
	cards       []string
	typeName    string
	description string
	turnKill    int
}

var knownCombos = []knownCombo{
	{cards: []string{"Thassa's Oracle", "Demonic Consultation"}, typeName: "two_card_win", description: "Immediate win line by emptying library before Oracle trigger resolves.", turnKill: 3},
	{cards: []string{"Thassa's Oracle", "Tainted Pact"}, typeName: "two_card_win", description: "Oracle plus selective exile line for deterministic win setup.", turnKill: 3},
	{cards: []string{"Heliod, Sun-Crowned", "Walking Ballista"}, typeName: "infinite_damage", description: "Lifegain counter loop creates infinite damage shots.", turnKill: 4},
	{cards: []string{"Devoted Druid", "Vizier of Remedies"}, typeName: "infinite_mana", description: "Untap counter prevention loop for infinite green mana.", turnKill: 3},
	{cards: []string{"Kiki-Jiki, Mirror Breaker", "Restoration Angel"}, typeName: "infinite_creatures", description: "Repeated blink-copy loop generates lethal board immediately.", turnKill: 5},
	{cards: []string{"Painter's Servant", "Grindstone"}, typeName: "mill_kill", description: "Color lock enables full-library mill in one activation cycle.", turnKill: 3},
	{cards: []string{"Underworld Breach", "Brain Freeze"}, typeName: "storm_loop", description: "Escape loop mills and recasts for lethal storm sequence.", turnKill: 3},
}

func detectDeckCombos(cards []*domain.Card, quantities map[string]int) []deckCombo {
	nameSet := make(map[string]bool, len(cards))
	canonical := make(map[string]string, len(cards))
	byName := make(map[string]*domain.Card, len(cards))
	for _, c := range cards {
		if c == nil {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(c.Name))
		if key == "" {
			continue
		}
		nameSet[key] = true
		canonical[key] = c.Name
		byName[key] = c
	}

	out := make([]deckCombo, 0, 4)
	for _, combo := range knownCombos {
		ok := true
		parts := make([]string, 0, len(combo.cards))
		for _, part := range combo.cards {
			k := strings.ToLower(strings.TrimSpace(part))
			if !nameSet[k] {
				ok = false
				break
			}
			if canonical[k] != "" {
				parts = append(parts, canonical[k])
			} else {
				parts = append(parts, part)
			}
		}
		if ok {
			emb := embeddingPairScore(parts, byName)
			comboScore := 70 + int(math.Round(emb*30))
			if comboScore > 100 {
				comboScore = 100
			}
			out = append(out, deckCombo{
				Cards:       parts,
				Type:        combo.typeName,
				Description: combo.description,
				TurnKill:    combo.turnKill,
				Score:       comboScore,
			})
		}
	}
	return out
}

func detectSynergyPackages(cards []*domain.Card, quantities map[string]int) []synergyPackage {
	type agg struct {
		cards []string
		score int
	}

	pk := map[string]*agg{}
	ensure := func(name string) *agg {
		if v, ok := pk[name]; ok {
			return v
		}
		v := &agg{cards: []string{}, score: 0}
		pk[name] = v
		return v
	}

	for _, c := range cards {
		if c == nil {
			continue
		}
		qty := quantities[c.ID]
		if qty <= 0 {
			qty = 1
		}
		name := strings.TrimSpace(c.Name)
		typeLine := strings.ToLower(c.TypeLine)
		oracle := strings.ToLower(c.OracleText)

		if strings.Contains(oracle, "draw") || strings.Contains(oracle, "scry") || strings.Contains(oracle, "surveil") || strings.Contains(oracle, "look at the top") {
			p := ensure("draw_engine")
			p.cards = append(p.cards, name)
			p.score += qty * 6
		}
		if strings.Contains(oracle, "counter target") || strings.Contains(oracle, "destroy target") || strings.Contains(oracle, "exile target") || strings.Contains(oracle, "damage to any target") || strings.Contains(oracle, "target creature") {
			p := ensure("interaction_suite")
			p.cards = append(p.cards, name)
			p.score += qty * 5
		}
		if strings.Contains(typeLine, "creature") && c.CMC <= 2.0 {
			p := ensure("early_pressure")
			p.cards = append(p.cards, name)
			p.score += qty * 4
		}
		if strings.Contains(typeLine, "land") || strings.Contains(oracle, "add ") || strings.Contains(oracle, "search your library for a land") {
			p := ensure("mana_engine")
			p.cards = append(p.cards, name)
			p.score += qty * 3
		}
	}

	cardIndex := map[string]*domain.Card{}
	for _, c := range cards {
		if c == nil {
			continue
		}
		cardIndex[c.Name] = c
	}

	out := make([]synergyPackage, 0, len(pk))
	for name, data := range pk {
		if len(data.cards) < 3 {
			continue
		}
		seen := map[string]bool{}
		unique := make([]string, 0, len(data.cards))
		for _, n := range data.cards {
			if n == "" || seen[n] {
				continue
			}
			seen[n] = true
			unique = append(unique, n)
		}
		if len(unique) < 3 {
			continue
		}
		embCohesion := packageEmbeddingCohesion(unique, cardIndex)
		score := data.score + int(math.Round(embCohesion*20))
		if score > 100 {
			score = 100
		}
		out = append(out, synergyPackage{Name: name, Cards: unique, Score: score})
	}

	return out
}

func computeSynergyScore(combos []deckCombo, pkgs []synergyPackage) int {
	score := 0
	for _, c := range combos {
		if c.Score > 0 {
			score += int(math.Round(float64(c.Score) * 0.3))
		} else {
			score += 25
		}
	}
	for _, p := range pkgs {
		score += int(math.Round(float64(p.Score) * 0.35))
	}
	if score > 100 {
		return 100
	}
	if score < 0 {
		return 0
	}
	return score
}

func embeddingCoverage(cards []*domain.Card) float64 {
	if len(cards) == 0 {
		return 0
	}
	withEmb := 0
	for _, c := range cards {
		if c != nil && len(c.EmbeddingVector) > 0 {
			withEmb++
		}
	}
	return round4(float64(withEmb) / float64(len(cards)))
}

func embeddingPairScore(names []string, byName map[string]*domain.Card) float64 {
	if len(names) < 2 {
		return 0
	}
	vectors := make([][]float64, 0, len(names))
	for _, n := range names {
		k := strings.ToLower(strings.TrimSpace(n))
		if c, ok := byName[k]; ok && c != nil && len(c.EmbeddingVector) > 0 {
			vectors = append(vectors, c.EmbeddingVector)
		}
	}
	if len(vectors) < 2 {
		return 0
	}
	return averagePairwiseCosine(vectors)
}

func packageEmbeddingCohesion(names []string, byName map[string]*domain.Card) float64 {
	vectors := make([][]float64, 0, len(names))
	for _, n := range names {
		if c, ok := byName[n]; ok && c != nil && len(c.EmbeddingVector) > 0 {
			vectors = append(vectors, c.EmbeddingVector)
		}
	}
	if len(vectors) < 2 {
		return 0
	}
	return averagePairwiseCosine(vectors)
}

func averagePairwiseCosine(vectors [][]float64) float64 {
	if len(vectors) < 2 {
		return 0
	}
	pairs := 0
	total := 0.0
	for i := 0; i < len(vectors); i++ {
		for j := i + 1; j < len(vectors); j++ {
			total += cosineSimilarity(vectors[i], vectors[j])
			pairs++
		}
	}
	if pairs == 0 {
		return 0
	}
	return math.Max(0, math.Min(1, total/float64(pairs)))
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
