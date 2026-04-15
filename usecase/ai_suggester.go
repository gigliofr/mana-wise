package usecase

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/gigliofr/mana-wise/domain"
	"github.com/gigliofr/mana-wise/infrastructure/llm"
)

const (
	AIModeExternalOnly         = "external_only"
	AIModeInternalOnly         = "internal_only"
	AIModeHybridPreferExternal = "hybrid_prefer_external"
	AIModeHybridPreferInternal = "hybrid_prefer_internal"
)

// AISuggester orchestrates external LLM providers and internal rule-based suggestions.
type AISuggester struct {
	mode           string
	primary        *llm.Connector
	secondary      *llm.Connector
	internalEnable bool
	fallbackStatus map[int]bool
	fallbackOnTimeout bool
}

// NewAISuggester creates an AI suggester with runtime routing rules.
func NewAISuggester(mode string, primary, secondary *llm.Connector, internalEnable bool) *AISuggester {
	mode = strings.ToLower(strings.TrimSpace(mode))
	switch mode {
	case AIModeExternalOnly, AIModeInternalOnly, AIModeHybridPreferExternal, AIModeHybridPreferInternal:
	default:
		mode = AIModeHybridPreferExternal
	}
	defaultFallbackStatus := map[int]bool{429: true, 500: true, 502: true, 503: true, 504: true}
	return &AISuggester{
		mode:           mode,
		primary:        primary,
		secondary:      secondary,
		internalEnable: internalEnable,
		fallbackStatus: defaultFallbackStatus,
		fallbackOnTimeout: true,
	}
}

// WithFallbackPolicy configures which external failures should trigger internal fallback.
func (s *AISuggester) WithFallbackPolicy(statusCodes []int, fallbackOnTimeout bool) *AISuggester {
	if s == nil {
		return s
	}
	statusMap := make(map[int]bool, len(statusCodes))
	for _, code := range statusCodes {
		statusMap[code] = true
	}
	s.fallbackStatus = statusMap
	s.fallbackOnTimeout = fallbackOnTimeout
	return s
}

// Suggest returns AI suggestions, source, external-provider warning, and a hard error.
// externalErr is non-nil when an LLM call was attempted but failed and internal rules were used as fallback.
// cards is the resolved card slice (may be nil; used for card-name-specific suggestions).
func (s *AISuggester) Suggest(ctx context.Context, decklist, format, locale string, analysis *domain.AnalysisResult, cards []*domain.Card) (text, source string, externalErr, err error) {
	hash := llm.HashDecklist(decklist, format)

	switch s.mode {
	case AIModeInternalOnly:
		text, source, err = s.tryInternal(format, locale, analysis, cards)
		return
	case AIModeHybridPreferInternal:
		if text, source, err = s.tryInternal(format, locale, analysis, cards); err == nil {
			return
		}
		text, source, err = s.tryExternalChain(ctx, hash, decklist, format, locale, analysis, cards)
		if err == nil {
			text, source = s.enforceLocale(format, locale, analysis, cards, text, source)
			text = appendDeterministicCoachingFooter(text, locale, analysis, source)
		}
		return
	case AIModeExternalOnly:
		text, source, err = s.tryExternalChain(ctx, hash, decklist, format, locale, analysis, cards)
		if err == nil {
			text, source = s.enforceLocale(format, locale, analysis, cards, text, source)
			text = appendDeterministicCoachingFooter(text, locale, analysis, source)
		}
		return
	default: // hybrid_prefer_external
		var extErr error
		text, source, extErr = s.tryExternalChain(ctx, hash, decklist, format, locale, analysis, cards)
		if extErr == nil {
			text, source = s.enforceLocale(format, locale, analysis, cards, text, source)
			text = appendDeterministicCoachingFooter(text, locale, analysis, source)
			return
		}
		if !s.internalEnable {
			err = extErr
			return
		}
		if !s.shouldFallbackOnExternalError(extErr) {
			err = extErr
			return
		}
		var internalErr error
		text, source, internalErr = s.tryInternal(format, locale, analysis, cards)
		if internalErr != nil {
			err = extErr
			return
		}
		// Internal fallback succeeded; surface the LLM error as a warning.
		externalErr = extErr
		return
	}
}

func (s *AISuggester) shouldFallbackOnExternalError(err error) bool {
	if err == nil {
		return false
	}
	lower := strings.ToLower(err.Error())

	if s.fallbackOnTimeout {
		if strings.Contains(lower, "context deadline exceeded") || strings.Contains(lower, "timeout") || strings.Contains(lower, "timed out") {
			return true
		}
	}

	if strings.Contains(lower, "resource_exhausted") || strings.Contains(lower, "quota") || strings.Contains(lower, "too many requests") {
		return s.fallbackStatus[429]
	}

	if strings.Contains(lower, "service unavailable") || strings.Contains(lower, "bad gateway") || strings.Contains(lower, "gateway timeout") || strings.Contains(lower, "provider unavailable") {
		for _, code := range []int{503, 502, 504} {
			if s.fallbackStatus[code] {
				return true
			}
		}
	}

	for _, code := range extractStatusCodesFromError(err.Error()) {
		if s.fallbackStatus[code] {
			return true
		}
	}

	return false
}

func extractStatusCodesFromError(msg string) []int {
	re := regexp.MustCompile(`\b([1-5]\d\d)\b`)
	matches := re.FindAllStringSubmatch(msg, -1)
	if len(matches) == 0 {
		return nil
	}
	seen := map[int]bool{}
	out := make([]int, 0, len(matches))
	for _, m := range matches {
		if len(m) != 2 {
			continue
		}
		code, err := strconv.Atoi(m[1])
		if err != nil || seen[code] {
			continue
		}
		seen[code] = true
		out = append(out, code)
	}
	return out
}

func (s *AISuggester) tryExternalChain(ctx context.Context, hash, decklist, format, locale string, analysis *domain.AnalysisResult, cards []*domain.Card) (string, string, error) {
	if s.primary != nil {
		text, err := s.primary.Suggestions(ctx, hash, decklist, format, locale, analysis)
		if err == nil && strings.TrimSpace(text) != "" {
			return text, sourceLabel(s.primary), nil
		}
		if s.secondary == nil {
			if err != nil {
				return "", "", err
			}
			return "", "", fmt.Errorf("primary provider returned empty suggestions")
		}
	}

	if s.secondary != nil {
		text, err := s.secondary.Suggestions(ctx, hash, decklist, format, locale, analysis)
		if err == nil && strings.TrimSpace(text) != "" {
			return text, sourceLabel(s.secondary), nil
		}
		if err != nil {
			return "", "", err
		}
		return "", "", fmt.Errorf("secondary provider returned empty suggestions")
	}

	if s.internalEnable {
		return s.tryInternal(format, locale, analysis, cards)
	}
	return "", "", fmt.Errorf("no AI provider configured")
}

func (s *AISuggester) tryInternal(format, locale string, analysis *domain.AnalysisResult, cards []*domain.Card) (string, string, error) {
	if !s.internalEnable {
		return "", "", fmt.Errorf("internal rules disabled")
	}
	text := BuildInternalSuggestionsLocalized(analysis, format, locale, cards)
	if strings.TrimSpace(text) == "" {
		return "", "", fmt.Errorf("internal rules returned empty suggestions")
	}
	return text, "internal_rules", nil
}

func sourceLabel(c *llm.Connector) string {
	if c == nil {
		return ""
	}
	provider := strings.TrimSpace(c.Provider())
	if provider == "" {
		provider = "external"
	}
	model := strings.TrimSpace(c.Model())
	if model == "" {
		return provider
	}
	return provider + ":" + model
}

func (s *AISuggester) enforceLocale(format, locale string, analysis *domain.AnalysisResult, cards []*domain.Card, text, source string) (string, string) {
	locale = strings.ToLower(strings.TrimSpace(locale))
	if strings.HasPrefix(locale, "it") && seemsEnglishSuggestion(text) && s.internalEnable {
		if fallback, fallbackSource, err := s.tryInternal(format, "it", analysis, cards); err == nil && strings.TrimSpace(fallback) != "" {
			return fallback, fallbackSource
		}
	}
	return text, source
}

func seemsEnglishSuggestion(text string) bool {
	l := strings.ToLower(strings.TrimSpace(text))
	if l == "" {
		return false
	}
	if strings.Contains(l, "cut:") || strings.Contains(l, "add:") || strings.Contains(l, "why:") {
		return true
	}
	enSignals := []string{"you should", "your deck", "replace", "because", "interaction", "mana base"}
	hits := 0
	for _, s := range enSignals {
		if strings.Contains(l, s) {
			hits++
		}
	}
	return hits >= 2
}

func appendDeterministicCoachingFooter(text, locale string, analysis *domain.AnalysisResult, source string) string {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" || analysis == nil {
		return text
	}
	if strings.TrimSpace(source) == "internal_rules" {
		return text
	}
	if strings.Contains(trimmed, "Rule-check:") || strings.Contains(trimmed, "Controllo regole:") {
		return text
	}

	footer := buildDeterministicCoachingFooter(locale, analysis)
	if footer == "" {
		return text
	}
	return trimmed + "\n\n" + footer
}

func buildDeterministicCoachingFooter(locale string, analysis *domain.AnalysisResult) string {
	if analysis == nil {
		return ""
	}

	it := strings.HasPrefix(strings.ToLower(strings.TrimSpace(locale)), "it")
	mana := analysis.Mana
	interaction := analysis.Interaction
	colors := 0
	for _, req := range mana.SourceRequirements {
		if req.Required > 0 {
			colors++
		}
	}

	if it {
		return fmt.Sprintf(
			"Controllo regole:\n- Terre %d/%d, fonti totali %d/%d.\n- Consistenza: screw %.1f%%, flood %.1f%%, sweet spot %.1f%%.\n- Piano %s con %d colori: verifica che i primi turni non siano rallentati da troppe terre tappate.\n- Iterazione consigliata: 5-10 partite, poi aggiusta quantita (4x core, 2-3x situazionali, 1x silver bullet).",
			mana.LandCount, mana.IdealLandCount, mana.CurrentTotalSources, mana.TargetTotalSources,
			mana.ManaScrewChance, mana.ManaFloodChance, mana.SweetSpotChance,
			interaction.Archetype, colors,
		)
	}

	return fmt.Sprintf(
		"Rule-check:\n- Lands %d/%d, total sources %d/%d.\n- Consistency: screw %.1f%%, flood %.1f%%, sweet spot %.1f%%.\n- %s plan with %d colors: verify early turns are not slowed by too many tapped lands.\n- Suggested loop: play 5-10 matches, then tune quantities (4x core, 2-3x situational, 1x silver bullet).",
		mana.LandCount, mana.IdealLandCount, mana.CurrentTotalSources, mana.TargetTotalSources,
		mana.ManaScrewChance, mana.ManaFloodChance, mana.SweetSpotChance,
		interaction.Archetype, colors,
	)
}
