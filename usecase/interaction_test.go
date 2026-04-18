package usecase_test

import (
	"strings"
	"testing"

	"github.com/gigliofr/mana-wise/domain"
	"github.com/gigliofr/mana-wise/usecase"
)

func makeInteractionCard(id, oracleText, typeL string) *domain.Card {
	return &domain.Card{ID: id, Name: id, TypeLine: typeL, OracleText: oracleText}
}

func makeInteractionCardWithColors(id, oracleText, typeL string, colors ...string) *domain.Card {
	return &domain.Card{
		ID:            id,
		Name:          id,
		TypeLine:      typeL,
		OracleText:    oracleText,
		Colors:        colors,
		ColorIdentity: colors,
	}
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

func TestAnalyzeInteraction_MonoGreenDeck_DoesNotExpectCounters(t *testing.T) {
	cards := []*domain.Card{
		makeInteractionCardWithColors("elf", "{T}: Add {G}.", "Creature - Elf Druid", "G"),
		makeInteractionCardWithColors("beast", "Trample", "Creature - Beast", "G"),
	}
	qtys := map[string]int{"elf": 8, "beast": 12}

	result := usecase.AnalyzeInteraction(cards, qtys, "standard")

	for _, bd := range result.Breakdowns {
		if bd.Category == domain.InteractionCounter && bd.Ideal != 0 {
			t.Fatalf("expected mono-green counter ideal to be 0, got %d", bd.Ideal)
		}
	}

	for _, suggestion := range result.Suggestions {
		if strings.Contains(strings.ToLower(suggestion), "counter") {
			t.Fatalf("unexpected counter suggestion for mono-green deck: %s", suggestion)
		}
	}
}

func TestAnalyzeInteraction_Commander_NonGreenDeckKeepsRampFloor(t *testing.T) {
	cards := []*domain.Card{
		makeInteractionCardWithColors("wrb-1", "Deal 2 damage to any target.", "Instant", "W", "R", "B"),
		makeInteractionCardWithColors("wrb-2", "Draw a card.", "Sorcery", "W", "R", "B"),
	}
	qtys := map[string]int{"wrb-1": 20, "wrb-2": 20}

	result := usecase.AnalyzeInteraction(cards, qtys, "commander")

	var rampIdeal int
	for _, bd := range result.Breakdowns {
		if bd.Category == domain.InteractionRamp {
			rampIdeal = bd.Ideal
			break
		}
	}

	if rampIdeal < 8 {
		t.Fatalf("expected commander ramp ideal floor >= 8, got %d", rampIdeal)
	}
}

func TestAnalyzeInteraction_Commander_FloorsRaiseCoreTargets(t *testing.T) {
	cards := []*domain.Card{
		makeInteractionCardWithColors("aggro-1", "Haste", "Creature - Human Warrior", "R"),
		makeInteractionCardWithColors("aggro-2", "First strike", "Creature - Human Knight", "W"),
	}
	qtys := map[string]int{"aggro-1": 30, "aggro-2": 30}

	result := usecase.AnalyzeInteraction(cards, qtys, "commander")

	ideals := map[domain.InteractionCategory]int{}
	for _, bd := range result.Breakdowns {
		ideals[bd.Category] = bd.Ideal
	}

	if ideals[domain.InteractionRemoval] < 6 {
		t.Fatalf("expected commander removal floor >= 6, got %d", ideals[domain.InteractionRemoval])
	}
	if ideals[domain.InteractionDraw] < 6 {
		t.Fatalf("expected commander draw floor >= 6, got %d", ideals[domain.InteractionDraw])
	}
	if ideals[domain.InteractionProtection] < 2 {
		t.Fatalf("expected commander protection floor >= 2, got %d", ideals[domain.InteractionProtection])
	}
}
