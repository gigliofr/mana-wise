package usecase

import (
	"sort"
	"strings"

	"github.com/gigliofr/mana-wise/domain"
)

// CommanderBracketUseCase evaluates Commander decks against configurable bracket rules.
type CommanderBracketUseCase struct {
	cfg *domain.CommanderBracketConfig
}

// NewCommanderBracketUseCase builds a bracket evaluator from the supplied configuration.

func NewCommanderBracketUseCase(cfg *domain.CommanderBracketConfig) *CommanderBracketUseCase {
	if cfg == nil {
		defaultCfg := domain.DefaultCommanderBracketConfig()
		cfg = &defaultCfg
	}
	if !cfg.Enabled {
		defaultCfg := domain.DefaultCommanderBracketConfig()
		cfg = &defaultCfg
	}
	return &CommanderBracketUseCase{cfg: cfg}
}

// Evaluate derives the bracket classification for a Commander deck.
func (uc *CommanderBracketUseCase) Evaluate(cards []*domain.Card, quantities map[string]int) *domain.CommanderBracketAssessment {
	if uc == nil || uc.cfg == nil || !uc.cfg.Enabled || len(cards) == 0 {
		return nil
	}

	signals := collectCommanderSignals(cards, quantities, uc.cfg)
	bracket, reason, notes := classifyCommanderBracket(*uc.cfg, signals)
	return &domain.CommanderBracketAssessment{
		Bracket: bracket,
		Label:   bracketLabel(bracket),
		Reason:  reason,
		Signals: signals,
		Notes:   notes,
	}
}

// Config returns the live commander bracket configuration.
func (uc *CommanderBracketUseCase) Config() *domain.CommanderBracketConfig {
	if uc == nil {
		return nil
	}
	return uc.cfg
}

func collectCommanderSignals(cards []*domain.Card, quantities map[string]int, cfg *domain.CommanderBracketConfig) domain.CommanderBracketSignals {
	if cfg == nil {
		return domain.CommanderBracketSignals{}
	}
	decisiveSet := make(map[string]struct{}, len(cfg.DecisiveCards))
	for _, name := range cfg.DecisiveCards {
		decisiveSet[domain.NormalizeCommanderCardName(name)] = struct{}{}
	}

	tutorMatchers := normalizeTerms(append([]string{
		"search your library for a card",
		"search your library for a creature card",
		"search your library for an artifact card",
		"search your library for an enchantment card",
		"search your library for an instant card",
		"search your library for a sorcery card",
		"search your library for a planeswalker card",
		"search your library for a land card",
		"put it into your hand",
		"tutor",
	}, cfg.TutorKeywords...))
	extraTurnMatchers := normalizeTerms(cfg.ExtraTurnKeywords)
	massLandMatchers := normalizeTerms(cfg.MassLandDenialKeywords)
	comboMatchers := normalizeTerms(cfg.ComboKeywords)
	fastManaMatchers := normalizeTerms(cfg.FastManaKeywords)

	appendUnique := func(dst []string, name string) []string {
		name = strings.TrimSpace(name)
		if name == "" {
			return dst
		}
		for _, existing := range dst {
			if strings.EqualFold(existing, name) {
				return dst
			}
		}
		return append(dst, name)
	}

	var signals domain.CommanderBracketSignals
	var totalCMC float64
	for _, card := range cards {
		if card == nil {
			continue
		}
		qty := quantities[card.ID]
		if qty <= 0 {
			qty = 1
		}
		if card.IsLand() {
			signals.LandCount += qty
			continue
		}

		signals.NonLandCount += qty
		totalCMC += card.CMC * float64(qty)

		name := strings.TrimSpace(card.Name)
		nameKey := domain.NormalizeCommanderCardName(name)
		text := strings.ToLower(strings.TrimSpace(card.OracleText + " " + card.TypeLine + " " + strings.Join(card.Keywords, " ")))

		if _, ok := decisiveSet[nameKey]; ok {
			signals.DecisiveCards = appendUnique(signals.DecisiveCards, name)
		}
		if matchesAny(text, tutorMatchers) {
			signals.TutorCards = appendUnique(signals.TutorCards, name)
		}
		if matchesAny(text, extraTurnMatchers) {
			signals.ExtraTurnCards = appendUnique(signals.ExtraTurnCards, name)
		}
		if matchesAny(text, massLandMatchers) {
			signals.MassLandDenialCards = appendUnique(signals.MassLandDenialCards, name)
		}
		if matchesAny(text, comboMatchers) {
			signals.ComboCards = appendUnique(signals.ComboCards, name)
		}
		if matchesAny(text, fastManaMatchers) {
			signals.FastManaCards = appendUnique(signals.FastManaCards, name)
		}
	}

	if signals.NonLandCount > 0 {
		signals.AverageCMC = totalCMC / float64(signals.NonLandCount)
	}
	sort.Strings(signals.DecisiveCards)
	sort.Strings(signals.TutorCards)
	sort.Strings(signals.ExtraTurnCards)
	sort.Strings(signals.MassLandDenialCards)
	sort.Strings(signals.ComboCards)
	sort.Strings(signals.FastManaCards)
	return signals
}

func classifyCommanderBracket(cfg domain.CommanderBracketConfig, signals domain.CommanderBracketSignals) (int, string, []string) {
	decisiveCount := len(signals.DecisiveCards)
	tutorCount := len(signals.TutorCards)
	extraTurns := len(signals.ExtraTurnCards)
	massDenial := len(signals.MassLandDenialCards)
	comboCount := len(signals.ComboCards)
	fastManaCount := len(signals.FastManaCards)
	totalSignals := tutorCount + extraTurns + massDenial + comboCount + fastManaCount

	if massDenial > 0 {
		if fastManaCount >= cfg.CedhFastManaThreshold || comboCount >= cfg.CedhComboThreshold {
			return 5, "Competitive", []string{"mass land denial plus competitive pressure signals"}
		}
		return 4, "Optimized", []string{"mass land denial pushes the deck beyond the lower brackets"}
	}

	if extraTurns >= 2 || comboCount >= cfg.CedhComboThreshold || fastManaCount >= cfg.CedhFastManaThreshold || tutorCount >= cfg.CedhTutorThreshold || decisiveCount >= cfg.CedhDecisiveThreshold {
		return 5, "cEDH", []string{"competitive-density signals exceeded the cEDH thresholds"}
	}

	if decisiveCount > cfg.Bracket3MaxDecisive || totalSignals > cfg.Bracket4MaxSignals {
		return 4, "Optimized", []string{"the deck carries too many advanced game-plan signals for bracket 3"}
	}

	if decisiveCount == 0 && totalSignals == 0 {
		if signals.AverageCMC >= 3.2 || signals.LandCount >= 37 {
			return 1, "Casual", []string{"no advanced commander signals and a slower mana profile"}
		}
		return 2, "Upgraded", []string{"no advanced commander signals, but the shell is tighter than a raw precon"}
	}

	if decisiveCount == 0 && totalSignals <= cfg.Bracket2MaxSignals {
		return 2, "Upgraded", []string{"limited commander-specific pressure signals"}
	}

	if decisiveCount <= cfg.Bracket3MaxDecisive && totalSignals <= cfg.Bracket3MaxSignals {
		return 3, "Tuned", []string{"up to three decisive cards and limited support for higher brackets"}
	}

	return 4, "Optimized", []string{"the deck is tuned beyond bracket 3 but does not trip the competitive thresholds"}
}

func bracketLabel(bracket int) string {
	switch bracket {
	case 1:
		return "Casual"
	case 2:
		return "Upgraded"
	case 3:
		return "Tuned"
	case 4:
		return "Optimized"
	case 5:
		return "cEDH"
	default:
		return "Unknown"
	}
}

func normalizeTerms(terms []string) []string {
	out := make([]string, 0, len(terms))
	seen := map[string]bool{}
	for _, term := range terms {
		term = strings.ToLower(strings.TrimSpace(term))
		if term == "" || seen[term] {
			continue
		}
		seen[term] = true
		out = append(out, term)
	}
	return out
}

func matchesAny(text string, terms []string) bool {
	for _, term := range terms {
		if strings.Contains(text, term) {
			return true
		}
	}
	return false
}
