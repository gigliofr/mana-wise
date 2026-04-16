package usecase

import (
	"context"
	"strings"
	"testing"

	"github.com/gigliofr/mana-wise/domain"
	"github.com/gigliofr/mana-wise/infrastructure/llm"
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

func TestAISuggester_ExternalOnly_TypedNilConnector_ReturnsErrorNoPanic(t *testing.T) {
	var nilConnector *llm.Connector
	s := NewAISuggester(AIModeExternalOnly, nilConnector, nil, false)

	_, _, _, err := s.Suggest(context.Background(), "4 Lightning Bolt", "standard", "it", minimalAnalysis(), nil)
	if err == nil {
		t.Fatal("expected error when provider receiver is nil")
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

func TestAISuggester_HybridPreferExternal_PrimaryFails_SecondarySucceeds(t *testing.T) {
	primary := &stubProvider{
		provider: "openai",
		model:    "gpt-4o-mini",
		err:      assertErr("provider returned status code: 429"),
	}
	secondary := &stubProvider{
		provider:   "gemini",
		model:      "gemini-1.5-pro",
		suggestion: "1) CUT: X ADD: Y WHY: better curve",
	}

	s := NewAISuggester(AIModeHybridPreferExternal, primary, secondary, true)
	text, source, extErr, err := s.Suggest(context.Background(), "4 Lightning Bolt", "standard", "en", minimalAnalysis(), nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if extErr != nil {
		t.Fatalf("did not expect external warning when secondary succeeds, got: %v", extErr)
	}
	if !strings.Contains(source, "gemini") {
		t.Fatalf("expected secondary source label, got %q", source)
	}
	if strings.TrimSpace(text) == "" {
		t.Fatal("expected non-empty suggestions")
	}
}

func TestAISuggester_HybridPreferExternal_429FallbacksToInternalWhenExternalChainFails(t *testing.T) {
	primary := &stubProvider{
		provider: "openai",
		model:    "gpt-4o-mini",
		err:      assertErr("provider returned status code: 429"),
	}
	secondary := &stubProvider{
		provider: "gemini",
		model:    "gemini-1.5-pro",
		err:      assertErr("provider returned status code: 503"),
	}

	s := NewAISuggester(AIModeHybridPreferExternal, primary, secondary, true).WithFallbackPolicy([]int{429, 503}, true)
	text, source, extErr, err := s.Suggest(context.Background(), "4 Lightning Bolt", "standard", "it", minimalAnalysis(), nil)
	if err != nil {
		t.Fatalf("expected internal fallback, got error: %v", err)
	}
	if extErr == nil {
		t.Fatal("expected external warning when fallback to internal happens")
	}
	if source != "internal_rules" {
		t.Fatalf("expected internal source, got %q", source)
	}
	if strings.TrimSpace(text) == "" {
		t.Fatal("expected non-empty internal fallback suggestions")
	}
}

func TestAISuggester_HybridPreferExternal_400DoesNotFallbackWhenPolicyDisallows(t *testing.T) {
	primary := &stubProvider{
		provider: "openai",
		model:    "gpt-4o-mini",
		err:      assertErr("provider returned status code: 400"),
	}

	s := NewAISuggester(AIModeHybridPreferExternal, primary, nil, true).WithFallbackPolicy([]int{429, 503}, true)
	_, _, _, err := s.Suggest(context.Background(), "4 Lightning Bolt", "standard", "en", minimalAnalysis(), nil)
	if err == nil {
		t.Fatal("expected hard error when status is not in fallback policy")
	}
}

func TestAISuggester_FallbackPolicy_StatusCodes(t *testing.T) {
	s := NewAISuggester(AIModeHybridPreferExternal, nil, nil, true).WithFallbackPolicy([]int{503}, true)

	if s.shouldFallbackOnExternalError(context.DeadlineExceeded) == false {
		t.Fatal("expected timeout errors to fallback when enabled")
	}
	if s.shouldFallbackOnExternalError(assertErr("provider returned status code: 429")) {
		t.Fatal("did not expect fallback for 429 when only 503 is allowed")
	}
	if !s.shouldFallbackOnExternalError(assertErr("provider returned status code: 503")) {
		t.Fatal("expected fallback for configured status 503")
	}
}

func TestAISuggester_FallbackPolicy_TimeoutDisabled(t *testing.T) {
	s := NewAISuggester(AIModeHybridPreferExternal, nil, nil, true).WithFallbackPolicy([]int{429}, false)

	if s.shouldFallbackOnExternalError(assertErr("context deadline exceeded")) {
		t.Fatal("did not expect timeout fallback when disabled")
	}
	if !s.shouldFallbackOnExternalError(assertErr("provider quota exceeded (429)")) {
		t.Fatal("expected 429 fallback when configured")
	}
}

func assertErr(msg string) error {
	return &testErr{msg: msg}
}

type testErr struct {
	msg string
}

func (e *testErr) Error() string {
	return e.msg
}

type stubProvider struct {
	provider   string
	model      string
	suggestion string
	err        error
}

func (s *stubProvider) Suggestions(_ context.Context, _ string, _ string, _ string, _ string, _ *domain.AnalysisResult) (string, error) {
	if s.err != nil {
		return "", s.err
	}
	return s.suggestion, nil
}

func (s *stubProvider) Provider() string {
	return s.provider
}

func (s *stubProvider) Model() string {
	return s.model
}
