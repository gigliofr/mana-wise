package usecase

import (
	"fmt"
	"sort"
	"strings"

	"github.com/manawise/api/domain"
)

// BuildInternalSuggestions returns concise rule-based suggestions from deterministic analysis.
func BuildInternalSuggestions(a *domain.AnalysisResult) string {
	if a == nil {
		return ""
	}
	return BuildInternalSuggestionsLocalized(a, a.Format, "en", nil)
}

// BuildInternalSuggestionsLocalized returns locale-aware, card-specific suggestions.
// cards is the resolved deck card slice (may be nil; used to name specific cards in suggestions).
func BuildInternalSuggestionsLocalized(a *domain.AnalysisResult, format, locale string, cards []*domain.Card) string {
	if a == nil {
		return ""
	}

	format = strings.TrimSpace(strings.ToLower(format))
	if format == "" {
		format = strings.TrimSpace(strings.ToLower(a.Format))
	}
	if format == "" {
		format = "standard"
	}

	it := strings.HasPrefix(strings.ToLower(strings.TrimSpace(locale)), "it")

	type scoredSuggestion struct {
		score int
		text  string
	}
	items := make([]scoredSuggestion, 0, 8)

	// ── Land count ────────────────────────────────────────────────────────────
	landDelta := a.Mana.LandCount - a.Mana.IdealLandCount
	if landDelta <= -2 {
		slots := -landDelta
		if slots > 4 {
			slots = 4
		}
		if slots < 2 {
			slots = 2
		}
		cutNames := topHighCMCNonlands(cards, slots)
		cutDesc := fmt.Sprintf("%d expensive nonland slot(s) (prefer MV 5+)", slots)
		if len(cutNames) > 0 {
			cutDesc = strings.Join(cutNames, " / ")
		}
		var text string
		if it {
			text = fmt.Sprintf("TOGLI: %s. METTI: %d terra/e o magie di selezione a basso costo legali in %s. PERCHE': la base mana e' corta di %d (%d/%d).", cutDesc, slots, format, -landDelta, a.Mana.LandCount, a.Mana.IdealLandCount)
		} else {
			text = fmt.Sprintf("CUT: %s. ADD: %d land(s) or low-cost selection spell(s) legal in %s. WHY: mana base is short by %d (%d/%d).", cutDesc, slots, format, -landDelta, a.Mana.LandCount, a.Mana.IdealLandCount)
		}
		items = append(items, scoredSuggestion{score: 95, text: text})
	} else if landDelta >= 3 {
		slots := landDelta
		if slots > 4 {
			slots = 4
		}
		landNames := topLandsToCut(cards, slots)
		cutDesc := fmt.Sprintf("%d utility land(s)", slots)
		if len(landNames) > 0 {
			cutDesc = strings.Join(landNames, " / ")
		}
		var text string
		if it {
			text = fmt.Sprintf("TOGLI: %s. METTI: %d magia/e di interazione o vantaggio carte legali in %s. PERCHE': il numero di terre supera il benchmark di %d (%d/%d).", cutDesc, slots, format, landDelta, a.Mana.LandCount, a.Mana.IdealLandCount)
		} else {
			text = fmt.Sprintf("CUT: %s. ADD: %d interaction or card-advantage spell(s) legal in %s. WHY: land count is above benchmark by %d (%d/%d).", cutDesc, slots, format, landDelta, a.Mana.LandCount, a.Mana.IdealLandCount)
		}
		items = append(items, scoredSuggestion{score: 82, text: text})
	}

	// ── Average CMC ───────────────────────────────────────────────────────────
	if a.Mana.AverageCMC >= 3.6 {
		highNames := topHighCMCNonlands(cards, 3)
		cutDesc := "2-4 card(s) at MV 5+"
		if len(highNames) > 0 {
			cutDesc = strings.Join(highNames, " / ")
		}
		var text string
		if it {
			text = fmt.Sprintf("TOGLI: %s. METTI: 2-4 giocate proattive a MV 2-3 legali in %s. PERCHE': il mana value medio e' %.2f e i primi turni rischiano di essere lenti.", cutDesc, format, a.Mana.AverageCMC)
		} else {
			text = fmt.Sprintf("CUT: %s. ADD: 2-4 proactive play(s) at MV 2-3 legal in %s. WHY: average mana value is %.2f and early turns are likely slow.", cutDesc, format, a.Mana.AverageCMC)
		}
		items = append(items, scoredSuggestion{score: 78, text: text})
	}

	// ── Interaction score ─────────────────────────────────────────────────────
	if a.Interaction.TotalScore < 40 {
		cutNames := filterNonInteractiveCards(cards, 3)
		cutDesc := "3 low-impact win-more slot(s)"
		if it {
			cutDesc = "3 slot win-more a basso impatto"
		}
		if len(cutNames) > 0 {
			cutDesc = strings.Join(cutNames, " / ")
		}
		var text string
		if it {
			text = fmt.Sprintf("TOGLI: %s. METTI: 3 carte di interazione flessibile legali in %s, dando priorita' alle categorie piu' carenti. PERCHE': il punteggio interazioni e' %.1f/100.", cutDesc, format, a.Interaction.TotalScore)
		} else {
			text = fmt.Sprintf("CUT: %s. ADD: 3 flexible interaction card(s) legal in %s, prioritizing your weakest categories. WHY: interaction score is %.1f/100.", cutDesc, format, a.Interaction.TotalScore)
		}
		items = append(items, scoredSuggestion{score: 90, text: text})
	} else if a.Interaction.TotalScore < 70 {
		cutNames := filterNonInteractiveCards(cards, 2)
		cutDesc := "2 narrow reactive slot(s) with low matchup coverage"
		if it {
			cutDesc = "2 slot reattivi troppo situazionali"
		}
		if len(cutNames) > 0 {
			cutDesc = strings.Join(cutNames, " / ")
		}
		var text string
		if it {
			text = fmt.Sprintf("TOGLI: %s. METTI: 2 carte di interazione piu' trasversali legali in %s. PERCHE': l'interazione e' %.1f/100 e non ancora stabile.", cutDesc, format, a.Interaction.TotalScore)
		} else {
			text = fmt.Sprintf("CUT: %s. ADD: 2 broader interaction card(s) legal in %s. WHY: interaction is %.1f/100 and still unstable.", cutDesc, format, a.Interaction.TotalScore)
		}
		items = append(items, scoredSuggestion{score: 68, text: text})
	}

	// ── Interaction breakdown deficits ────────────────────────────────────────
	breakdowns := make([]domain.InteractionBreakdown, 0, len(a.Interaction.Breakdowns))
	for _, b := range a.Interaction.Breakdowns {
		if b.Delta < 0 {
			breakdowns = append(breakdowns, b)
		}
	}
	sort.Slice(breakdowns, func(i, j int) bool {
		return breakdowns[i].Delta < breakdowns[j].Delta
	})

	for i := 0; i < len(breakdowns) && i < 2; i++ {
		b := breakdowns[i]
		cat := strings.Title(string(b.Category))
		if it {
			cat = strings.ToUpper(string(b.Category[:1])) + string(b.Category[1:])
		}
		cutNames := filterNonInteractiveCards(cards, -b.Delta)
		cutDesc := fmt.Sprintf("%d win-more or redundant slot(s)", -b.Delta)
		if it {
			cutDesc = fmt.Sprintf("%d slot win-more o ridondanti", -b.Delta)
		}
		if len(cutNames) > 0 {
			cutDesc = strings.Join(cutNames, " / ")
		}
		hint := cardHintForCategory(format, string(b.Category), locale)
		var text string
		if it {
			addDesc := fmt.Sprintf("%d carta/e di tipo %s legali in %s", -b.Delta, strings.ToLower(string(b.Category)), format)
			if hint != "" {
				addDesc = hint
			}
			text = fmt.Sprintf("TOGLI: %s. METTI: %s. PERCHE': %s e' sotto target (%d vs %d).", cutDesc, addDesc, cat, b.Count, b.Ideal)
		} else {
			addDesc := fmt.Sprintf("%d %s card(s) legal in %s", -b.Delta, strings.ToLower(string(b.Category)), format)
			if hint != "" {
				addDesc = hint
			}
			text = fmt.Sprintf("CUT: %s. ADD: %s. WHY: %s is under target (%d vs %d).", cutDesc, addDesc, cat, b.Count, b.Ideal)
		}
		items = append(items, scoredSuggestion{score: 74 - i*4, text: text})
	}

	// ── Mana curve suggestion ─────────────────────────────────────────────────
	if len(a.Mana.Suggestions) > 0 {
		top := a.Mana.Suggestions[0]
		cutFromBand := topHighCMCNonlands(cards, 2)
		cutDesc := "1-2 underperforming slot(s) in the same mana band"
		if it {
			cutDesc = "1-2 slot sottoperformanti nella stessa fascia di mana"
		}
		if len(cutFromBand) > 0 {
			cutDesc = strings.Join(cutFromBand, " / ")
		}
		var text string
		if it {
			text = fmt.Sprintf("TOGLI: %s. METTI: 1-2 alternative legali in %s seguendo questa priorita': %s", cutDesc, format, top.Reason)
		} else {
			text = fmt.Sprintf("CUT: %s. ADD: 1-2 legal-in-%s alternatives following this priority: %s", cutDesc, format, top.Reason)
		}
		items = append(items, scoredSuggestion{score: urgencyScore(top.Urgency), text: text})
	}

	// ── Fallback ──────────────────────────────────────────────────────────────
	if len(items) == 0 {
		if it {
			return fmt.Sprintf("Base del mazzo bilanciata. Piano alternativo: TOGLI 2 slot marginali del main deck in base al meta locale, METTI 2 risposte flessibili legali in %s e verifica 5-10 partite prima di ulteriori cambi.", format)
		}
		return fmt.Sprintf("Deck fundamentals look balanced. Alternative plan: CUT 2 marginal main-deck slots, ADD 2 flexible interaction cards legal in %s, then re-check after 5-10 matches.", format)
	}

	sort.Slice(items, func(i, j int) bool {
		return items[i].score > items[j].score
	})

	max := 3
	if len(items) < max {
		max = len(items)
	}
	lines := make([]string, 0, max)
	for i := 0; i < max; i++ {
		lines = append(lines, fmt.Sprintf("%d) %s", i+1, items[i].text))
	}
	return strings.Join(lines, "\n")
}

func urgencyScore(urgency string) int {
	switch strings.ToLower(strings.TrimSpace(urgency)) {
	case "critical":
		return 88
	case "moderate":
		return 72
	default:
		return 58
	}
}

// topHighCMCNonlands returns up to n non-land card names sorted by CMC descending.
func topHighCMCNonlands(cards []*domain.Card, n int) []string {
	var nonlands []*domain.Card
	for _, c := range cards {
		if c != nil && !strings.Contains(strings.ToLower(c.TypeLine), "land") {
			nonlands = append(nonlands, c)
		}
	}
	sort.Slice(nonlands, func(i, j int) bool {
		return nonlands[i].CMC > nonlands[j].CMC
	})
	names := make([]string, 0, n)
	seen := make(map[string]bool)
	for _, c := range nonlands {
		if len(names) >= n {
			break
		}
		if !seen[c.Name] {
			names = append(names, c.Name)
			seen[c.Name] = true
		}
	}
	return names
}

// topLandsToCut returns up to n land names, prioritizing non-basic lands (less popular first).
func topLandsToCut(cards []*domain.Card, n int) []string {
	var nonbasic, basic []*domain.Card
	for _, c := range cards {
		if c == nil {
			continue
		}
		tl := strings.ToLower(c.TypeLine)
		if strings.Contains(tl, "land") {
			if strings.Contains(tl, "basic") {
				basic = append(basic, c)
			} else {
				nonbasic = append(nonbasic, c)
			}
		}
	}
	sort.Slice(nonbasic, func(i, j int) bool {
		if nonbasic[i].EdhrecRank == nonbasic[j].EdhrecRank {
			return nonbasic[i].Name < nonbasic[j].Name
		}
		return nonbasic[i].EdhrecRank > nonbasic[j].EdhrecRank
	})
	sort.Slice(basic, func(i, j int) bool { return basic[i].Name < basic[j].Name })

	var all []*domain.Card
	all = append(all, nonbasic...)
	all = append(all, basic...)

	names := make([]string, 0, n)
	seen := make(map[string]bool)
	for _, c := range all {
		if len(names) >= n {
			break
		}
		if !seen[c.Name] {
			names = append(names, c.Name)
			seen[c.Name] = true
		}
	}
	return names
}

// filterNonInteractiveCards returns up to n non-land card names that appear least interactive
// (no removal, counter, draw, ramp, or protection keywords in oracle text).
func filterNonInteractiveCards(cards []*domain.Card, n int) []string {
	interactiveKeywords := []string{
		"destroy", "exile", "counter target", "draw a card", "draw two", "draw three",
		"search your library", "add {", "create a ", "return target", "discard",
		"protection from", "hexproof", "indestructible",
	}
	var nonInteractive []*domain.Card
	for _, c := range cards {
		if c == nil || strings.Contains(strings.ToLower(c.TypeLine), "land") {
			continue
		}
		ot := strings.ToLower(c.OracleText)
		interactive := false
		for _, kw := range interactiveKeywords {
			if strings.Contains(ot, kw) {
				interactive = true
				break
			}
		}
		if !interactive {
			nonInteractive = append(nonInteractive, c)
		}
	}
	sort.Slice(nonInteractive, func(i, j int) bool {
		return nonInteractive[i].CMC > nonInteractive[j].CMC
	})
	names := make([]string, 0, n)
	seen := make(map[string]bool)
	for _, c := range nonInteractive {
		if len(names) >= n {
			break
		}
		if !seen[c.Name] {
			names = append(names, c.Name)
			seen[c.Name] = true
		}
	}
	return names
}

// cardHintForCategory returns a short suggestion string with well-known card names
// for the given format, interaction category, and locale.
func cardHintForCategory(format, category, locale string) string {
	type hint struct{ it, en string }
	hints := map[string]hint{
		"commander:draw":       {"es. Rhystic Study / Phyrexian Arena / Necropotence", "e.g. Rhystic Study / Phyrexian Arena / Necropotence"},
		"commander:removal":    {"es. Swords to Plowshares / Path to Exile / Beast Within", "e.g. Swords to Plowshares / Path to Exile / Beast Within"},
		"commander:counter":    {"es. Counterspell / Swan Song / Fierce Guardianship", "e.g. Counterspell / Swan Song / Fierce Guardianship"},
		"commander:ramp":       {"es. Sol Ring / Cultivate / Arcane Signet", "e.g. Sol Ring / Cultivate / Arcane Signet"},
		"commander:protection": {"es. Swiftfoot Boots / Lightning Greaves / Heroic Intervention", "e.g. Swiftfoot Boots / Lightning Greaves / Heroic Intervention"},
		"standard:draw":        {"es. Preordain / Memory Deluge / Bitter Reunion", "e.g. Preordain / Memory Deluge / Bitter Reunion"},
		"standard:removal":     {"es. Cut Down / Sheoldred's Edict / Abrade", "e.g. Cut Down / Sheoldred's Edict / Abrade"},
		"standard:counter":     {"es. Make Disappear / Negate / Disdainful Stroke", "e.g. Make Disappear / Negate / Disdainful Stroke"},
		"standard:ramp":        {"es. Elvish Mystic / Topiary Stomper / Explore", "e.g. Elvish Mystic / Topiary Stomper / Explore"},
		"modern:draw":          {"es. Expressive Iteration / Preordain / Brainstorm", "e.g. Expressive Iteration / Preordain / Brainstorm"},
		"modern:removal":       {"es. Lightning Bolt / Fatal Push / Path to Exile", "e.g. Lightning Bolt / Fatal Push / Path to Exile"},
		"modern:counter":       {"es. Counterspell / Force of Negation / Remand", "e.g. Counterspell / Force of Negation / Remand"},
		"modern:ramp":          {"es. Utopia Sprawl / Birds of Paradise / Arbor Elf", "e.g. Utopia Sprawl / Birds of Paradise / Arbor Elf"},
		"pioneer:draw":         {"es. Opt / Consider / Curious Obsession", "e.g. Opt / Consider / Curious Obsession"},
		"pioneer:removal":      {"es. Fatal Push / Eliminate / Chained to the Rocks", "e.g. Fatal Push / Eliminate / Chained to the Rocks"},
		"pioneer:counter":      {"es. Negate / Make Disappear / Disdainful Stroke", "e.g. Negate / Make Disappear / Disdainful Stroke"},
	}

	key := fmt.Sprintf("%s:%s", strings.ToLower(format), strings.ToLower(category))
	if h, ok := hints[key]; ok {
		if strings.HasPrefix(strings.ToLower(strings.TrimSpace(locale)), "it") {
			return h.it
		}
		return h.en
	}
	return ""
}
