package usecase

import (
	"context"
	"fmt"
	"strings"

	"github.com/manawise/api/domain"
	"github.com/manawise/api/infrastructure/llm"
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
}

// NewAISuggester creates an AI suggester with runtime routing rules.
func NewAISuggester(mode string, primary, secondary *llm.Connector, internalEnable bool) *AISuggester {
	mode = strings.ToLower(strings.TrimSpace(mode))
	switch mode {
	case AIModeExternalOnly, AIModeInternalOnly, AIModeHybridPreferExternal, AIModeHybridPreferInternal:
	default:
		mode = AIModeHybridPreferExternal
	}
	return &AISuggester{
		mode:           mode,
		primary:        primary,
		secondary:      secondary,
		internalEnable: internalEnable,
	}
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
		}
		return
	case AIModeExternalOnly:
		text, source, err = s.tryExternalChain(ctx, hash, decklist, format, locale, analysis, cards)
		if err == nil {
			text, source = s.enforceLocale(format, locale, analysis, cards, text, source)
		}
		return
	default: // hybrid_prefer_external
		var extErr error
		text, source, extErr = s.tryExternalChain(ctx, hash, decklist, format, locale, analysis, cards)
		if extErr == nil {
			text, source = s.enforceLocale(format, locale, analysis, cards, text, source)
			return
		}
		if !s.internalEnable {
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
