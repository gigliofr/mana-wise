package usecase_test

import (
	"strings"
	"testing"

	"github.com/gigliofr/mana-wise/domain"
	"github.com/gigliofr/mana-wise/usecase"
)

func makeLegalityCard(id, name, typeLine, rarity string, legalities map[string]string) *domain.Card {
	return &domain.Card{
		ID:         id,
		Name:       name,
		TypeLine:   typeLine,
		Rarity:     rarity,
		Legalities: legalities,
	}
}

func TestDetermineDeckLegalityForFormat_StandardIllegalCard(t *testing.T) {
	cards := []*domain.Card{
		makeLegalityCard("bolt", "Lightning Bolt", "Instant", "common", map[string]string{"standard": "not_legal", "modern": "legal"}),
	}
	quantities := map[string]int{"bolt": 60}

	res := usecase.DetermineDeckLegalityForFormat(cards, quantities, "standard")
	if res.IsLegal {
		t.Fatalf("expected standard deck to be illegal")
	}
	if len(res.IllegalCards) == 0 {
		t.Fatalf("expected illegal card details")
	}
}

func TestDetermineDeckLegalityForFormat_VintageRestrictedLimit(t *testing.T) {
	cards := []*domain.Card{
		makeLegalityCard("lotus", "Black Lotus", "Artifact", "rare", map[string]string{"vintage": "restricted"}),
		makeLegalityCard("island", "Island", "Basic Land - Island", "common", map[string]string{"vintage": "legal"}),
	}
	quantities := map[string]int{"lotus": 2, "island": 58}

	res := usecase.DetermineDeckLegalityForFormat(cards, quantities, "vintage")
	if res.IsLegal {
		t.Fatalf("expected vintage deck with 2 restricted cards to be illegal")
	}
}

func TestDetermineDeckLegalityForFormat_CommanderSingleton(t *testing.T) {
	cards := []*domain.Card{
		makeLegalityCard("solring", "Sol Ring", "Artifact", "uncommon", map[string]string{"commander": "legal"}),
		makeLegalityCard("forest", "Forest", "Basic Land - Forest", "common", map[string]string{"commander": "legal"}),
	}
	quantities := map[string]int{"solring": 2, "forest": 98}

	res := usecase.DetermineDeckLegalityForFormat(cards, quantities, "commander")
	if res.IsLegal {
		t.Fatalf("expected commander deck with duplicate non-basic to be illegal")
	}
}

func TestDetermineDeckLegalityForCommanderFormat_SingletonRuleAndCommanderSeparation(t *testing.T) {
	// Test that commander-specific validation enforces:
	// 1. Singleton rule for main deck (excluding commander)
	// 2. Commander must not appear in main deck
	// 3. Deck must be exactly 100 cards (including commander)
	cards := []*domain.Card{
		makeLegalityCard("atraxa", "Atraxa, Praetors' Voice", "Creature", "mythic", map[string]string{"commander": "legal"}),
		makeLegalityCard("solring", "Sol Ring", "Artifact", "uncommon", map[string]string{"commander": "legal"}),
		makeLegalityCard("forest", "Forest", "Basic Land - Forest", "common", map[string]string{"commander": "legal"}),
	}
	// Atraxa is the commander (1 copy), Sol Ring has 2 copies (singleton violation),
	// and Forest fills the rest. Total: 1 (Atraxa) + 2 (Sol Ring) + 97 (Forest) = 100
	quantities := map[string]int{"atraxa": 1, "solring": 2, "forest": 97}
	commanderIDs := map[string]bool{"atraxa": true}

	res := usecase.DetermineDeckLegalityForCommanderFormat(cards, quantities, commanderIDs)
	if res.IsLegal {
		t.Fatalf("expected commander deck with duplicate Sol Ring to be illegal")
	}
	if len(res.IllegalCards) == 0 {
		t.Fatalf("expected illegal card issues for Sol Ring")
	}

	// Verify the issue is about singleton.
	found := false
	for _, issue := range res.IllegalCards {
		if issue.CardName == "Sol Ring" && strings.Contains(issue.Reason, "singleton rule") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected singleton violation for Sol Ring; got issues: %+v", res.IllegalCards)
	}
}

func TestDetermineDeckLegalityForCommanderFormat_CommanderInMainDeck(t *testing.T) {
	// Test that having the commander card in the main deck is flagged as illegal
	cards := []*domain.Card{
		makeLegalityCard("atraxa", "Atraxa, Praetors' Voice", "Creature", "mythic", map[string]string{"commander": "legal"}),
		makeLegalityCard("forest", "Forest", "Basic Land - Forest", "common", map[string]string{"commander": "legal"}),
	}
	// Atraxa appears twice: once as commander (1x) and once in the main deck (1x).
	// This totals to 2x Atraxa, which is a singleton violation (and conceptually wrong).
	quantities := map[string]int{"atraxa": 2, "forest": 98}
	commanderIDs := map[string]bool{"atraxa": true}

	res := usecase.DetermineDeckLegalityForCommanderFormat(cards, quantities, commanderIDs)
	if res.IsLegal {
		t.Fatalf("expected commander deck with commander card in main deck to be illegal")
	}

	// Should have an issue about deck size or singleton violation on Atraxa.
	if len(res.IllegalCards) == 0 && len(res.Issues) == 0 {
		t.Fatalf("expected illegal card issues or deck size issues")
	}
}

func TestDetermineDeckLegalityForCommanderFormat_ValidDeck(t *testing.T) {
	// Test that a valid commander deck passes
	cards := []*domain.Card{
		makeLegalityCard("atraxa", "Atraxa, Praetors' Voice", "Creature", "mythic", map[string]string{"commander": "legal"}),
		makeLegalityCard("solring", "Sol Ring", "Artifact", "uncommon", map[string]string{"commander": "legal"}),
		makeLegalityCard("forest", "Forest", "Basic Land - Forest", "common", map[string]string{"commander": "legal"}),
	}
	// Valid: Atraxa (1x) + Sol Ring (1x) + Forest (98x) = 100
	quantities := map[string]int{"atraxa": 1, "solring": 1, "forest": 98}
	commanderIDs := map[string]bool{"atraxa": true}

	res := usecase.DetermineDeckLegalityForCommanderFormat(cards, quantities, commanderIDs)
	if !res.IsLegal {
		t.Fatalf("expected valid commander deck to be legal; got issues: %+v, illegal: %+v", res.Issues, res.IllegalCards)
	}
}

func TestDetermineDeckLegalityForFormat_PauperIllegalCard(t *testing.T) {
	// Pauper legality is determined by Scryfall's legalities["pauper"] field,
	// not by the card's rarity field. A card with pauper="not_legal" is illegal
	// regardless of its rarity; a card with pauper="legal" is legal even if rare
	// (because it was printed as common in some set).
	cards := []*domain.Card{
		makeLegalityCard("common", "Good Common", "Instant", "common", map[string]string{"pauper": "legal"}),
		makeLegalityCard("notlegal", "Snapcaster Mage", "Creature", "rare", map[string]string{"pauper": "not_legal"}),
	}
	quantities := map[string]int{"common": 56, "notlegal": 4}

	res := usecase.DetermineDeckLegalityForFormat(cards, quantities, "pauper")
	if res.IsLegal {
		t.Fatalf("expected pauper deck with not_legal card to be illegal")
	}
}

func TestDetermineDeckLegalityForFormat_PauperLegalRarePrint(t *testing.T) {
	// A card printed at rare in its latest set but also printed as common
	// in another set is legal in Pauper — Scryfall marks it pauper="legal".
	// Use 4 copies to stay within the copy limit.
	cards := []*domain.Card{
		makeLegalityCard("counterspell", "Counterspell", "Instant", "uncommon", map[string]string{"pauper": "legal"}),
		makeLegalityCard("island", "Island", "Basic Land - Island", "common", map[string]string{"pauper": "legal"}),
	}
	quantities := map[string]int{"counterspell": 4, "island": 56}

	res := usecase.DetermineDeckLegalityForFormat(cards, quantities, "pauper")
	if !res.IsLegal {
		t.Fatalf("expected Counterspell (pauper=legal) to be legal in pauper; got illegal: %+v", res.IllegalCards)
	}
}

func TestDetermineDeckLegalityAllFormats_ReturnsAll(t *testing.T) {
	cards := []*domain.Card{
		makeLegalityCard("island", "Island", "Basic Land - Island", "common", map[string]string{
			"standard": "legal", "pioneer": "legal", "modern": "legal", "legacy": "legal", "vintage": "legal", "commander": "legal", "pauper": "legal",
		}),
	}
	quantities := map[string]int{"island": 60}

	res := usecase.DetermineDeckLegalityAllFormats(cards, quantities)
	if len(res) != len(domain.SupportedFormats) {
		t.Fatalf("expected %d format results, got %d", len(domain.SupportedFormats), len(res))
	}
}
