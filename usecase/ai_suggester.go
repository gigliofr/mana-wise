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

// Suggest returns AI suggestions, source label, and a non-fatal error if all strategies fail.
// cards is the resolved card slice (may be nil; used for card-name-specific suggestions).
func (s *AISuggester) Suggest(ctx context.Context, decklist, format, locale string, analysis *domain.AnalysisResult, cards []*domain.Card) (string, string, error) {
	hash := llm.HashDecklist(decklist, format)

	switch s.mode {
	case AIModeInternalOnly:
		return s.tryInternal(format, locale, analysis, cards)
	case AIModeHybridPreferInternal:
		if text, source, err := s.tryInternal(format, locale, analysis, cards); err == nil {
			return text, source, nil
		}
		return s.tryExternalChain(ctx, hash, decklist, format, locale, analysis, cards)
	case AIModeExternalOnly:
		return s.tryExternalChain(ctx, hash, decklist, format, locale, analysis, cards)
	default: // hybrid_prefer_external
		text, source, err := s.tryExternalChain(ctx, hash, decklist, format, locale, analysis, cards)
		if err == nil {
			return text, source, nil
		}
		if !s.internalEnable {
			return "", "", err
		}
		internalText, internalSource, internalErr := s.tryInternal(format, locale, analysis, cards)
		if internalErr != nil {
			return "", "", err
		}
		return internalText, internalSource, nil
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
