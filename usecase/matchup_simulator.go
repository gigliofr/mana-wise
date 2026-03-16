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
	Decklist        string
	Format          string
	PlayerArchetype string
	Opponents       []string
}

// MatchupEstimate represents one opponent matchup estimate.
type MatchupEstimate struct {
	OpponentArchetype string   `json:"opponent_archetype"`
	WinRate           float64  `json:"win_rate"`
	Confidence        float64  `json:"confidence"`
	KeyFactors        []string `json:"key_factors,omitempty"`
	Verdict           string   `json:"verdict"`
}

// MatchupSimulationResult is returned by the simulator endpoint.
type MatchupSimulationResult struct {
	Format          string            `json:"format"`
	PlayerArchetype string            `json:"player_archetype"`
	Matchups        []MatchupEstimate `json:"matchups"`
	Summary         string            `json:"summary"`
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

	names := make([]string, 0, len(entries))
	for n := range entries {
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
	result := MatchupSimulationResult{
		Format:          strings.ToLower(strings.TrimSpace(req.Format)),
		PlayerArchetype: playerArchetype,
		Matchups:        make([]MatchupEstimate, 0, len(opponents)),
	}

	for _, opp := range opponents {
		winRate, factors := estimateWinRate(playerArchetype, opp, features)
		result.Matchups = append(result.Matchups, MatchupEstimate{
			OpponentArchetype: opp,
			WinRate:           round2(clamp(winRate, 0.25, 0.75)),
			Confidence:        round2(calcConfidence(features)),
			KeyFactors:        factors,
			Verdict:           verdictFromWinRate(winRate),
		})
	}

	sort.Slice(result.Matchups, func(i, j int) bool {
		return result.Matchups[i].WinRate > result.Matchups[j].WinRate
	})

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

func estimateWinRate(player, opp string, f matchupFeatures) (float64, []string) {
	base := baseMatchupWinRate(player, opp)
	adj := 0.0
	factors := []string{}

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

func buildMatchupSummary(res MatchupSimulationResult) string {
	if len(res.Matchups) == 0 {
		return "No matchup estimates available"
	}
	best := res.Matchups[0]
	worst := res.Matchups[len(res.Matchups)-1]
	avg := 0.0
	for _, m := range res.Matchups {
		avg += m.WinRate
	}
	avg = avg / float64(len(res.Matchups))
	return fmt.Sprintf("Estimated pre-sideboard performance: %.0f%% average, best vs %s (%.0f%%), weakest vs %s (%.0f%%).", math.Round(avg*100), best.OpponentArchetype, math.Round(best.WinRate*100), worst.OpponentArchetype, math.Round(worst.WinRate*100))
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
