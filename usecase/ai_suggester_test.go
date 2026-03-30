package usecase

import (
	"context"
	"strings"
	"testing"

	"github.com/manawise/api/domain"
)

// minimalAnalysis returns a non-nil AnalysisResult with balanced metrics
// so that BuildInternalSuggestions returns the "balanced" fallback message.
func minimalAnalysis() *domain.AnalysisResult {
	return &domain.AnalysisResult{
		Format: "commander",
		Mana: domain.ManaAnalysis{
			LandCount:      37,
			IdealLandCount: 37,
			AverageCMC:     3.0,
		},
		Interaction: domain.InteractionAnalysis{
			TotalScore: 75.0,
		},
	}
}

// ---- Mode: internal_only ----

func TestAISuggester_InternalOnly_ReturnsInternalSource(t *testing.T) {
	s := NewAISuggester(AIModeInternalOnly, nil, nil, true)
	text, source, _, err := s.Suggest(context.Background(), "4 Lightning Bolt", "standard", "it", minimalAnalysis(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if source != "internal_rules" {
		t.Errorf("expected source 'internal_rules', got %q", source)
	}
	if text == "" {
		t.Error("expected non-empty suggestion text")
	}
}

func TestAISuggester_InternalOnly_DisabledReturnsError(t *testing.T) {
	s := NewAISuggester(AIModeInternalOnly, nil, nil, false)
	_, _, _, err := s.Suggest(context.Background(), "4 Lightning Bolt", "standard", "it", minimalAnalysis(), nil)
	if err == nil {
		t.Error("expected error when internal rules disabled, got nil")
	}
}

// ---- Mode: external_only, no providers, internal enabled → falls back to internal ----

func TestAISuggester_ExternalOnly_NoProviders_InternalEnabled_FallsBack(t *testing.T) {
	// tryExternalChain falls through to tryInternal when internalEnable=true and no external
	// provider is configured. This is the defined last-resort behaviour.
	s := NewAISuggester(AIModeExternalOnly, nil, nil, true)
	_, source, _, err := s.Suggest(context.Background(), "4 Lightning Bolt", "standard", "it", minimalAnalysis(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if source != "internal_rules" {
		t.Errorf("expected 'internal_rules' fallback, got %q", source)
	}
}

// ---- Mode: external_only, no providers, internal disabled → should fail ----

func TestAISuggester_ExternalOnly_NoProviders_InternalDisabled_ReturnsError(t *testing.T) {
	s := NewAISuggester(AIModeExternalOnly, nil, nil, false)
	_, _, _, err := s.Suggest(context.Background(), "4 Lightning Bolt", "standard", "it", minimalAnalysis(), nil)
	if err == nil {
		t.Error("expected error when no providers and internal rules disabled")
	}
}

// ---- Mode: hybrid_prefer_external, no providers → falls back to internal ----

func TestAISuggester_HybridPreferExternal_NoProviders_FallsBackToInternal(t *testing.T) {
	s := NewAISuggester(AIModeHybridPreferExternal, nil, nil, true)
	text, source, _, err := s.Suggest(context.Background(), "4 Lightning Bolt", "standard", "it", minimalAnalysis(), nil)
	if err != nil {
		t.Fatalf("expected fallback to internal, got error: %v", err)
	}
	if source != "internal_rules" {
		t.Errorf("expected source 'internal_rules' after fallback, got %q", source)
	}
	if text == "" {
		t.Error("expected non-empty fallback text")
	}
}

func TestAISuggester_HybridPreferExternal_NoProviders_InternalDisabled_ReturnsError(t *testing.T) {
	s := NewAISuggester(AIModeHybridPreferExternal, nil, nil, false)
	_, _, _, err := s.Suggest(context.Background(), "4 Lightning Bolt", "standard", "it", minimalAnalysis(), nil)
	if err == nil {
		t.Error("expected error when no providers and internal disabled")
	}
}

// ---- Mode: hybrid_prefer_internal, no providers → internal is used, no error ----

func TestAISuggester_HybridPreferInternal_UsesInternal(t *testing.T) {
	s := NewAISuggester(AIModeHybridPreferInternal, nil, nil, true)
	text, source, _, err := s.Suggest(context.Background(), "4 Lightning Bolt", "standard", "it", minimalAnalysis(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if source != "internal_rules" {
		t.Errorf("expected 'internal_rules', got %q", source)
	}
	if text == "" {
		t.Error("expected non-empty suggestion text")
	}
}

// ---- Unknown mode defaults to hybrid_prefer_external ----

func TestAISuggester_UnknownMode_DefaultsToHybridPreferExternal(t *testing.T) {
	s := NewAISuggester("garbage_mode", nil, nil, true)
	// No providers → should fall back to internal (hybrid_prefer_external behaviour)
	_, source, _, err := s.Suggest(context.Background(), "4 Lightning Bolt", "standard", "it", minimalAnalysis(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if source != "internal_rules" {
		t.Errorf("expected 'internal_rules' for default mode fallback, got %q", source)
	}
}

// ---- Nil analysis input ----

func TestAISuggester_InternalOnly_NilAnalysis_ReturnsError(t *testing.T) {
	s := NewAISuggester(AIModeInternalOnly, nil, nil, true)
	_, _, _, err := s.Suggest(context.Background(), "", "standard", "it", nil, nil)
	// BuildInternalSuggestions returns "" for nil → tryInternal returns error
	if err == nil {
		t.Error("expected error for nil analysis input")
	}
}

func TestAppendDeterministicCoachingFooter_AppendsForExternalSource(t *testing.T) {
	a := minimalAnalysis()
	a.Mana.CurrentTotalSources = 35
	a.Mana.TargetTotalSources = 37
	a.Mana.ManaScrewChance = 21.3
	a.Mana.ManaFloodChance = 12.7
	a.Mana.SweetSpotChance = 66.0
	a.Mana.SourceRequirements = []domain.ColorSourceRequirement{{Color: "R", Required: 12, Current: 10, Gap: 2}}
	a.Interaction.Archetype = "aggro"

	base := "1) Keep pressure high."
	out := appendDeterministicCoachingFooter(base, "en", a, "gemini:gemini-pro-latest")

	if !strings.Contains(out, "Rule-check:") {
		t.Fatalf("expected Rule-check footer, got: %s", out)
	}
	if !strings.Contains(out, "Lands 37/37") {
		t.Fatalf("expected lands summary in footer, got: %s", out)
	}
}

func TestAppendDeterministicCoachingFooter_DoesNotAppendForInternalSource(t *testing.T) {
	out := appendDeterministicCoachingFooter("1) Internal suggestion", "it", minimalAnalysis(), "internal_rules")
	if strings.Contains(out, "Controllo regole:") || strings.Contains(out, "Rule-check:") {
		t.Fatalf("did not expect coaching footer for internal source, got: %s", out)
	}
}

func TestBuildDeterministicCoachingFooter_ItalianLocale(t *testing.T) {
	a := minimalAnalysis()
	a.Interaction.Archetype = "control"
	footer := buildDeterministicCoachingFooter("it", a)
	if !strings.Contains(footer, "Controllo regole:") {
		t.Fatalf("expected italian footer label, got: %s", footer)
	}
}
