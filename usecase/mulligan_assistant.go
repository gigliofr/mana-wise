package usecase

import (
	"context"
	"fmt"
	"math/rand"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/manawise/api/domain"
)

var mulliganLineRE = regexp.MustCompile(`^(\d+)x?\s+(.+)$`)

// MulliganAssistantUseCase estimates keep rates for different hand sizes.
type MulliganAssistantUseCase struct {
	cardRepo domain.CardRepository
}

// MulliganSimulationRequest carries the simulation input.
type MulliganSimulationRequest struct {
	Decklist   string
	Format     string
	Archetype  string
	Iterations int
	OnPlay     bool
}

// MulliganSummary contains aggregate stats for a specific hand size.
type MulliganSummary struct {
	HandSize      int     `json:"hand_size"`
	KeepRate      float64 `json:"keep_rate"`
	AvgLands      float64 `json:"avg_lands"`
	AvgEarlyPlays float64 `json:"avg_early_plays"`
}

// MulliganSimulationResult is returned by the mulligan simulation.
type MulliganSimulationResult struct {
	Format         string            `json:"format"`
	Archetype      string            `json:"archetype"`
	OnPlay         bool              `json:"on_play"`
	Iterations     int               `json:"iterations"`
	Summaries      []MulliganSummary `json:"summaries"`
	Recommendation string            `json:"recommendation"`
}

type mulliganCard struct {
	isLand        bool
	isInteraction bool
	isCantrip     bool
	isThreat      bool
	cmc           float64
}

// NewMulliganAssistantUseCase creates the mulligan simulator.
func NewMulliganAssistantUseCase(cardRepo domain.CardRepository) *MulliganAssistantUseCase {
	return &MulliganAssistantUseCase{cardRepo: cardRepo}
}

// Execute runs mulligan simulations for hand sizes 7, 6 and 5.
func (uc *MulliganAssistantUseCase) Execute(ctx context.Context, req MulliganSimulationRequest) (MulliganSimulationResult, error) {
	entries := parseMulliganDecklist(req.Decklist)
	if len(entries) == 0 {
		return MulliganSimulationResult{}, fmt.Errorf("decklist is empty")
	}

	iterations := req.Iterations
	if iterations <= 0 {
		iterations = 1000
	}
	if iterations > 10000 {
		iterations = 10000
	}

	nameList := make([]string, 0, len(entries))
	for name := range entries {
		nameList = append(nameList, name)
	}

	cardMap := map[string]*domain.Card{}
	if uc.cardRepo != nil {
		if cards, err := uc.cardRepo.FindByNames(ctx, nameList); err == nil {
			for _, c := range cards {
				if c != nil {
					cardMap[strings.ToLower(strings.TrimSpace(c.Name))] = c
				}
			}
		}
	}

	pool := buildSimulationPool(entries, cardMap)
	if len(pool) < 40 {
		return MulliganSimulationResult{}, fmt.Errorf("deck too small for mulligan simulation")
	}

	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	archetype := normalizeArchetype(req.Archetype)

	result := MulliganSimulationResult{
		Format:     strings.ToLower(strings.TrimSpace(req.Format)),
		Archetype:  archetype,
		OnPlay:     req.OnPlay,
		Iterations: iterations,
		Summaries:  []MulliganSummary{},
	}

	for _, handSize := range []int{7, 6, 5} {
		keepCount := 0
		lands := 0
		early := 0

		for i := 0; i < iterations; i++ {
			hand := drawHand(pool, handSize, rng)
			stats := evaluateHand(hand)
			lands += stats.lands
			early += stats.earlyPlays
			if shouldKeep(stats, archetype, req.OnPlay) {
				keepCount++
			}
		}

		result.Summaries = append(result.Summaries, MulliganSummary{
			HandSize:      handSize,
			KeepRate:      round2(float64(keepCount) / float64(iterations)),
			AvgLands:      round2(float64(lands) / float64(iterations)),
			AvgEarlyPlays: round2(float64(early) / float64(iterations)),
		})
	}

	result.Recommendation = buildMulliganRecommendation(result)
	return result, nil
}

type handStats struct {
	lands       int
	earlyPlays  int
	interaction int
	cantrips    int
	threats     int
	nonLands    int
}

func evaluateHand(hand []mulliganCard) handStats {
	stats := handStats{}
	for _, c := range hand {
		if c.isLand {
			stats.lands++
			continue
		}
		stats.nonLands++
		if c.cmc <= 2.5 {
			stats.earlyPlays++
		}
		if c.isInteraction {
			stats.interaction++
		}
		if c.isCantrip {
			stats.cantrips++
		}
		if c.isThreat {
			stats.threats++
		}
	}
	return stats
}

func shouldKeep(stats handStats, archetype string, onPlay bool) bool {
	landMin, landMax := 2, 4
	if !onPlay {
		landMax = 5
	}

	switch archetype {
	case "aggro":
		if !onPlay {
			landMin = 1
		}
		return stats.lands >= landMin && stats.lands <= 4 && stats.earlyPlays >= 2 && stats.nonLands >= 3
	case "control":
		return stats.lands >= 2 && stats.lands <= landMax && stats.interaction >= 1 && (stats.interaction+stats.cantrips) >= 2
	case "combo":
		return stats.lands >= 2 && stats.lands <= 4 && (stats.cantrips+stats.interaction) >= 1 && (stats.threats+stats.cantrips) >= 2
	default: // midrange
		return stats.lands >= 2 && stats.lands <= 4 && stats.earlyPlays >= 1 && stats.nonLands >= 3
	}
}

func buildMulliganRecommendation(res MulliganSimulationResult) string {
	if len(res.Summaries) == 0 {
		return "No recommendation available"
	}
	seven := res.Summaries[0]
	if seven.KeepRate >= 0.72 {
		return "Keep discipline can be conservative: 7-card hands are generally stable."
	}
	if seven.KeepRate >= 0.58 {
		return "Mulligan decisions are medium sensitivity: prioritize hands with early interaction and clean mana."
	}
	return "High mulligan pressure detected: adjust mana base or increase early-game density before event play."
}

func normalizeArchetype(a string) string {
	x := strings.ToLower(strings.TrimSpace(a))
	switch x {
	case "aggressive":
		return "aggro"
	case "ctrl":
		return "control"
	case "aggro", "control", "combo", "midrange", "ramp":
		return x
	default:
		return "midrange"
	}
}

func parseMulliganDecklist(decklist string) map[string]int {
	out := map[string]int{}
	for _, raw := range strings.Split(decklist, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "//") || strings.HasPrefix(line, "#") {
			continue
		}
		if isSectionHeader(line) {
			continue
		}
		qty, name := parseMulliganLine(line)
		if name == "" || qty <= 0 {
			continue
		}
		out[name] += qty
	}
	return out
}

func parseMulliganLine(line string) (int, string) {
	m := mulliganLineRE.FindStringSubmatch(line)
	if len(m) == 3 {
		q, err := strconv.Atoi(m[1])
		if err == nil {
			return q, strings.TrimSpace(m[2])
		}
	}
	return 1, strings.TrimSpace(line)
}

func buildSimulationPool(entries map[string]int, cardMap map[string]*domain.Card) []mulliganCard {
	pool := []mulliganCard{}
	for name, qty := range entries {
		if qty <= 0 {
			continue
		}
		mc := classifyMulliganCard(name, cardMap[strings.ToLower(strings.TrimSpace(name))])
		for i := 0; i < qty; i++ {
			pool = append(pool, mc)
		}
	}
	return pool
}

func classifyMulliganCard(name string, card *domain.Card) mulliganCard {
	nameLower := strings.ToLower(strings.TrimSpace(name))
	if card == nil {
		isBasicLand := strings.Contains(nameLower, "plains") || strings.Contains(nameLower, "island") || strings.Contains(nameLower, "swamp") || strings.Contains(nameLower, "mountain") || strings.Contains(nameLower, "forest") || strings.Contains(nameLower, "land")
		isCantrip := strings.Contains(nameLower, "consider") || strings.Contains(nameLower, "opt") || strings.Contains(nameLower, "impulse") || strings.Contains(nameLower, "ponder") || strings.Contains(nameLower, "preordain")
		isInteraction := strings.Contains(nameLower, "negate") || strings.Contains(nameLower, "counter") || strings.Contains(nameLower, "duress") || strings.Contains(nameLower, "thoughtseize") || strings.Contains(nameLower, "go for the throat") || strings.Contains(nameLower, "bolt") || strings.Contains(nameLower, "strike") || strings.Contains(nameLower, "remove")
		return mulliganCard{isLand: isBasicLand, isCantrip: isCantrip, isInteraction: isInteraction, isThreat: !isBasicLand, cmc: 2.5}
	}

	typeLine := strings.ToLower(card.TypeLine)
	oracle := strings.ToLower(card.OracleText)
	isLand := strings.Contains(typeLine, "land")
	isInteraction := strings.Contains(oracle, "counter target") || strings.Contains(oracle, "destroy target") || strings.Contains(oracle, "exile target") || strings.Contains(oracle, "damage to") || strings.Contains(oracle, "target opponent discards")
	isCantrip := strings.Contains(oracle, "draw a card") || strings.Contains(oracle, "scry") || strings.Contains(oracle, "surveil") || strings.Contains(oracle, "look at the top")
	isThreat := strings.Contains(typeLine, "creature") || strings.Contains(typeLine, "planeswalker") || strings.Contains(typeLine, "battle")

	return mulliganCard{
		isLand:        isLand,
		isInteraction: isInteraction,
		isCantrip:     isCantrip,
		isThreat:      isThreat,
		cmc:           card.CMC,
	}
}

func drawHand(pool []mulliganCard, size int, rng *rand.Rand) []mulliganCard {
	if size > len(pool) {
		size = len(pool)
	}
	idx := make([]int, len(pool))
	for i := range idx {
		idx[i] = i
	}
	for i := len(idx) - 1; i > 0; i-- {
		j := rng.Intn(i + 1)
		idx[i], idx[j] = idx[j], idx[i]
	}
	hand := make([]mulliganCard, 0, size)
	for i := 0; i < size; i++ {
		hand = append(hand, pool[idx[i]])
	}
	return hand
}
