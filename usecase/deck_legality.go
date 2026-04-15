package usecase

import (
	"fmt"
	"sort"
	"strings"

	"github.com/gigliofr/mana-wise/domain"
)

// IllegalCardIssue describes why a specific card entry violates format legality.
type IllegalCardIssue struct {
	CardName string `json:"card_name"`
	Quantity int    `json:"quantity"`
	Reason   string `json:"reason"`
	Legality string `json:"legality"`
	Format   string `json:"format"`
	CardID   string `json:"card_id,omitempty"`
}

// DeckLegalityResult is the legality report for one format.
type DeckLegalityResult struct {
	Format       string             `json:"format"`
	IsLegal      bool               `json:"is_legal"`
	DeckSize     int                `json:"deck_size"`
	Issues       []string           `json:"issues,omitempty"`
	IllegalCards []IllegalCardIssue `json:"illegal_cards,omitempty"`
}

// DetermineDeckLegalityForFormat validates a deck against one target format.
// cards may contain duplicates; quantities must be keyed by card ID.
func DetermineDeckLegalityForFormat(cards []*domain.Card, quantities map[string]int, format string) DeckLegalityResult {
	normalized := domain.NormalizeFormat(format)
	result := DeckLegalityResult{
		Format:   normalized,
		DeckSize: totalDeckSize(cards, quantities),
		IsLegal:  true,
	}

	if !domain.IsValidFormat(normalized) {
		result.IsLegal = false
		result.Issues = append(result.Issues, fmt.Sprintf("unsupported format: %s", format))
		return result
	}

	unique := uniqueCardsByID(cards)

	for _, card := range unique {
		qty := quantities[card.ID]
		if qty <= 0 {
			continue
		}
		status := strings.ToLower(strings.TrimSpace(card.Legalities[normalized]))
		if status == "" {
			status = "unknown"
		}

		if !isCardAllowedInFormat(status, normalized, card) {
			result.IllegalCards = append(result.IllegalCards, IllegalCardIssue{
				CardName: card.Name,
				CardID:   card.ID,
				Quantity: qty,
				Legality: status,
				Format:   normalized,
				Reason:   fmt.Sprintf("card is not legal in %s", normalized),
			})
		}

		copyLimit := maxCopiesAllowed(card, normalized, status)
		if copyLimit > 0 && qty > copyLimit {
			result.IllegalCards = append(result.IllegalCards, IllegalCardIssue{
				CardName: card.Name,
				CardID:   card.ID,
				Quantity: qty,
				Legality: status,
				Format:   normalized,
				Reason:   fmt.Sprintf("too many copies: %d > %d", qty, copyLimit),
			})
		}
	}

	if normalized == "commander" {
		if result.DeckSize != 100 {
			result.Issues = append(result.Issues, fmt.Sprintf("commander deck must contain exactly 100 cards, found %d", result.DeckSize))
		}
	} else {
		if result.DeckSize < 60 {
			result.Issues = append(result.Issues, fmt.Sprintf("%s deck must contain at least 60 cards, found %d", normalized, result.DeckSize))
		}
	}

	if len(result.IllegalCards) > 0 || len(result.Issues) > 0 {
		result.IsLegal = false
	}

	sort.Slice(result.IllegalCards, func(i, j int) bool {
		if result.IllegalCards[i].CardName == result.IllegalCards[j].CardName {
			return result.IllegalCards[i].Reason < result.IllegalCards[j].Reason
		}
		return result.IllegalCards[i].CardName < result.IllegalCards[j].CardName
	})

	return result
}

// DetermineDeckLegalityAllFormats validates a deck for all supported formats.
func DetermineDeckLegalityAllFormats(cards []*domain.Card, quantities map[string]int) map[string]DeckLegalityResult {
	out := make(map[string]DeckLegalityResult, len(domain.SupportedFormats))
	for _, f := range domain.SupportedFormats {
		out[f] = DetermineDeckLegalityForFormat(cards, quantities, f)
	}
	return out
}

func uniqueCardsByID(cards []*domain.Card) []*domain.Card {
	byID := make(map[string]*domain.Card, len(cards))
	for _, c := range cards {
		if c == nil || c.ID == "" {
			continue
		}
		if _, ok := byID[c.ID]; !ok {
			byID[c.ID] = c
		}
	}
	out := make([]*domain.Card, 0, len(byID))
	for _, c := range byID {
		out = append(out, c)
	}
	return out
}

func totalDeckSize(cards []*domain.Card, quantities map[string]int) int {
	total := 0
	for _, c := range uniqueCardsByID(cards) {
		if qty := quantities[c.ID]; qty > 0 {
			total += qty
		}
	}
	return total
}

func isCardAllowedInFormat(status, format string, card *domain.Card) bool {
	switch format {
	case "vintage":
		return status == "legal" || status == "restricted"
	default:
		return status == "legal"
	}
}

func maxCopiesAllowed(card *domain.Card, format, status string) int {
	if format == "vintage" && status == "restricted" {
		return 1
	}
	if format == "commander" {
		if isBasicLand(card) || allowsAnyNumberCopies(card) {
			return 0
		}
		return 1
	}
	if isBasicLand(card) || allowsAnyNumberCopies(card) {
		return 0
	}
	return 4
}

func isBasicLand(card *domain.Card) bool {
	if card == nil {
		return false
	}
	return card.IsBasicLand()
}

func allowsAnyNumberCopies(card *domain.Card) bool {
	text := strings.ToLower(card.OracleText)
	return strings.Contains(text, "a deck can have any number of cards named")
}

// CheckCommanderColorIdentity validates that every card in a Commander deck
// has a color identity that is a subset of the combined commander's color identity.
// commanderCards should only contain the commander(s) (1 or 2 for Partner).
// Returns a slice of violations that can be merged into a DeckLegalityResult.
func CheckCommanderColorIdentity(commanderCards, deckCards []*domain.Card, quantities map[string]int) []IllegalCardIssue {
	if len(commanderCards) == 0 {
		return nil
	}

	// Build combined commander color identity set.
	allowed := make(map[string]bool)
	for _, cmd := range commanderCards {
		for _, ci := range cmd.ColorIdentity {
			allowed[strings.ToUpper(ci)] = true
		}
	}

	unique := uniqueCardsByID(deckCards)
	var violations []IllegalCardIssue
	for _, card := range unique {
		qty := quantities[card.ID]
		if qty == 0 {
			continue
		}
		// Basic lands are always allowed regardless of color identity.
		if isBasicLand(card) {
			continue
		}
		for _, ci := range card.ColorIdentity {
			if !allowed[strings.ToUpper(ci)] {
				violations = append(violations, IllegalCardIssue{
					CardName: card.Name,
					CardID:   card.ID,
					Quantity: qty,
					Legality: "color_identity_violation",
					Format:   "commander",
					Reason:   fmt.Sprintf("color identity {%s} is outside the commander's color identity", strings.ToUpper(ci)),
				})
				break // report once per card
			}
		}
	}
	return violations
}
