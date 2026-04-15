package usecase

import (
	"testing"

	"github.com/gigliofr/mana-wise/domain"
)

func TestGetManaProducingColors_LlanowarElves(t *testing.T) {
	card := &domain.Card{
		Name:       "Llanowar Elves",
		OracleText: "{T}: Add {G}.",
	}
	colors := getManaProducingColors(card, map[string]bool{"G": true})
	if len(colors) != 1 || colors[0] != "G" {
		t.Errorf("Expected [G], got %v", colors)
	}
}

func TestGetManaProducingColors_DualMana(t *testing.T) {
	card := &domain.Card{
		Name:       "Archdruid",
		OracleText: "{T}: Add {W}, {U}, or {B}.",
	}
	colors := getManaProducingColors(card, map[string]bool{"W": true, "U": true, "B": true})
	if len(colors) != 3 {
		t.Errorf("Expected 3 colors, got %d: %v", len(colors), colors)
	}
}

func TestGetManaProducingColors_NoMana(t *testing.T) {
	card := &domain.Card{
		Name:       "Lightning Bolt",
		OracleText: "Lightning Bolt deals 3 damage to any target.",
	}
	colors := getManaProducingColors(card, map[string]bool{"R": true})
	if len(colors) != 0 {
		t.Errorf("Expected [], got %v", colors)
	}
}

func TestGetManaProducingColors_MultiColor(t *testing.T) {
	card := &domain.Card{
		Name:       "Utopia Sprawl",
		OracleText: "Enchant Forest\nWhenever a player plays a land, that player adds {G}.",
	}
	colors := getManaProducingColors(card, map[string]bool{"G": true})
	if len(colors) != 1 || colors[0] != "G" {
		t.Errorf("Expected [G], got %v", colors)
	}
}

func TestGetManaProducingColors_BlackGreen(t *testing.T) {
	card := &domain.Card{
		Name:       "Cabal Archon",
		OracleText: "{T}, Pay 1 life: Add {B} or {G}.",
	}
	colors := getManaProducingColors(card, map[string]bool{"B": true, "G": true})
	if len(colors) != 2 {
		t.Errorf("Expected 2 colors, got %d: %v", len(colors), colors)
	}
}

func TestAnalyzeManaCurve_CountsCreatureMana(t *testing.T) {
	// Create a test deck with a mana-producing creature and some spells
	cards := []*domain.Card{
		{
			ID:         "elf1",
			Name:       "Llanowar Elves",
			CMC:        1,
			TypeLine:   "Creature — Elf Druid",
			OracleText: "{T}: Add {G}.",
			Colors:     []string{"G"},
		},
		{
			ID:         "forest1",
			Name:       "Forest",
			CMC:        0,
			TypeLine:   "Basic Land — Forest",
			OracleText: "{T}: Add {G}.",
			Colors:     []string{},
		},
	}

	quantities := map[string]int{
		"elf1":     2,
		"forest1":  3,
	}

	result := AnalyzeManaCurve(cards, quantities, "modern")

	if result.CurrentTotalSources != 5 {
		t.Errorf("Expected total current sources to be 5 (2 Elves + 3 Forests), got %d", result.CurrentTotalSources)
	}
}
