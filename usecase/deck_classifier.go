package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/gigliofr/mana-wise/domain"
)

// DeckClassifierUseCase classifies a decklist into its archetype and computes
// a mana-curve fingerprint, color identity, and a short strategy description.
type DeckClassifierUseCase struct {
	cardRepo domain.CardRepository
}

// NewDeckClassifierUseCase creates a new deck classifier.
func NewDeckClassifierUseCase(cardRepo domain.CardRepository) *DeckClassifierUseCase {
	return &DeckClassifierUseCase{cardRepo: cardRepo}
}

// DeckClassifyRequest is the input for the deck classifier.
type DeckClassifyRequest struct {
	Decklist string
	Format   string
}

// CurveBuckets counts spells per CMC window (lands excluded).
type CurveBuckets struct {
	One     int `json:"one"`
	Two     int `json:"two"`
	Three   int `json:"three"`
	Four    int `json:"four"`
	FivePlus int `json:"five_plus"`
}

// DeckClassifyResult is returned by the classifier endpoint.
type DeckClassifyResult struct {
	Archetype           string       `json:"archetype"`
	ColorIdentity       []string     `json:"color_identity"`
	ManaCurve           CurveBuckets `json:"mana_curve"`
	StrategyDescription string       `json:"strategy_description"`
	Confidence          float64      `json:"confidence"`
}

// Execute performs deck classification and returns the result.
func (uc *DeckClassifierUseCase) Execute(ctx context.Context, req DeckClassifyRequest) (DeckClassifyResult, error) {
	entries := parseDecklistQuantities(req.Decklist)
	if len(entries) == 0 {
		return DeckClassifyResult{}, fmt.Errorf("decklist is empty")
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
	archetype := inferPlayerArchetype(features)
	colors := detectColors(entries, cardMap)
	curve := computeCurveBuckets(entries, cardMap)
	confidence := round2(calcConfidence(features))
	desc := buildStrategyDescription(archetype, colors, curve, features, strings.ToLower(strings.TrimSpace(req.Format)))

	return DeckClassifyResult{
		Archetype:           archetype,
		ColorIdentity:       colors,
		ManaCurve:           curve,
		StrategyDescription: desc,
		Confidence:          confidence,
	}, nil
}

// detectColors returns the WUBRG color identity present in the deck.
// It uses card metadata when available, with a land-name heuristic as fallback.
func detectColors(entries map[string]int, cardMap map[string]*domain.Card) []string {
	cards := make([]*domain.Card, 0, len(entries))
	for name := range entries {
		key := strings.ToLower(strings.TrimSpace(name))
		if c, ok := cardMap[key]; ok {
			cards = append(cards, c)
		}
	}

	if len(cards) > 0 {
		return detectedDeckColors(cards)
	}

	// Fallback: infer from basic land names when card metadata is unavailable.
	set := map[string]bool{}
	for name := range entries {
		n := strings.ToLower(strings.TrimSpace(name))
		switch {
		case strings.Contains(n, "plains") || strings.Contains(n, "pianura"):
			set["W"] = true
		case strings.Contains(n, "island") || strings.Contains(n, "isola"):
			set["U"] = true
		case strings.Contains(n, "swamp") || strings.Contains(n, "palude"):
			set["B"] = true
		case strings.Contains(n, "mountain") || strings.Contains(n, "montagna"):
			set["R"] = true
		case strings.Contains(n, "forest") || strings.Contains(n, "foresta"):
			set["G"] = true
		}
	}
	out := make([]string, 0, len(set))
	for c := range set {
		out = append(out, c)
	}
	// Return WUBRG order.
	ordered := []string{}
	for _, c := range []string{"W", "U", "B", "R", "G"} {
		if set[c] {
			ordered = append(ordered, c)
		}
	}
	return ordered
}

// computeCurveBuckets tallies non-land spell counts per CMC window.
func computeCurveBuckets(entries map[string]int, cardMap map[string]*domain.Card) CurveBuckets {
	var b CurveBuckets
	for name, qty := range entries {
		key := strings.ToLower(strings.TrimSpace(name))
		card, ok := cardMap[key]
		if !ok {
			continue
		}
		if isLandCard(card) {
			continue
		}
		cmc := int(card.CMC + 0.5) // round to nearest int
		switch {
		case cmc <= 1:
			b.One += qty
		case cmc == 2:
			b.Two += qty
		case cmc == 3:
			b.Three += qty
		case cmc == 4:
			b.Four += qty
		default:
			b.FivePlus += qty
		}
	}
	return b
}

// buildStrategyDescription generates a human-readable strategy summary.
func buildStrategyDescription(archetype string, colors []string, curve CurveBuckets, f matchupFeatures, format string) string {
	colorName := colorGroupName(colors)
	fmtLabel := ""
	if format != "" {
		fmtLabel = " (" + strings.Title(format) + ")"
	}

	var core string
	switch archetype {
	case "aggro":
		lowCurve := curve.One + curve.Two
		core = fmt.Sprintf(
			"%s aggressive deck%s with %d low-cost threats (1-2 mana). Strategy: apply maximum pressure in the early turns and end the game before the opponent stabilizes.",
			colorName, fmtLabel, lowCurve,
		)
	case "control":
		core = fmt.Sprintf(
			"%s control deck%s with %d interaction spells and %d card-draw effects. Strategy: answer every threat, draw ahead, and close the game with a small suite of finishers.",
			colorName, fmtLabel, f.cheapInteraction+f.counters, f.cantrips,
		)
	case "combo":
		core = fmt.Sprintf(
			"%s combo deck%s with %d cantrips and %d disruption pieces. Strategy: dig for key combo pieces while protecting them with counterspells or hand disruption.",
			colorName, fmtLabel, f.cantrips, f.counters+f.discard,
		)
	default: // midrange
		midCurve := curve.Two + curve.Three + curve.Four
		core = fmt.Sprintf(
			"%s midrange deck%s with a balanced %d-card mid-curve (2-4 mana). Strategy: trade efficiently, generate card advantage, and close with high-impact top-end threats.",
			colorName, fmtLabel, midCurve,
		)
	}
	return core
}

// colorGroupName returns a readable color combination label.
func colorGroupName(colors []string) string {
	if len(colors) == 0 {
		return "Colorless"
	}
	names := map[string]string{
		"W": "White", "U": "Blue", "B": "Black", "R": "Red", "G": "Green",
	}
	// Well-known guild / shard / wedge labels.
	combos := map[string]string{
		"WU": "Azorius", "UB": "Dimir", "BR": "Rakdos", "RG": "Gruul", "WG": "Selesnya",
		"WB": "Orzhov", "UR": "Izzet", "BG": "Golgari", "WR": "Boros", "UG": "Simic",
		"WUB": "Esper", "UBR": "Grixis", "BRG": "Jund", "WRG": "Naya", "WUG": "Bant",
		"WUBR": "Non-Green", "UBRG": "Non-White", "WBRG": "Non-Blue",
		"WURG": "Non-Black", "WUBG": "Non-Red",
		"WUBRG": "Five-Color",
	}
	key := strings.Join(colors, "")
	if label, ok := combos[key]; ok {
		return label
	}
	if len(colors) == 1 {
		return names[colors[0]]
	}
	parts := make([]string, 0, len(colors))
	for _, c := range colors {
		if n, ok := names[c]; ok {
			parts = append(parts, n)
		}
	}
	return strings.Join(parts, "-")
}
