package usecase

import (
	"math"
	"sort"
	"strings"

	"github.com/gigliofr/mana-wise/domain"
)

var colorCategoryCapabilities = map[string]map[domain.InteractionCategory]float64{
	"W": {
		domain.InteractionRemoval:    0.9,
		domain.InteractionCounter:    0.0,
		domain.InteractionDraw:       0.3,
		domain.InteractionRamp:       0.2,
		domain.InteractionProtection: 0.8,
		domain.InteractionDiscard:    0.0,
	},
	"U": {
		domain.InteractionRemoval:    0.5,
		domain.InteractionCounter:    1.0,
		domain.InteractionDraw:       1.0,
		domain.InteractionRamp:       0.1,
		domain.InteractionProtection: 0.3,
		domain.InteractionDiscard:    0.2,
	},
	"B": {
		domain.InteractionRemoval:    1.0,
		domain.InteractionCounter:    0.0,
		domain.InteractionDraw:       0.8,
		domain.InteractionRamp:       0.1,
		domain.InteractionProtection: 0.1,
		domain.InteractionDiscard:    1.0,
	},
	"R": {
		domain.InteractionRemoval:    0.9,
		domain.InteractionCounter:    0.0,
		domain.InteractionDraw:       0.5,
		domain.InteractionRamp:       0.2,
		domain.InteractionProtection: 0.1,
		domain.InteractionDiscard:    0.2,
	},
	"G": {
		domain.InteractionRemoval:    0.5,
		domain.InteractionCounter:    0.0,
		domain.InteractionDraw:       0.6,
		domain.InteractionRamp:       1.0,
		domain.InteractionProtection: 0.7,
		domain.InteractionDiscard:    0.0,
	},
}

func detectedDeckColors(cards []*domain.Card) []string {
	set := make(map[string]bool)
	for _, card := range cards {
		if card == nil {
			continue
		}
		for _, color := range card.ColorIdentity {
			addColorSymbol(set, color)
		}
		for _, color := range card.Colors {
			addColorSymbol(set, color)
		}
		if len(card.ColorIdentity) == 0 && len(card.Colors) == 0 {
			for _, color := range extractColorSymbols(card.ManaCost + " " + card.OracleText) {
				addColorSymbol(set, color)
			}
		}
	}
	colors := make([]string, 0, len(set))
	for color := range set {
		colors = append(colors, color)
	}
	sort.Strings(colors)
	return colors
}

func interactionColorMultipliers(deckColors []string) map[domain.InteractionCategory]float64 {
	result := map[domain.InteractionCategory]float64{
		domain.InteractionRemoval:    1.0,
		domain.InteractionCounter:    1.0,
		domain.InteractionDraw:       1.0,
		domain.InteractionRamp:       1.0,
		domain.InteractionProtection: 1.0,
		domain.InteractionDiscard:    1.0,
	}
	if len(deckColors) == 0 {
		return result
	}
	for category := range result {
		result[category] = 0.0
	}
	for _, color := range deckColors {
		caps, ok := colorCategoryCapabilities[color]
		if !ok {
			continue
		}
		for category, value := range caps {
			if value > result[category] {
				result[category] = value
			}
		}
	}
	return result
}

func applyInteractionColorMultipliers(idealMap map[domain.InteractionCategory]int, deckColors []string) map[domain.InteractionCategory]int {
	if len(deckColors) == 0 {
		return idealMap
	}
	multipliers := interactionColorMultipliers(deckColors)
	adjusted := make(map[domain.InteractionCategory]int, len(idealMap))
	for category, ideal := range idealMap {
		multiplier := multipliers[category]
		if multiplier <= 0 {
			adjusted[category] = 0
			continue
		}
		value := int(math.Round(float64(ideal) * multiplier))
		if ideal > 0 && value == 0 {
			value = 1
		}
		adjusted[category] = value
	}
	return adjusted
}

func deckSupportsCategory(deckColors []string, category domain.InteractionCategory) bool {
	if len(deckColors) == 0 {
		return true
	}
	return interactionColorMultipliers(deckColors)[category] > 0
}

func addColorSymbol(set map[string]bool, raw string) {
	color := strings.ToUpper(strings.TrimSpace(raw))
	if _, ok := colorCategoryCapabilities[color]; ok {
		set[color] = true
	}
}

func extractColorSymbols(text string) []string {
	upper := strings.ToUpper(text)
	set := make(map[string]bool)
	for _, symbol := range []string{"W", "U", "B", "R", "G"} {
		if strings.Contains(upper, "{"+symbol+"}") {
			set[symbol] = true
		}
	}
	colors := make([]string, 0, len(set))
	for color := range set {
		colors = append(colors, color)
	}
	sort.Strings(colors)
	return colors
}

func localizedDeckColors(deckColors []string, locale string) string {
	if len(deckColors) == 0 {
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(locale)), "it") {
			return "i tuoi colori"
		}
		return "your colors"
	}
	labels := map[string]struct{ it, en string }{
		"W": {"bianco", "white"},
		"U": {"blu", "blue"},
		"B": {"nero", "black"},
		"R": {"rosso", "red"},
		"G": {"verde", "green"},
	}
	parts := make([]string, 0, len(deckColors))
	for _, color := range deckColors {
		label, ok := labels[color]
		if !ok {
			continue
		}
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(locale)), "it") {
			parts = append(parts, label.it)
		} else {
			parts = append(parts, label.en)
		}
	}
	if len(parts) == 0 {
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(locale)), "it") {
			return "i tuoi colori"
		}
		return "your colors"
	}
	if strings.HasPrefix(strings.ToLower(strings.TrimSpace(locale)), "it") {
		return strings.Join(parts, "/")
	}
	return strings.Join(parts, "/")
}
