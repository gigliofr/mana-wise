package usecase_test

import (
	"testing"

	"github.com/manawise/api/domain"
	"github.com/manawise/api/usecase"
)

func makeInteractionCard(id, oracleText, typeL string) *domain.Card {
	return &domain.Card{ID: id, Name: id, TypeLine: typeL, OracleText: oracleText}
}

func TestAnalyzeInteraction_RemovalDetection(t *testing.T) {
	cards := []*domain.Card{
		makeInteractionCard("bolt", "Lightning Bolt deals 3 damage to any target.", "Instant"),
		makeInteractionCard("path", "Exile target creature.", "Instant"),
	}
	qtys := map[string]int{"bolt": 4, "path": 4}

	result := usecase.AnalyzeInteraction(cards, qtys, "modern")

	var removalCount int
	for _, bd := range result.Breakdowns {
		if bd.Category == domain.InteractionRemoval {
			removalCount = bd.Count
		}
	}
	// bolt: "deals 3 damage to any target" matches "damage to"; path: "exile target" matches "exile target".
	if removalCount != 8 {
		t.Errorf("expected 8 removal spells, got %d", removalCount)
	}
}

func TestAnalyzeInteraction_DrawDetection(t *testing.T) {
	cards := []*domain.Card{
		makeInteractionCard("preordain", "Scry 2, then draw a card.", "Sorcery"),
		makeInteractionCard("divination", "Draw two cards.", "Sorcery"),
	}
	qtys := map[string]int{"preordain": 4, "divination": 4}

	result := usecase.AnalyzeInteraction(cards, qtys, "modern")

	for _, bd := range result.Breakdowns {
		if bd.Category == domain.InteractionDraw {
			if bd.Count != 8 {
				t.Errorf("expected 8 draw spells, got %d", bd.Count)
			}
			return
		}
	}
	t.Error("draw breakdown not found")
}

func TestAnalyzeInteraction_TotalScoreRange(t *testing.T) {
	// Well-rounded deck should have score between 0 and 100.
	cards := []*domain.Card{
		makeInteractionCard("r1", "Destroy target creature.", "Instant"),
		makeInteractionCard("r2", "Counter target spell.", "Instant"),
		makeInteractionCard("r3", "Draw a card.", "Sorcery"),
		makeInteractionCard("r4", "Add {G}{G}. Search your library for a basic land.", "Sorcery"),
	}
	qtys := map[string]int{"r1": 6, "r2": 4, "r3": 6, "r4": 6}

	result := usecase.AnalyzeInteraction(cards, qtys, "modern")
	if result.TotalScore < 0 || result.TotalScore > 100 {
		t.Errorf("total score %f out of range [0,100]", result.TotalScore)
	}
}

func TestAnalyzeInteraction_CommanderWeights(t *testing.T) {
	// Commander needs more ramp — ensure ideal for ramp is higher than modern.
	cards := []*domain.Card{}
	qtys := map[string]int{}

	rampModern := usecase.AnalyzeInteraction(cards, qtys, "modern")
	rampCommander := usecase.AnalyzeInteraction(cards, qtys, "commander")

	var modernRampIdeal, commanderRampIdeal int
	for _, bd := range rampModern.Breakdowns {
		if bd.Category == domain.InteractionRamp {
			modernRampIdeal = bd.Ideal
		}
	}
	for _, bd := range rampCommander.Breakdowns {
		if bd.Category == domain.InteractionRamp {
			commanderRampIdeal = bd.Ideal
		}
	}
	if commanderRampIdeal <= modernRampIdeal {
		t.Errorf("commander ramp ideal (%d) should be higher than modern (%d)", commanderRampIdeal, modernRampIdeal)
	}
}

func TestAnalyzeInteraction_ArchetypeRampDetected(t *testing.T) {
	// 10 elf-ramp cards out of 22 non-land cards (>20% ramp ratio)
	cards := []*domain.Card{
		makeInteractionCard("elf", "{T}: Add {G}.", "Creature - Elf Druid"),
		makeInteractionCard("forest", "{T}: Add {G}.", "Basic Land - Forest"),
	}
	qtys := map[string]int{"elf": 10, "forest": 12}

	result := usecase.AnalyzeInteraction(cards, qtys, "standard")
	if result.Archetype != "ramp" {
		t.Fatalf("expected ramp archetype, got %q", result.Archetype)
	}
}

func TestAnalyzeInteraction_RampDeck_NoCounterOrDiscardSuggestions(t *testing.T) {
	cards := []*domain.Card{
		makeInteractionCard("elf", "{T}: Add {G}.", "Creature - Elf Druid"),
		makeInteractionCard("forest", "{T}: Add {G}.", "Basic Land - Forest"),
	}
	qtys := map[string]int{"elf": 10, "forest": 12}

	result := usecase.AnalyzeInteraction(cards, qtys, "standard")

	for _, s := range result.Suggestions {
		if len(s) >= 7 && s[:7] == "Your co" {
			t.Errorf("unexpected counter suggestion in ramp deck: %s", s)
		}
		if len(s) >= 9 && s[:9] == "Your disc" {
			t.Errorf("unexpected discard suggestion in ramp deck: %s", s)
		}
	}
}

func TestAnalyzeInteraction_AggroDeck_ArchetypeDetected(t *testing.T) {
	// Low-CMC burn deck, no ramp
	cards := []*domain.Card{
		makeInteractionCard("bolt", "Lightning Bolt deals 3 damage to any target.", "Instant"),
		makeInteractionCard("monk", "Haste.", "Creature - Human Monk"),
	}
	qtys := map[string]int{"bolt": 4, "monk": 8}

	result := usecase.AnalyzeInteraction(cards, qtys, "modern")
	if result.Archetype != "aggro" {
		t.Fatalf("expected aggro archetype, got %q", result.Archetype)
	}
}
