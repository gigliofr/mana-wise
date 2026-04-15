package usecase

import (
	"fmt"
	"math"
	"strings"

	"github.com/gigliofr/mana-wise/domain"
)

// interactionKeywords maps category to oracle-text/type keywords that identify the card.
var interactionKeywords = map[domain.InteractionCategory][]string{
	domain.InteractionRemoval: {
		// Hard removal
		"destroy target", "exile target",
		// Bounce / temporary removal
		"return target creature to its owner's hand",
		"return target nonland permanent to its owner's hand",
		// Damage-based removal: "deals 3 damage to any target" (Bolt/Shock), fixed damage to creature
		"damage to any target", "damage to target creature", "damage to each creature",
		"deals damage equal to",
		// Board wipes
		"destroy all creatures", "exile all creatures",
		"destroy all nonland permanents", "each creature gets -",
		// Fight
		"fights target", "fights another target", "target creature fights",
		// -N/-N that usually kills
		"-1/-1 counter", "-2/-2", "-3/-3", "-4/-4", "-x/-x",
		// Tap-locking (pseudo-removal)
		"doesn't untap", "tap all",
	},
	domain.InteractionCounter: {
		"counter target spell", "counter any target spell",
		"counter target activated", "counter target triggered",
		"counter target creature spell", "counter target noncreature",
		"counter target instant", "counter target sorcery",
	},
	domain.InteractionDraw: {
		// Actual card draw
		"draw a card", "draw two cards", "draw three cards",
		"draw x cards", "draw that many cards",
		// Cantrip patterns
		"draw cards equal", "draw cards for each",
		// Investigate / Clue tokens give card draw
		"investigate",
		// Looter / replace effects
		"draw a card, then discard", "draw, then discard",
	},
	domain.InteractionRamp: {
		// Land fetch
		"search your library for a basic land",
		"search your library for a land",
		"land card onto the battlefield",
		"put a land card",
		// Mana acceleration
		"add {w}", "add {u}", "add {b}", "add {r}", "add {g}",
		"add mana of any color", "add one mana of any",
		"add two mana of any", "add {c}",
		// Mana artifacts / dorks (+1 mana per use)
		"{t}: add {",
		// Treasure/food create more colored mana
		"create a treasure token", "create two treasure",
	},
	domain.InteractionProtection: {
		"hexproof", "indestructible", "protection from",
		"regenerate", "shroud",
		"can't be countered", "can't be the target",
		"phase out",
	},
	domain.InteractionDiscard: {
		"discard a card", "discard two cards", "discard three cards",
		"each player discards", "discard their hand",
		"discard x cards",
	},
}

// formatInteractionIdeals defines ideal card counts per category per format.
// Index order: removal, counter, draw, ramp, protection, discard.
type interactionIdeal struct {
	Removal    int
	Counter    int
	Draw       int
	Ramp       int
	Protection int
	Discard    int
}

var formatInteractionIdeals = map[string]interactionIdeal{
	"commander": {Removal: 10, Counter: 5, Draw: 10, Ramp: 12, Protection: 4, Discard: 2},
	"modern":    {Removal: 8, Counter: 4, Draw: 6, Ramp: 4, Protection: 3, Discard: 4},
	"pioneer":   {Removal: 8, Counter: 4, Draw: 6, Ramp: 4, Protection: 2, Discard: 4},
	"legacy":    {Removal: 8, Counter: 6, Draw: 8, Ramp: 4, Protection: 3, Discard: 4},
	"vintage":   {Removal: 6, Counter: 8, Draw: 8, Ramp: 6, Protection: 3, Discard: 4},
	"standard":  {Removal: 8, Counter: 4, Draw: 6, Ramp: 4, Protection: 2, Discard: 3},
	"pauper":    {Removal: 8, Counter: 4, Draw: 6, Ramp: 4, Protection: 2, Discard: 4},
}

// categoryWeights defines the scoring weight for each category.
var categoryWeights = map[domain.InteractionCategory]float64{
	domain.InteractionRemoval:    1.5,
	domain.InteractionCounter:    1.2,
	domain.InteractionDraw:       1.3,
	domain.InteractionRamp:       1.4,
	domain.InteractionProtection: 0.8,
	domain.InteractionDiscard:    0.7,
}

// archetypeMultipliers scales format base ideals per detected archetype.
// A 0.0 value means the category is not expected for this archetype and
// suggestions for it will be suppressed.
var archetypeMultipliers = map[domain.DeckArchetype]map[domain.InteractionCategory]float64{
	domain.ArchetypeAggro: {
		domain.InteractionRemoval:    0.5,
		domain.InteractionCounter:    0.0,
		domain.InteractionDraw:       0.25,
		domain.InteractionRamp:       0.25,
		domain.InteractionProtection: 0.75,
		domain.InteractionDiscard:    0.75,
	},
	domain.ArchetypeControl: {
		domain.InteractionRemoval:    1.0,
		domain.InteractionCounter:    1.5,
		domain.InteractionDraw:       1.5,
		domain.InteractionRamp:       0.5,
		domain.InteractionProtection: 1.0,
		domain.InteractionDiscard:    0.75,
	},
	domain.ArchetypeRamp: {
		domain.InteractionRemoval:    0.5,
		domain.InteractionCounter:    0.25,
		domain.InteractionDraw:       0.5,
		domain.InteractionRamp:       2.0,
		domain.InteractionProtection: 0.5,
		domain.InteractionDiscard:    0.0,
	},
	domain.ArchetypeMidrange: {
		domain.InteractionRemoval:    1.0,
		domain.InteractionCounter:    0.75,
		domain.InteractionDraw:       0.75,
		domain.InteractionRamp:       0.75,
		domain.InteractionProtection: 0.75,
		domain.InteractionDiscard:    0.75,
	},
	domain.ArchetypeUnknown: {
		domain.InteractionRemoval:    1.0,
		domain.InteractionCounter:    1.0,
		domain.InteractionDraw:       1.0,
		domain.InteractionRamp:       1.0,
		domain.InteractionProtection: 1.0,
		domain.InteractionDiscard:    1.0,
	},
}

// detectArchetype infers deck play style from interaction distribution and avgCMC.
func detectArchetype(counts map[domain.InteractionCategory]int, avgCMC float64, totalNonLandCards int) domain.DeckArchetype {
	if totalNonLandCards == 0 {
		return domain.ArchetypeUnknown
	}
	ramp := counts[domain.InteractionRamp]
	counter := counts[domain.InteractionCounter]
	draw := counts[domain.InteractionDraw]
	rampRatio := float64(ramp) / float64(totalNonLandCards)
	if rampRatio >= 0.20 || ramp >= 8 {
		return domain.ArchetypeRamp
	}
	if counter >= 4 || draw >= 8 {
		return domain.ArchetypeControl
	}
	if avgCMC < 2.2 {
		return domain.ArchetypeAggro
	}
	return domain.ArchetypeMidrange
}

// applyArchetypeMultipliers scales base ideals with archetype multipliers.
func applyArchetypeMultipliers(base interactionIdeal, archetype domain.DeckArchetype) map[domain.InteractionCategory]int {
	mults, ok := archetypeMultipliers[archetype]
	if !ok {
		mults = archetypeMultipliers[domain.ArchetypeUnknown]
	}
	apply := func(b int, cat domain.InteractionCategory) int {
		v := int(math.Round(float64(b) * mults[cat]))
		if v < 0 {
			return 0
		}
		return v
	}
	return map[domain.InteractionCategory]int{
		domain.InteractionRemoval:    apply(base.Removal, domain.InteractionRemoval),
		domain.InteractionCounter:    apply(base.Counter, domain.InteractionCounter),
		domain.InteractionDraw:       apply(base.Draw, domain.InteractionDraw),
		domain.InteractionRamp:       apply(base.Ramp, domain.InteractionRamp),
		domain.InteractionProtection: apply(base.Protection, domain.InteractionProtection),
		domain.InteractionDiscard:    apply(base.Discard, domain.InteractionDiscard),
	}
}

// AnalyzeInteraction performs the deterministic interaction-density analysis.
func AnalyzeInteraction(cards []*domain.Card, quantities map[string]int, format string) domain.InteractionAnalysis {
	baseIdeals := getInteractionIdeals(format)

	// Count cards per category; also collect avgCMC for archetype detection.
	counts := make(map[domain.InteractionCategory]int)
	totalNonLandCards := 0
	totalCMC := 0.0
	for _, card := range cards {
		qty := quantities[card.ID]
		if qty == 0 {
			qty = 1
		}
		if isLandCard(card) {
			continue
		}

		totalNonLandCards += qty
		totalCMC += card.CMC * float64(qty)

		text := strings.ToLower(card.OracleText + " " + card.TypeLine)
		for cat, keywords := range interactionKeywords {
			for _, kw := range keywords {
				if strings.Contains(text, kw) {
					counts[cat] += qty
					break // count card once per category
				}
			}
		}
	}

	var avgCMC float64
	if totalNonLandCards > 0 {
		avgCMC = totalCMC / float64(totalNonLandCards)
	}

	archetype := detectArchetype(counts, avgCMC, totalNonLandCards)
	deckColors := detectedDeckColors(cards)
	idealMap := applyInteractionColorMultipliers(applyArchetypeMultipliers(baseIdeals, archetype), deckColors)
	mults := archetypeMultipliers[archetype]

	categories := []domain.InteractionCategory{
		domain.InteractionRemoval,
		domain.InteractionCounter,
		domain.InteractionDraw,
		domain.InteractionRamp,
		domain.InteractionProtection,
		domain.InteractionDiscard,
	}

	var totalScore float64
	var breakdowns []domain.InteractionBreakdown
	var suggestions []string

	for _, cat := range categories {
		count := counts[cat]
		ideal := idealMap[cat]
		weight := categoryWeights[cat]

		var score float64
		if ideal > 0 {
			ratio := float64(count) / float64(ideal)
			if ratio > 1 {
				ratio = 1
			}
			score = ratio * weight
		} else {
			// Category not expected for this archetype — award full weight.
			score = weight
		}

		breakdowns = append(breakdowns, domain.InteractionBreakdown{
			Category: cat,
			Count:    count,
			Weight:   weight,
			Score:    score,
			Ideal:    ideal,
			Delta:    count - ideal,
		})
		totalScore += score

		// Suggest only for relevant categories that the deck can reasonably support.
		if ideal > 0 && mults[cat] > 0.0 && count < ideal/2 {
			suggestions = append(suggestions, fmt.Sprintf(
				"Your %s package (%d cards) is below ideal for a %s deck (%d expected). Consider adding more.",
				cat, count, archetype, ideal,
			))
		}
	}

	// Normalize total score to 0–100.
	maxPossible := 0.0
	for _, w := range categoryWeights {
		maxPossible += w
	}
	if maxPossible > 0 {
		totalScore = (totalScore / maxPossible) * 100
	}

	return domain.InteractionAnalysis{
		Format:      format,
		Archetype:   string(archetype),
		TotalScore:  round(totalScore, 1),
		Breakdowns:  breakdowns,
		Suggestions: suggestions,
	}
}

func getInteractionIdeals(format string) interactionIdeal {
	if v, ok := formatInteractionIdeals[strings.ToLower(format)]; ok {
		return v
	}
	return formatInteractionIdeals["modern"]
}

func round(x float64, decimals int) float64 {
	p := 1.0
	for i := 0; i < decimals; i++ {
		p *= 10
	}
	return float64(int(x*p+0.5)) / p
}
