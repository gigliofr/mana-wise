package usecase

import (
	"context"
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/manawise/api/domain"
)

// MatchupSimulatorUseCase estimates pre-sideboard matchup win rates.
type MatchupSimulatorUseCase struct {
	cardRepo domain.CardRepository
}

// MatchupSimulationRequest contains simulation input.
type MatchupSimulationRequest struct {
	Decklist          string
	SideboardDecklist string
	Format            string
	PlayerArchetype   string
	Opponents         []string
	OnPlay            bool               // true = player goes first; affects win-rate estimates
	MetaShares        map[string]float64 // optional per-archetype prevalence overrides (0-1); missing archetypes use format defaults
}

// MatchupEstimate represents one opponent matchup estimate.
type MatchupEstimate struct {
	OpponentArchetype string          `json:"opponent_archetype"`
	MetaShare         float64         `json:"meta_share"`
	WinRate           float64         `json:"win_rate"`
	PostBoardWinRate  float64         `json:"post_board_win_rate,omitempty"`
	SideboardDelta    float64         `json:"sideboard_delta,omitempty"`
	SuggestedIns      []SideboardSwap `json:"suggested_ins,omitempty"`
	SuggestedOuts     []SideboardSwap `json:"suggested_outs,omitempty"`
	Confidence        float64         `json:"confidence"`
	KeyFactors        []string        `json:"key_factors,omitempty"`
	Verdict           string          `json:"verdict"`
}

// MatchupSimulationResult is returned by the simulator endpoint.
type MatchupSimulationResult struct {
	Format                string            `json:"format"`
	PlayerArchetype       string            `json:"player_archetype"`
	OnPlay                bool              `json:"on_play"`
	MetaWeightedWinRate   float64           `json:"meta_weighted_win_rate"`
	Matchups              []MatchupEstimate `json:"matchups"`
	Summary               string            `json:"summary"`
}

type matchupFeatures struct {
	lands            int
	cheapInteraction int
	counters         int
	discard          int
	cantrips         int
	sweepers         int
	threats          int
	topEnd           int
	recognized       int
	totalCards       int
}

// NewMatchupSimulatorUseCase creates a new matchup simulator.
func NewMatchupSimulatorUseCase(cardRepo domain.CardRepository) *MatchupSimulatorUseCase {
	return &MatchupSimulatorUseCase{cardRepo: cardRepo}
}

// Execute computes heuristic pre-sideboard matchup estimations.
func (uc *MatchupSimulatorUseCase) Execute(ctx context.Context, req MatchupSimulationRequest) (MatchupSimulationResult, error) {
	entries := parseDecklistQuantities(req.Decklist)
	if len(entries) == 0 {
		return MatchupSimulationResult{}, fmt.Errorf("decklist is empty")
	}
	sideboard := parseDecklistQuantities(req.SideboardDecklist)

	names := make([]string, 0, len(entries))
	for n := range entries {
		names = append(names, n)
	}
	for n := range sideboard {
		names = append(names, n)
	}

	cardMap := map[string]*domain.Card{}
	if uc.cardRepo != nil {
		if cards, err := uc.cardRepo.FindByNames(ctx, names); err == nil {
			for _, c := range cards {
				if c != nil {
					cardMap[strings.ToLower(strings.TrimSpace(c.Name))] = c
				}
			}
		}
	}

	features := computeMatchupFeatures(entries, cardMap)
	playerArchetype := normalizeArchetype(req.PlayerArchetype)
	if strings.TrimSpace(req.PlayerArchetype) == "" {
		playerArchetype = inferPlayerArchetype(features)
	}

	opponents := normalizeOpponents(req.Opponents)
	format := strings.ToLower(strings.TrimSpace(req.Format))
	metaShares := resolveMetaShares(format, opponents, req.MetaShares)

	result := MatchupSimulationResult{
		Format:          format,
		PlayerArchetype: playerArchetype,
		OnPlay:          req.OnPlay,
		Matchups:        make([]MatchupEstimate, 0, len(opponents)),
	}

	for _, opp := range opponents {
		winRate, factors := estimateWinRate(playerArchetype, opp, features, req.OnPlay)
		postBoard, sideDelta, sideFactors := applySideboardDelta(opp, winRate, sideboard, cardMap)
		factors = append(factors, sideFactors...)
		sideIns, sideOuts := buildMiniSideboardPlan(entries, sideboard, cardMap, opp)
		result.Matchups = append(result.Matchups, MatchupEstimate{
			OpponentArchetype: opp,
			MetaShare:         round2(metaShares[opp]),
			WinRate:           round2(clamp(winRate, 0.25, 0.75)),
			PostBoardWinRate:  round2(clamp(postBoard, 0.25, 0.8)),
			SideboardDelta:    round2(sideDelta),
			SuggestedIns:      sideIns,
			SuggestedOuts:     sideOuts,
			Confidence:        round2(calcConfidence(features)),
			KeyFactors:        factors,
			Verdict:           verdictFromWinRate(winRate),
		})
	}

	sort.Slice(result.Matchups, func(i, j int) bool {
		return result.Matchups[i].WinRate > result.Matchups[j].WinRate
	})

	result.MetaWeightedWinRate = round2(computeMetaWeightedWinRate(result.Matchups))
	result.Summary = buildMatchupSummary(result)
	return result, nil
}

func normalizeOpponents(opponents []string) []string {
	if len(opponents) == 0 {
		return []string{"aggro", "midrange", "control", "combo"}
	}
	out := []string{}
	seen := map[string]bool{}
	for _, o := range opponents {
		n := normalizeArchetype(o)
		if !seen[n] {
			seen[n] = true
			out = append(out, n)
		}
	}
	if len(out) == 0 {
		return []string{"aggro", "midrange", "control", "combo"}
	}
	return out
}

func computeMatchupFeatures(entries map[string]int, cardMap map[string]*domain.Card) matchupFeatures {
	f := matchupFeatures{}
	for name, qty := range entries {
		if qty <= 0 {
			continue
		}
		lower := strings.ToLower(strings.TrimSpace(name))
		card := cardMap[lower]
		for i := 0; i < qty; i++ {
			f.totalCards++
			isLand, isCounter, isDiscard, isCantrip, isSweeper, isThreat, isCheapInteraction, isTopEnd := classifyMatchupCard(lower, card)
			if card != nil {
				f.recognized++
			}
			if isLand {
				f.lands++
				continue
			}
			if isCounter {
				f.counters++
			}
			if isDiscard {
				f.discard++
			}
			if isCantrip {
				f.cantrips++
			}
			if isSweeper {
				f.sweepers++
			}
			if isThreat {
				f.threats++
			}
			if isCheapInteraction {
				f.cheapInteraction++
			}
			if isTopEnd {
				f.topEnd++
			}
		}
	}
	return f
}

func classifyMatchupCard(nameLower string, card *domain.Card) (bool, bool, bool, bool, bool, bool, bool, bool) {
	text := nameLower
	typeLine := ""
	oracle := ""
	cmc := 2.5
	if card != nil {
		typeLine = strings.ToLower(card.TypeLine)
		oracle = strings.ToLower(card.OracleText)
		text = text + " " + typeLine + " " + oracle
		if card.CMC > 0 {
			cmc = card.CMC
		}
	}

	isLand := strings.Contains(typeLine, "land") || strings.Contains(nameLower, "plains") || strings.Contains(nameLower, "island") || strings.Contains(nameLower, "swamp") || strings.Contains(nameLower, "mountain") || strings.Contains(nameLower, "forest") || strings.Contains(nameLower, "evolving wilds")
	isCounter := strings.Contains(text, "counter target") || strings.Contains(nameLower, "negate") || strings.Contains(nameLower, "make disappear")
	isDiscard := strings.Contains(text, "discard") || strings.Contains(nameLower, "duress") || strings.Contains(nameLower, "thoughtseize")
	isCantrip := strings.Contains(text, "draw a card") || strings.Contains(nameLower, "consider") || strings.Contains(nameLower, "opt") || strings.Contains(nameLower, "impulse") || strings.Contains(nameLower, "ponder") || strings.Contains(nameLower, "preordain")
	isSweeper := strings.Contains(text, "destroy all") || strings.Contains(text, "each creature") || strings.Contains(nameLower, "brotherhood") || strings.Contains(nameLower, "sunfall")
	isThreat := strings.Contains(typeLine, "creature") || strings.Contains(typeLine, "planeswalker") || strings.Contains(typeLine, "battle") || strings.Contains(nameLower, "sheoldred") || strings.Contains(nameLower, "dragon")
	isRemoval := strings.Contains(text, "destroy target") || strings.Contains(text, "exile target") || strings.Contains(text, "damage to any target") || strings.Contains(nameLower, "go for the throat") || strings.Contains(nameLower, "lightning strike")
	isCheapInteraction := (isCounter || isDiscard || isRemoval) && cmc <= 3.0
	isTopEnd := cmc >= 4.0

	return isLand, isCounter, isDiscard, isCantrip, isSweeper, isThreat, isCheapInteraction, isTopEnd
}

func inferPlayerArchetype(f matchupFeatures) string {
	nonLands := f.totalCards - f.lands
	if nonLands <= 0 {
		return "midrange"
	}
	if f.cheapInteraction+f.counters+f.cantrips >= 18 && f.threats <= 14 {
		return "control"
	}
	if f.threats >= 18 && f.topEnd <= 8 {
		return "aggro"
	}
	if f.cantrips >= 10 && f.counters+f.discard >= 8 && f.threats <= 12 {
		return "combo"
	}
	return "midrange"
}

func estimateWinRate(player, opp string, f matchupFeatures, onPlay bool) (float64, []string) {
	base := baseMatchupWinRate(player, opp)
	playDelta, playFactor := playDrawAdjustment(player, opp, onPlay)
	adj := playDelta
	factors := []string{}
	if playFactor != "" {
		factors = append(factors, playFactor)
	}

	switch opp {
	case "aggro":
		adj += 0.006 * float64(f.cheapInteraction)
		adj += 0.012 * float64(f.sweepers)
		adj -= 0.004 * float64(f.topEnd)
		if f.cheapInteraction >= 10 {
			factors = append(factors, "High cheap interaction density")
		}
		if f.sweepers >= 2 {
			factors = append(factors, "Main-deck sweepers improve game 1")
		}
	case "control":
		adj += 0.008 * float64(f.threats)
		adj += 0.007 * float64(f.cantrips)
		adj += 0.01 * float64(f.discard)
		adj += 0.005 * float64(f.counters)
		if f.threats >= 14 {
			factors = append(factors, "Pressure package can tax control answers")
		}
		if f.discard+f.counters >= 8 {
			factors = append(factors, "Stack/disruption tools are above average")
		}
	case "combo":
		adj += 0.012 * float64(f.counters)
		adj += 0.01 * float64(f.discard)
		adj += 0.004 * float64(f.cantrips)
		adj -= 0.003 * float64(f.topEnd)
		if f.counters+f.discard >= 8 {
			factors = append(factors, "Disruption package is combo-relevant")
		}
	case "midrange":
		adj += 0.005 * float64(f.threats)
		adj += 0.005 * float64(f.cheapInteraction)
		adj += 0.003 * float64(f.cantrips)
		if f.threats >= 12 && f.cheapInteraction >= 8 {
			factors = append(factors, "Balanced threats and interaction")
		}
	}

	if len(factors) == 0 {
		factors = append(factors, "Heuristic estimate with limited card identity coverage")
	}

	return base + adj, factors
}

// playDrawAdjustment returns the win-rate delta and a human-readable factor
// for going first (onPlay=true) or second (onPlay=false).
func playDrawAdjustment(player, opp string, onPlay bool) (float64, string) {
	// Positive values = advantage when going first.
	matrix := map[string]map[string]float64{
		"aggro":    {"aggro": 0.010, "midrange": 0.025, "control": 0.035, "combo": 0.020},
		"midrange": {"aggro": 0.015, "midrange": 0.010, "control": 0.020, "combo": 0.010},
		"control":  {"aggro": -0.025, "midrange": 0.010, "control": 0.015, "combo": 0.020},
		"combo":    {"aggro": 0.020, "midrange": 0.015, "control": 0.015, "combo": 0.010},
	}
	delta := 0.015 // default small on-play edge
	if row, ok := matrix[player]; ok {
		if v, ok2 := row[opp]; ok2 {
			delta = v
		}
	}
	if !onPlay {
		delta = -delta
	}
	if delta == 0 {
		return 0, ""
	}
	position := "on the draw"
	if onPlay {
		position = "on the play"
	}
	sign := "+"
	if delta < 0 {
		sign = "-"
	}
	return delta, fmt.Sprintf("Play/draw position (%s): %s%.0fpp", position, sign, math.Abs(delta*100))
}

func baseMatchupWinRate(player, opp string) float64 {
	matrix := map[string]map[string]float64{
		"aggro":    {"aggro": 0.50, "midrange": 0.48, "control": 0.54, "combo": 0.47},
		"midrange": {"aggro": 0.52, "midrange": 0.50, "control": 0.49, "combo": 0.48},
		"control":  {"aggro": 0.46, "midrange": 0.51, "control": 0.50, "combo": 0.47},
		"combo":    {"aggro": 0.53, "midrange": 0.51, "control": 0.52, "combo": 0.50},
	}
	if row, ok := matrix[player]; ok {
		if v, ok := row[opp]; ok {
			return v
		}
	}
	return 0.5
}

func calcConfidence(f matchupFeatures) float64 {
	if f.totalCards == 0 {
		return 0.5
	}
	coverage := float64(f.recognized) / float64(f.totalCards)
	base := 0.52 + (0.35 * coverage)
	if f.totalCards < 50 {
		base -= 0.05
	}
	return clamp(base, 0.45, 0.9)
}

func verdictFromWinRate(winRate float64) string {
	if winRate >= 0.56 {
		return "favored"
	}
	if winRate >= 0.49 {
		return "close"
	}
	return "unfavored"
}

// defaultMetaShares returns approximate archetype prevalence for the given format.
func defaultMetaShares(format string) map[string]float64 {
	switch format {
	case "standard":
		return map[string]float64{"aggro": 0.28, "midrange": 0.32, "control": 0.24, "combo": 0.16}
	case "pioneer":
		return map[string]float64{"aggro": 0.25, "midrange": 0.30, "control": 0.25, "combo": 0.20}
	case "modern":
		return map[string]float64{"aggro": 0.22, "midrange": 0.28, "control": 0.20, "combo": 0.30}
	case "legacy", "vintage":
		return map[string]float64{"aggro": 0.18, "midrange": 0.22, "control": 0.25, "combo": 0.35}
	case "commander", "edh":
		return map[string]float64{"aggro": 0.10, "midrange": 0.50, "control": 0.25, "combo": 0.15}
	default:
		return map[string]float64{"aggro": 0.25, "midrange": 0.25, "control": 0.25, "combo": 0.25}
	}
}

// resolveMetaShares merges user overrides with format defaults and normalises to
// the subset of opponents actually requested, so shares always sum to 1.0.
func resolveMetaShares(format string, opponents []string, overrides map[string]float64) map[string]float64 {
	defaults := defaultMetaShares(format)
	raw := map[string]float64{}
	for _, opp := range opponents {
		if v, ok := overrides[opp]; ok && v > 0 {
			raw[opp] = v
		} else if v, ok2 := defaults[opp]; ok2 {
			raw[opp] = v
		} else {
			raw[opp] = 0.25 // fallback for unknown archetypes
		}
	}
	// Normalise
	total := 0.0
	for _, v := range raw {
		total += v
	}
	if total == 0 {
		total = 1
	}
	out := map[string]float64{}
	for k, v := range raw {
		out[k] = v / total
	}
	return out
}

// computeMetaWeightedWinRate returns win rate weighted by each matchup's MetaShare.
func computeMetaWeightedWinRate(matchups []MatchupEstimate) float64 {
	if len(matchups) == 0 {
		return 0
	}
	w := 0.0
	for _, m := range matchups {
		w += m.WinRate * m.MetaShare
	}
	return w
}

func buildMatchupSummary(res MatchupSimulationResult) string {
	if len(res.Matchups) == 0 {
		return "No matchup estimates available"
	}
	best := res.Matchups[0]
	worst := res.Matchups[len(res.Matchups)-1]
	avgPre := 0.0
	avgPost := 0.0
	for _, m := range res.Matchups {
		avgPre += m.WinRate
		avgPost += m.PostBoardWinRate
	}
	avgPre = avgPre / float64(len(res.Matchups))
	avgPost = avgPost / float64(len(res.Matchups))
	meta := fmt.Sprintf(" (meta-weighted: %.0f%%)", math.Round(res.MetaWeightedWinRate*100))
	if avgPost > 0 {
		return fmt.Sprintf("Estimated performance: %.0f%% pre-board and %.0f%% post-board average%s; best pre-board vs %s (%.0f%%), weakest pre-board vs %s (%.0f%%).", math.Round(avgPre*100), math.Round(avgPost*100), meta, best.OpponentArchetype, math.Round(best.WinRate*100), worst.OpponentArchetype, math.Round(worst.WinRate*100))
	}
	return fmt.Sprintf("Estimated pre-sideboard performance: %.0f%% average%s, best vs %s (%.0f%%), weakest vs %s (%.0f%%).", math.Round(avgPre*100), meta, best.OpponentArchetype, math.Round(best.WinRate*100), worst.OpponentArchetype, math.Round(worst.WinRate*100))
}

func applySideboardDelta(opp string, preWinRate float64, sideboard map[string]int, cardMap map[string]*domain.Card) (float64, float64, []string) {
	if len(sideboard) == 0 {
		return preWinRate, 0, nil
	}
	score := 0.0
	for name, qty := range sideboard {
		if qty <= 0 {
			continue
		}
		lower := strings.ToLower(strings.TrimSpace(name))
		card := cardMap[lower]
		score += float64(qty) * sideboardTagWeight(opp, inferSideboardTags(lower, card))
	}
	if score <= 0 {
		return preWinRate, 0, []string{"Sideboard has low specific overlap for this matchup"}
	}
	delta := clamp(score*0.0035, 0, 0.08)
	return preWinRate + delta, delta, []string{fmt.Sprintf("Post-board tools project +%.0fpp vs %s", math.Round(delta*100), opp)}
}

func inferSideboardTags(nameLower string, card *domain.Card) map[string]bool {
	text := nameLower
	typeLine := ""
	oracle := ""
	if card != nil {
		typeLine = strings.ToLower(card.TypeLine)
		oracle = strings.ToLower(card.OracleText)
		text = text + " " + typeLine + " " + oracle
	}
	tags := map[string]bool{}
	if strings.Contains(text, "counter target") || strings.Contains(nameLower, "negate") || strings.Contains(nameLower, "disdainful") {
		tags["counter"] = true
		tags["anti_combo"] = true
		tags["anti_control"] = true
	}
	if strings.Contains(text, "discard") || strings.Contains(nameLower, "duress") || strings.Contains(nameLower, "thoughtseize") {
		tags["discard"] = true
		tags["anti_combo"] = true
		tags["anti_control"] = true
	}
	if strings.Contains(text, "destroy all") || strings.Contains(text, "each creature") || strings.Contains(nameLower, "sunfall") || strings.Contains(nameLower, "brotherhood") {
		tags["sweeper"] = true
		tags["anti_aggro"] = true
	}
	if strings.Contains(text, "destroy target creature") || strings.Contains(text, "exile target creature") || strings.Contains(nameLower, "go for the throat") || strings.Contains(nameLower, "bolt") || strings.Contains(nameLower, "strike") {
		tags["cheap_removal"] = true
		tags["anti_aggro"] = true
	}
	if strings.Contains(text, "graveyard") && (strings.Contains(text, "exile") || strings.Contains(text, "can't")) {
		tags["graveyard_hate"] = true
		tags["anti_combo"] = true
	}
	if strings.Contains(text, "gain") && strings.Contains(text, "life") {
		tags["lifegain"] = true
		tags["anti_aggro"] = true
	}
	if strings.Contains(typeLine, "creature") || strings.Contains(typeLine, "planeswalker") {
		tags["threat"] = true
	}
	return tags
}

func sideboardTagWeight(opp string, tags map[string]bool) float64 {
	weights := map[string]map[string]float64{
		"aggro": {
			"anti_aggro":    2.0,
			"cheap_removal": 1.8,
			"sweeper":       2.6,
			"lifegain":      1.6,
		},
		"control": {
			"anti_control": 1.8,
			"counter":      1.5,
			"discard":      1.9,
			"threat":       1.1,
		},
		"combo": {
			"anti_combo":    2.0,
			"counter":       1.9,
			"discard":       1.9,
			"graveyard_hate": 1.5,
		},
		"midrange": {
			"cheap_removal": 1.2,
			"threat":        1.2,
			"counter":       0.8,
		},
	}
	w := 0.0
	table := weights[opp]
	for tag := range tags {
		w += table[tag]
	}
	return w
}

func buildMiniSideboardPlan(mainDeck, sideboard map[string]int, cardMap map[string]*domain.Card, opponent string) ([]SideboardSwap, []SideboardSwap) {
	if len(sideboard) == 0 {
		return nil, nil
	}

	insScored := scoreSideboardIns(sideboard, cardMap, desiredTagsForMatchup(opponent))
	if len(insScored) == 0 {
		fallbackNames := make([]string, 0, len(sideboard))
		for name := range sideboard {
			fallbackNames = append(fallbackNames, name)
		}
		sort.Strings(fallbackNames)
		ins := make([]SideboardSwap, 0, 3)
		totalIn := 0
		for _, name := range fallbackNames {
			if len(ins) >= 3 {
				break
			}
			qty := sideboard[name]
			if qty > 2 {
				qty = 2
			}
			if qty <= 0 {
				continue
			}
			ins = append(ins, SideboardSwap{Card: name, Qty: qty, Reason: "Flexible post-board slot for this matchup"})
			totalIn += qty
		}
		outsScored := scoreMainDeckOuts(mainDeck, cardMap, opponent, totalIn)
		outs := make([]SideboardSwap, 0, 3)
		for i := 0; i < len(outsScored) && i < 3; i++ {
			qty := outsScored[i].qty
			if qty > 2 {
				qty = 2
			}
			if qty <= 0 {
				continue
			}
			outs = append(outs, SideboardSwap{Card: outsScored[i].name, Qty: qty, Reason: outsScored[i].reason})
		}
		return ins, outs
	}
	sort.Slice(insScored, func(i, j int) bool { return insScored[i].score > insScored[j].score })

	maxUnique := 3
	if len(insScored) < maxUnique {
		maxUnique = len(insScored)
	}
	ins := make([]SideboardSwap, 0, maxUnique)
	totalIn := 0
	for i := 0; i < maxUnique; i++ {
		qty := insScored[i].qty
		if qty > 2 {
			qty = 2
		}
		if qty <= 0 {
			continue
		}
		totalIn += qty
		ins = append(ins, SideboardSwap{Card: insScored[i].name, Qty: qty, Reason: insScored[i].reason})
	}

	outsScored := scoreMainDeckOuts(mainDeck, cardMap, opponent, totalIn)
	maxOutUnique := 3
	if len(outsScored) < maxOutUnique {
		maxOutUnique = len(outsScored)
	}
	outs := make([]SideboardSwap, 0, maxOutUnique)
	for i := 0; i < maxOutUnique; i++ {
		qty := outsScored[i].qty
		if qty > 2 {
			qty = 2
		}
		if qty <= 0 {
			continue
		}
		outs = append(outs, SideboardSwap{Card: outsScored[i].name, Qty: qty, Reason: outsScored[i].reason})
	}

	return ins, outs
}

func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}
