package usecase

import (
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/gigliofr/mana-wise/domain"
)

// CommanderBracketUseCase evaluates Commander decks against the bracket rules.
// The configuration can be read and replaced at runtime (admin API, Scryfall
// sync); all access goes through an RWMutex.
type CommanderBracketUseCase struct {
	mu  sync.RWMutex
	cfg domain.CommanderBracketConfig
}

// NewCommanderBracketUseCase builds a bracket evaluator from the supplied
// configuration (copied). A nil or disabled configuration falls back to the
// defaults.
func NewCommanderBracketUseCase(cfg *domain.CommanderBracketConfig) *CommanderBracketUseCase {
	if cfg == nil || !cfg.Enabled {
		def := domain.DefaultCommanderBracketConfig()
		cfg = &def
	}
	return &CommanderBracketUseCase{cfg: *cfg}
}

// Config returns a copy of the live configuration.
func (uc *CommanderBracketUseCase) Config() domain.CommanderBracketConfig {
	uc.mu.RLock()
	defer uc.mu.RUnlock()
	return uc.cfg
}

// SetConfig replaces the live configuration.
func (uc *CommanderBracketUseCase) SetConfig(cfg domain.CommanderBracketConfig) {
	uc.mu.Lock()
	defer uc.mu.Unlock()
	uc.cfg = cfg
}

// SetGameChangers replaces only the Game Changers list (used by the Scryfall sync).
func (uc *CommanderBracketUseCase) SetGameChangers(names []string) {
	uc.mu.Lock()
	defer uc.mu.Unlock()
	uc.cfg.GameChangers = append([]string(nil), names...)
}

// Evaluate derives the minimum bracket for a Commander deck.
func (uc *CommanderBracketUseCase) Evaluate(cards []*domain.Card, quantities map[string]int) *domain.CommanderBracketAssessment {
	if uc == nil || len(cards) == 0 {
		return nil
	}
	cfg := uc.Config()
	if !cfg.Enabled {
		return nil
	}

	signals := collectCommanderSignals(cards, quantities, &cfg)
	bracket, reason, notes := classifyCommanderBracket(cfg, signals)
	return &domain.CommanderBracketAssessment{
		Bracket: bracket,
		Label:   domain.CommanderBracketLabel(bracket),
		Reason:  reason,
		Signals: signals,
		Notes:   notes,
		Legal:   len(signals.BannedCards) == 0,
	}
}

func nameSet(names []string) map[string]struct{} {
	set := make(map[string]struct{}, len(names))
	for _, n := range names {
		set[domain.NormalizeCommanderCardName(n)] = struct{}{}
	}
	return set
}

func appendUniqueName(dst []string, name string) []string {
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

func collectCommanderSignals(cards []*domain.Card, quantities map[string]int, cfg *domain.CommanderBracketConfig) domain.CommanderBracketSignals {
	if cfg == nil {
		return domain.CommanderBracketSignals{}
	}
	gameChangers := nameSet(cfg.GameChangers)
	extraTurnSet := nameSet(cfg.ExtraTurnCards)
	massLandSet := nameSet(cfg.MassLandDenialCards)

	tutorMatchers := normalizeTerms(cfg.TutorKeywords)
	extraTurnMatchers := normalizeTerms(cfg.ExtraTurnKeywords)
	massLandMatchers := normalizeTerms(cfg.MassLandDenialKeywords)
	fastManaMatchers := normalizeTerms(cfg.FastManaKeywords)

	var signals domain.CommanderBracketSignals
	var totalCMC float64
	byName := make(map[string]*domain.Card, len(cards))

	for _, card := range cards {
		if card == nil {
			continue
		}
		qty := quantities[card.ID]
		if qty <= 0 {
			qty = 1
		}
		name := strings.TrimSpace(card.Name)
		nameKey := domain.NormalizeCommanderCardName(name)
		byName[nameKey] = card

		// Checks that also apply to lands (several Game Changers are lands).
		if _, ok := gameChangers[nameKey]; ok || card.GameChanger {
			signals.GameChangerCards = appendUniqueName(signals.GameChangerCards, name)
		}
		if status, ok := card.Legalities["commander"]; ok && status == "banned" {
			signals.BannedCards = appendUniqueName(signals.BannedCards, name)
		}

		if card.IsLand() {
			signals.LandCount += qty
			continue
		}

		signals.NonLandCount += qty
		totalCMC += card.CMC * float64(qty)

		text := strings.ToLower(strings.TrimSpace(card.OracleText + " " + card.TypeLine + " " + strings.Join(card.Keywords, " ")))

		if matchesAny(text, tutorMatchers) {
			signals.TutorCards = appendUniqueName(signals.TutorCards, name)
		}
		if _, ok := extraTurnSet[nameKey]; ok || matchesAny(text, extraTurnMatchers) {
			signals.ExtraTurnCards = appendUniqueName(signals.ExtraTurnCards, name)
		}
		if _, ok := massLandSet[nameKey]; ok || matchesAny(text, massLandMatchers) {
			signals.MassLandDenialCards = appendUniqueName(signals.MassLandDenialCards, name)
		}
		if matchesAny(text, fastManaMatchers) {
			signals.FastManaCards = appendUniqueName(signals.FastManaCards, name)
		}
	}

	signals.TwoCardCombos = detectTwoCardCombos(byName, cfg.TwoCardCombos, cfg.EarlyComboMaxManaValue)

	if signals.NonLandCount > 0 {
		signals.AverageCMC = totalCMC / float64(signals.NonLandCount)
	}
	sort.Strings(signals.GameChangerCards)
	sort.Strings(signals.TutorCards)
	sort.Strings(signals.ExtraTurnCards)
	sort.Strings(signals.MassLandDenialCards)
	sort.Strings(signals.FastManaCards)
	sort.Strings(signals.BannedCards)
	return signals
}

// detectTwoCardCombos returns the known combos fully present in the deck.
// A combo is "early" when the mana values of its pieces add up to at most
// earlyMaxMV, a proxy for "can be assembled before turn 6".
func detectTwoCardCombos(byName map[string]*domain.Card, combos []domain.TwoCardCombo, earlyMaxMV float64) []domain.DetectedCombo {
	if earlyMaxMV <= 0 {
		earlyMaxMV = 6
	}
	var out []domain.DetectedCombo
	for _, combo := range combos {
		a, okA := byName[domain.NormalizeCommanderCardName(combo.Cards[0])]
		b, okB := byName[domain.NormalizeCommanderCardName(combo.Cards[1])]
		if !okA || !okB {
			continue
		}
		mv := a.CMC + b.CMC
		out = append(out, domain.DetectedCombo{
			Cards:     []string{a.Name, b.Name},
			Result:    combo.Result,
			ManaValue: mv,
			Early:     mv <= earlyMaxMV,
		})
	}
	sort.Slice(out, func(i, j int) bool { return strings.Join(out[i].Cards, "+") < strings.Join(out[j].Cards, "+") })
	return out
}

func classifyCommanderBracket(cfg domain.CommanderBracketConfig, s domain.CommanderBracketSignals) (int, string, []string) {
	gc := len(s.GameChangerCards)
	extraTurns := len(s.ExtraTurnCards)
	mld := len(s.MassLandDenialCards)
	earlyCombos, lateCombos := 0, 0
	for _, c := range s.TwoCardCombos {
		if c.Early {
			earlyCombos++
		} else {
			lateCombos++
		}
	}

	maxGC := cfg.Bracket3MaxGameChangers
	if maxGC <= 0 {
		maxGC = 3
	}
	maxExtraTurns := cfg.Bracket3MaxExtraTurns
	if maxExtraTurns <= 0 {
		maxExtraTurns = 2
	}

	var notes []string
	if len(s.BannedCards) > 0 {
		notes = append(notes, fmt.Sprintf("banned in Commander: %s", strings.Join(s.BannedCards, ", ")))
	}
	if len(s.TutorCards) > 0 {
		notes = append(notes, "tutors are reported for information: since October 2025 they no longer raise the bracket")
	}

	// Bracket 4 triggers.
	var b4 []string
	if gc > maxGC {
		b4 = append(b4, fmt.Sprintf("%d Game Changers (Bracket 3 allows up to %d)", gc, maxGC))
	}
	if mld > 0 {
		b4 = append(b4, "mass land denial")
	}
	if extraTurns > maxExtraTurns {
		b4 = append(b4, fmt.Sprintf("%d extra-turn cards (likely turn chains)", extraTurns))
	}
	if earlyCombos > 0 {
		b4 = append(b4, "two-card combo that can fire in the early game")
	}

	if len(b4) > 0 {
		if gc >= cfg.CedhGameChangerThreshold &&
			earlyCombos >= cfg.CedhEarlyComboThreshold &&
			len(s.FastManaCards) >= cfg.CedhFastManaThreshold &&
			len(s.TutorCards) >= cfg.CedhTutorThreshold {
			notes = append(notes, "cEDH is about tournament intent: confirm it with your table")
			return 5, "competitive density (Game Changers, early combos, fast mana and tutors) matches cEDH", notes
		}
		return 4, strings.Join(b4, "; "), notes
	}

	// Bracket 3 triggers.
	var b3 []string
	if gc > 0 {
		b3 = append(b3, fmt.Sprintf("%d Game Changer(s)", gc))
	}
	if lateCombos > 0 {
		b3 = append(b3, "late-game two-card combo")
	}
	if len(b3) > 0 {
		return 3, strings.Join(b3, "; "), notes
	}

	notes = append(notes, "Bracket 1 (Exhibition) depends on the deck's theme and intent, which a list alone cannot show")
	return 2, "no Game Changers, mass land denial, extra-turn chains or two-card combos", notes
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
