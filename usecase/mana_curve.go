package usecase

import (
	"fmt"
	"math"
	"strings"

	"github.com/manawise/api/domain"
)

// formatParams holds tuning parameters for a given MTG format.
type formatParams struct {
	deckSize       int
	idealLandRatio float64 // fraction of deck that should be lands
	maxCMC         int     // CMC above which cards are considered "heavy"
	curveTarget    []int   // ideal number of cards per CMC slot (1-6+)
}

var formatDefaults = map[string]formatParams{
	"commander": {deckSize: 100, idealLandRatio: 0.37, maxCMC: 5, curveTarget: []int{0, 12, 15, 18, 12, 8, 5}},
	"modern":    {deckSize: 60, idealLandRatio: 0.38, maxCMC: 4, curveTarget: []int{0, 10, 14, 10, 8, 4, 2}},
	"pioneer":   {deckSize: 60, idealLandRatio: 0.38, maxCMC: 4, curveTarget: []int{0, 8, 14, 12, 8, 4, 2}},
	"legacy":    {deckSize: 60, idealLandRatio: 0.37, maxCMC: 3, curveTarget: []int{0, 14, 16, 10, 6, 4, 2}},
	"vintage":   {deckSize: 60, idealLandRatio: 0.35, maxCMC: 3, curveTarget: []int{0, 16, 14, 10, 6, 4, 2}},
	"standard":  {deckSize: 60, idealLandRatio: 0.40, maxCMC: 5, curveTarget: []int{0, 6, 12, 14, 10, 6, 4}},
	"pauper":    {deckSize: 60, idealLandRatio: 0.38, maxCMC: 4, curveTarget: []int{0, 10, 14, 10, 8, 4, 2}},
}

// defaultParams returns the params for a format, falling back to modern.
func defaultParams(format string) formatParams {
	if p, ok := formatDefaults[strings.ToLower(format)]; ok {
		return p
	}
	return formatDefaults["modern"]
}

// AnalyzeManaCurve performs the deterministic mana-curve analysis for a set of cards.
func AnalyzeManaCurve(cards []*domain.Card, quantities map[string]int, format string) domain.ManaAnalysis {
	params := defaultParams(format)

	result := domain.ManaAnalysis{
		Format:            format,
		ColorDistribution: make(map[string]int),
		PipDistribution:   make(map[string]int),
	}

	// Build CMC buckets (0-6+) and count lands.
	buckets := make(map[int]int)
	totalCards := 0
	totalCMC := 0.0
	landCount := 0

	for _, card := range cards {
		qty := quantities[card.ID]
		if qty == 0 {
			qty = 1
		}
		cardCMC := int(math.Round(card.CMC))

		isLand := isLandCard(card)
		if isLand {
			landCount += qty
		} else {
			bucketKey := cardCMC
			if bucketKey > 6 {
				bucketKey = 6
			}
			buckets[bucketKey] += qty
			totalCMC += card.CMC * float64(qty)
			totalCards += qty
		}

		// Colour distribution
		for _, c := range card.Colors {
			result.ColorDistribution[c] += qty
		}

		// Pip distribution — count coloured pips in mana cost (excluding lands).
		if !isLand {
			for pip, count := range countPips(card.ManaCost) {
				result.PipDistribution[pip] += count * qty
			}
		}
	}

	result.LandCount = landCount
	result.TotalCards = totalCards + landCount
	if totalCards > 0 {
		result.AverageCMC = math.Round(totalCMC/float64(totalCards)*100) / 100
	}

	// Ideal land count based on format ratio
	result.IdealLandCount = int(math.Round(float64(params.deckSize) * params.idealLandRatio))

	// Build sorted distribution slice
	for cmc := 0; cmc <= 6; cmc++ {
		result.Distribution = append(result.Distribution, domain.CMCBucket{CMC: cmc, Count: buckets[cmc]})
	}

	// Generate suggestions
	result.Suggestions = generateManaSuggestions(result, params, landCount)

	return result
}

func isLandCard(card *domain.Card) bool {
	t := strings.ToLower(card.TypeLine)
	return strings.Contains(t, "land")
}

func generateManaSuggestions(analysis domain.ManaAnalysis, params formatParams, landCount int) []domain.ManaCurveSuggestion {
	var sug []domain.ManaCurveSuggestion

	// Land count check
	diff := landCount - analysis.IdealLandCount
	if diff > 2 {
		sug = append(sug, domain.ManaCurveSuggestion{
			Type:    "remove",
			CMC:     0,
			Reason:  "Land count is higher than the format ideal — consider cutting basic lands for more spells.",
			Urgency: urgency(diff, 4, 7),
		})
	} else if diff < -2 {
		sug = append(sug, domain.ManaCurveSuggestion{
			Type:    "add",
			CMC:     0,
			Reason:  "Land count is below the format ideal — mana issues may cause inconsistent draws.",
			Urgency: urgency(-diff, 4, 7),
		})
	}

	// High-CMC density check
	heavyCards := 0
	for _, b := range analysis.Distribution {
		if b.CMC >= params.maxCMC+1 {
			heavyCards += b.Count
		}
	}
	if analysis.TotalCards > 0 {
		heavyRatio := float64(heavyCards) / float64(analysis.TotalCards)
		if heavyRatio > 0.20 {
			sug = append(sug, domain.ManaCurveSuggestion{
				Type:    "remove",
				CMC:     params.maxCMC + 1,
				Reason:  "Too many high-CMC cards. Reducing the top end will improve the early game.",
				Urgency: urgency(int(heavyRatio*100), 25, 35),
			})
		}
	}

	// Average CMC check (format-specific thresholds)
	avgThresholdHigh := 3.2
	avgThresholdLow := 1.8
	if strings.Contains(analysis.Format, "commander") {
		avgThresholdHigh = 3.8
		avgThresholdLow = 2.5
	}
	if analysis.AverageCMC > avgThresholdHigh {
		sug = append(sug, domain.ManaCurveSuggestion{
			Type:    "remove",
			Reason:  fmt.Sprintf("Average CMC %.2f is high for %s. Swap some expensive spells for cheaper interaction.", analysis.AverageCMC, analysis.Format),
			Urgency: "moderate",
		})
	} else if analysis.AverageCMC < avgThresholdLow && analysis.TotalCards > 10 {
		sug = append(sug, domain.ManaCurveSuggestion{
			Type:    "add",
			Reason:  "Average CMC is very low — consider adding a few mid-game threats.",
			Urgency: "minor",
		})
	}

	return sug
}

func urgency(value, moderate, critical int) string {
	if value >= critical {
		return "critical"
	}
	if value >= moderate {
		return "moderate"
	}
	return "minor"
}

// countPips parses a Scryfall mana cost string like "{2}{W}{W}{U}" and returns
// the count of each coloured pip (W, U, B, R, G, C).
// Generic and Phyrexian mana symbols are not counted since they don't restrict colour.
func countPips(manaCost string) map[string]int {
	pips := make(map[string]int)
	// Walk through each {X} symbol in the mana cost string.
	i := 0
	for i < len(manaCost) {
		if manaCost[i] != '{' {
			i++
			continue
		}
		j := i + 1
		for j < len(manaCost) && manaCost[j] != '}' {
			j++
		}
		if j >= len(manaCost) {
			break
		}
		sym := strings.ToUpper(manaCost[i+1 : j])
		switch sym {
		case "W", "U", "B", "R", "G", "C":
			pips[sym]++
		case "W/P", "U/P", "B/P", "R/P", "G/P":
			// Phyrexian mana: count the colour component.
			pips[string(sym[0])]++
		default:
			// Hybrid symbols like {W/U}: count both colours.
			if len(sym) == 3 && sym[1] == '/' {
				c1, c2 := string(sym[0]), string(sym[2])
				if isColour(c1) {
					pips[c1]++
				}
				if isColour(c2) {
					pips[c2]++
				}
			}
		}
		i = j + 1
	}
	return pips
}

func isColour(s string) bool {
	return s == "W" || s == "U" || s == "B" || s == "R" || s == "G" || s == "C"
}
