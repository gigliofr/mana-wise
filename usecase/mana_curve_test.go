package usecase_test

import (
	"testing"

	"github.com/manawise/api/domain"
	"github.com/manawise/api/usecase"
)

func makeCard(id string, cmc float64, typeL string, colors ...string) *domain.Card {
	return &domain.Card{
		ID:       id,
		Name:     id,
		CMC:      cmc,
		TypeLine: typeL,
		Colors:   colors,
	}
}

func TestAnalyzeManaCurve_BasicModern(t *testing.T) {
	// A typical 60-card modern-ish deck: 24 lands + 36 spells.
	cards := []*domain.Card{}
	qtys := map[string]int{}

	// Lands
	for i := 0; i < 24; i++ {
		id := "land-" + string(rune('A'+i))
		cards = append(cards, makeCard(id, 0, "Basic Land"))
		qtys[id] = 1
	}
	// 1-drop spells ×12
	for i := 0; i < 12; i++ {
		id := "one-" + string(rune('A'+i))
		cards = append(cards, makeCard(id, 1, "Instant", "R"))
		qtys[id] = 1
	}
	// 2-drop spells ×12
	for i := 0; i < 12; i++ {
		id := "two-" + string(rune('A'+i))
		cards = append(cards, makeCard(id, 2, "Creature", "W"))
		qtys[id] = 1
	}
	// 3-drop ×8, 4-drop ×4
	for i := 0; i < 8; i++ {
		id := "three-" + string(rune('A'+i))
		cards = append(cards, makeCard(id, 3, "Sorcery", "U"))
		qtys[id] = 1
	}
	for i := 0; i < 4; i++ {
		id := "four-" + string(rune('A'+i))
		cards = append(cards, makeCard(id, 4, "Creature", "G"))
		qtys[id] = 1
	}

	result := usecase.AnalyzeManaCurve(cards, qtys, "modern")

	if result.Format != "modern" {
		t.Errorf("expected format 'modern', got %q", result.Format)
	}
	if result.LandCount != 24 {
		t.Errorf("expected 24 lands, got %d", result.LandCount)
	}
	if result.TotalCards != 60 {
		t.Errorf("expected 60 total cards, got %d", result.TotalCards)
	}
	if result.AverageCMC <= 0 {
		t.Error("average CMC should be positive")
	}
	if len(result.Distribution) != 7 {
		t.Errorf("expected 7 CMC buckets (0–6+), got %d", len(result.Distribution))
	}
}

func TestAnalyzeManaCurve_IdealLandCount(t *testing.T) {
	cards := []*domain.Card{makeCard("l1", 0, "Basic Land")}
	qtys := map[string]int{"l1": 1}

	tests := []struct {
		format   string
		wantLow  int
		wantHigh int
	}{
		{"modern", 21, 25},
		{"commander", 35, 40},
		{"standard", 22, 26},
	}

	for _, tt := range tests {
		result := usecase.AnalyzeManaCurve(cards, qtys, tt.format)
		if result.IdealLandCount < tt.wantLow || result.IdealLandCount > tt.wantHigh {
			t.Errorf("format=%s: ideal land count %d not in [%d,%d]",
				tt.format, result.IdealLandCount, tt.wantLow, tt.wantHigh)
		}
	}
}

func TestAnalyzeManaCurve_LandSuggestion(t *testing.T) {
	// Deck with too few lands (only 10 for a 60-card modern deck).
	cards := []*domain.Card{}
	qtys := map[string]int{}
	for i := 0; i < 10; i++ {
		id := "land-" + string(rune('A'+i))
		cards = append(cards, makeCard(id, 0, "Basic Land"))
		qtys[id] = 1
	}
	for i := 0; i < 50; i++ {
		id := "spell-" + string(rune('A'+i%26)) + string(rune('a'+i/26))
		cards = append(cards, makeCard(id, 2, "Instant", "U"))
		qtys[id] = 1
	}

	result := usecase.AnalyzeManaCurve(cards, qtys, "modern")
	if len(result.Suggestions) == 0 {
		t.Error("expected at least one suggestion for a deck with too few lands")
	}
}

func TestAnalyzeManaCurve_UnknownFormatFallsBack(t *testing.T) {
	cards := []*domain.Card{makeCard("land", 0, "Basic Land")}
	qtys := map[string]int{"land": 1}
	result := usecase.AnalyzeManaCurve(cards, qtys, "freeform")
	// Should not panic and should return some ideal.
	if result.IdealLandCount == 0 {
		t.Error("expected non-zero ideal land count even for unknown format")
	}
}
