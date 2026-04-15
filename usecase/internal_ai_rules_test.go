package usecase

import (
	"strings"
	"testing"

	"github.com/gigliofr/mana-wise/domain"
)

func makeAnalysis(landCount, idealLand int, avgCMC float64, totalScore float64, breakdowns []domain.InteractionBreakdown) *domain.AnalysisResult {
	return &domain.AnalysisResult{
		Format: "commander",
		Mana: domain.ManaAnalysis{
			LandCount:      landCount,
			IdealLandCount: idealLand,
			AverageCMC:     avgCMC,
		},
		Interaction: domain.InteractionAnalysis{
			TotalScore: totalScore,
			Breakdowns: breakdowns,
		},
	}
}

func TestBuildInternalSuggestions_NilInput(t *testing.T) {
	result := BuildInternalSuggestions(nil)
	if result != "" {
		t.Errorf("expected empty string for nil input, got %q", result)
	}
}

func TestBuildInternalSuggestions_BalancedDeck(t *testing.T) {
	a := makeAnalysis(37, 37, 3.0, 75.0, nil)
	result := BuildInternalSuggestions(a)
	if result == "" {
		t.Error("expected fallback message for balanced deck, got empty string")
	}
	if !strings.Contains(result, "balanced") {
		t.Errorf("expected 'balanced' in output for a well-balanced deck, got %q", result)
	}
}

func TestBuildInternalSuggestions_TooFewLands(t *testing.T) {
	a := makeAnalysis(33, 37, 3.0, 75.0, nil) // delta = -4 (≤ -2)
	result := BuildInternalSuggestions(a)
	if !strings.Contains(result, "short by") {
		t.Errorf("expected land shortage message, got %q", result)
	}
}

func TestBuildInternalSuggestions_TooManyLands(t *testing.T) {
	a := makeAnalysis(42, 37, 3.0, 75.0, nil) // delta = +5 (≥ 3)
	result := BuildInternalSuggestions(a)
	if !strings.Contains(result, "above benchmark") {
		t.Errorf("expected excess land message, got %q", result)
	}
}

func TestBuildInternalSuggestions_HighCMC(t *testing.T) {
	a := makeAnalysis(37, 37, 4.0, 75.0, nil)
	result := BuildInternalSuggestions(a)
	if !strings.Contains(result, "MV") {
		t.Errorf("expected high mana value message, got %q", result)
	}
}

func TestBuildInternalSuggestions_LowInteraction(t *testing.T) {
	a := makeAnalysis(37, 37, 3.0, 30.0, nil) // TotalScore < 40
	result := BuildInternalSuggestions(a)
	if !strings.Contains(result, "interaction score") {
		t.Errorf("expected low interaction message, got %q", result)
	}
}

func TestBuildInternalSuggestions_MediumInteraction(t *testing.T) {
	a := makeAnalysis(37, 37, 3.0, 55.0, nil) // 40 <= TotalScore < 70
	result := BuildInternalSuggestions(a)
	if !strings.Contains(result, "unstable") {
		t.Errorf("expected medium interaction message, got %q", result)
	}
}

func TestBuildInternalSuggestions_BreakdownDeficit(t *testing.T) {
	breakdowns := []domain.InteractionBreakdown{
		{Category: domain.InteractionRemoval, Count: 2, Ideal: 7, Delta: -5},
	}
	a := makeAnalysis(37, 37, 3.0, 75.0, breakdowns)
	result := BuildInternalSuggestions(a)
	if !strings.Contains(strings.ToLower(result), "removal") {
		t.Errorf("expected removal deficit message, got %q", result)
	}
}

func TestBuildInternalSuggestions_MaxThreeSuggestions(t *testing.T) {
	breakdowns := []domain.InteractionBreakdown{
		{Category: domain.InteractionRemoval, Count: 1, Ideal: 7, Delta: -6},
		{Category: domain.InteractionCounter, Count: 1, Ideal: 5, Delta: -4},
	}
	// Also triggers: few lands + high CMC + low interaction
	a := makeAnalysis(33, 37, 4.1, 25.0, breakdowns)
	result := BuildInternalSuggestions(a)

	// Count numbered suggestions "1) ", "2) ", "3) " in output
	count := strings.Count(result, "\n") + 1
	if count > 3 {
		t.Errorf("expected at most 3 suggestions, got %d lines: %q", count, result)
	}
}

func TestBuildInternalSuggestions_WithManaSuggestion(t *testing.T) {
	a := makeAnalysis(37, 37, 3.0, 75.0, nil)
	a.Mana.Suggestions = []domain.ManaCurveSuggestion{
		{Type: "add", CMC: 2, Reason: "Add more 2-drops to improve curve.", Urgency: "critical"},
	}
	result := BuildInternalSuggestions(a)
	if !strings.Contains(result, "2-drops") {
		t.Errorf("expected mana suggestion in output, got %q", result)
	}
}

func TestBuildInternalSuggestionsLocalized_MonoGreenCounterFallbackStaysInColor(t *testing.T) {
	a := makeAnalysis(37, 37, 3.0, 75.0, []domain.InteractionBreakdown{
		{Category: domain.InteractionCounter, Count: 0, Ideal: 1, Delta: -1},
	})
	result := BuildInternalSuggestionsLocalized(a, "standard", "it", []*domain.Card{
		{ID: "elf", Name: "Llanowar Scout", TypeLine: "Creature - Elf Scout", ManaCost: "{G}", Colors: []string{"G"}, ColorIdentity: []string{"G"}},
		{ID: "beast", Name: "Huge Beast", TypeLine: "Creature - Beast", ManaCost: "{3}{G}", Colors: []string{"G"}, ColorIdentity: []string{"G"}},
	})
	if strings.Contains(result, "Negate") || strings.Contains(result, "Make Disappear") || strings.Contains(result, "Disdainful Stroke") {
		t.Fatalf("expected no blue counter examples for mono-green deck, got %q", result)
	}
	if !strings.Contains(result, "senza forzare splash blu") {
		t.Fatalf("expected in-color fallback wording for mono-green counter gap, got %q", result)
	}
}
