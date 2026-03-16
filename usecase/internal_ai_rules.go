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

	type scoredSuggestion struct {
		score int
		text  string
	}
	items := make([]scoredSuggestion, 0, 8)

	landDelta := a.Mana.LandCount - a.Mana.IdealLandCount
	if landDelta <= -2 {
		items = append(items, scoredSuggestion{
			score: 95,
			text:  fmt.Sprintf("Your mana base is short by %d lands versus the format benchmark (%d/%d). Add lands or low-cost card selection to improve opening-hand consistency.", -landDelta, a.Mana.LandCount, a.Mana.IdealLandCount),
		})
	} else if landDelta >= 3 {
		items = append(items, scoredSuggestion{
			score: 82,
			text:  fmt.Sprintf("You are above the land benchmark by %d (%d/%d). Consider cutting a few lands for interaction or card advantage slots.", landDelta, a.Mana.LandCount, a.Mana.IdealLandCount),
		})
	}

	if a.Mana.AverageCMC >= 3.6 {
		items = append(items, scoredSuggestion{
			score: 78,
			text:  fmt.Sprintf("Average CMC is %.2f, which may slow your early turns. Shift 2-4 slots toward cheaper plays to smooth turns 2-4.", a.Mana.AverageCMC),
		})
	}

	if a.Interaction.TotalScore < 40 {
		items = append(items, scoredSuggestion{
			score: 90,
			text:  fmt.Sprintf("Interaction density is low (%.1f/100). Prioritize flexible answers in your weakest categories before adding win-more cards.", a.Interaction.TotalScore),
		})
	} else if a.Interaction.TotalScore < 70 {
		items = append(items, scoredSuggestion{
			score: 68,
			text:  fmt.Sprintf("Interaction is serviceable (%.1f/100) but not fully stable. Upgrading 2-3 reactive slots can improve matchup spread.", a.Interaction.TotalScore),
		})
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
		items = append(items, scoredSuggestion{
			score: 74 - i*4,
			text:  fmt.Sprintf("%s is under target (%d vs %d ideal for this profile). Add %d more slot(s) to reduce pressure points.", strings.Title(string(b.Category)), b.Count, b.Ideal, -b.Delta),
		})
	}

	if len(a.Mana.Suggestions) > 0 {
		top := a.Mana.Suggestions[0]
		items = append(items, scoredSuggestion{
			score: urgencyScore(top.Urgency),
			text:  top.Reason,
		})
	}

	if len(items) == 0 {
		return "Deck fundamentals look balanced. Tune sideboard plans for your expected meta and keep monitoring land/mana consistency over multiple matches."
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
