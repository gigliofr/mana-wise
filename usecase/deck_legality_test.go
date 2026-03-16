package usecase_test

import (
	"testing"

	"github.com/manawise/api/domain"
	"github.com/manawise/api/usecase"
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

func TestDetermineDeckLegalityForFormat_PauperNeedsCommon(t *testing.T) {
	cards := []*domain.Card{
		makeLegalityCard("common", "Good Common", "Instant", "common", map[string]string{"pauper": "legal"}),
		makeLegalityCard("rare", "Rare Bomb", "Creature", "rare", map[string]string{"pauper": "legal"}),
	}
	quantities := map[string]int{"common": 56, "rare": 4}

	res := usecase.DetermineDeckLegalityForFormat(cards, quantities, "pauper")
	if res.IsLegal {
		t.Fatalf("expected pauper deck with rare card to be illegal")
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
