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

	if a.Interaction.TotalScore < 40 {
		var text string
		if it {
			text = fmt.Sprintf("TOGLI: 3 slot win-more a basso impatto. METTI: 3 carte di interazione flessibile legali in %s, dando priorita' alle categorie piu' carenti. PERCHE': il punteggio interazioni e' %.1f/100.", format, a.Interaction.TotalScore)
		} else {
			text = fmt.Sprintf("CUT: 3 low-impact win-more slot(s). ADD: 3 flexible interaction card(s) legal in %s, prioritizing your weakest categories. WHY: interaction score is %.1f/100.", format, a.Interaction.TotalScore)
		}
		items = append(items, scoredSuggestion{score: 90, text: text})
	} else if a.Interaction.TotalScore < 70 {
		var text string
		if it {
			text = fmt.Sprintf("TOGLI: 2 slot reattivi troppo situazionali. METTI: 2 carte di interazione piu' trasversali legali in %s. PERCHE': l'interazione e' %.1f/100 e non ancora stabile.", format, a.Interaction.TotalScore)
		} else {
			text = fmt.Sprintf("CUT: 2 narrow reactive slot(s) with low matchup coverage. ADD: 2 broader interaction card(s) legal in %s. WHY: interaction is %.1f/100 and still unstable.", format, a.Interaction.TotalScore)
		}
		items = append(items, scoredSuggestion{score: 68, text: text})
	}

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
		var text string
		if it {
			text = fmt.Sprintf("TOGLI: %d slot da categorie in eccesso. METTI: %d carta/e di tipo %s legali in %s. PERCHE': %s e' sotto target (%d vs %d).", -b.Delta, -b.Delta, strings.ToLower(string(b.Category)), format, cat, b.Count, b.Ideal)
		} else {
			text = fmt.Sprintf("CUT: %d slot(s) from overrepresented categories. ADD: %d %s card(s) legal in %s. WHY: %s is under target (%d vs %d).", -b.Delta, -b.Delta, strings.ToLower(string(b.Category)), format, cat, b.Count, b.Ideal)
		}
		items = append(items, scoredSuggestion{score: 74 - i*4, text: text})
	}

	if len(a.Mana.Suggestions) > 0 {
		top := a.Mana.Suggestions[0]
		var text string
		if it {
			text = fmt.Sprintf("TOGLI: 1-2 slot sottoperformanti nella stessa fascia di mana. METTI: 1-2 alternative legali in %s seguendo questa priorita': %s", format, top.Reason)
		} else {
			text = fmt.Sprintf("CUT: 1-2 underperforming slot(s) in the same mana band. ADD: 1-2 legal-in-%s alternatives following this priority: %s", format, top.Reason)
		}
		items = append(items, scoredSuggestion{score: urgencyScore(top.Urgency), text: text})
	}

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
	// Sort non-basic by EDHREC rank descending (higher number = less popular = cut first)
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
