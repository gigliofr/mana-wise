package usecase

import (
	"testing"

	"github.com/gigliofr/mana-wise/domain"
)

func mkCard(id, name, oracle string) *domain.Card {
	return &domain.Card{
		ID:         id,
		Name:       name,
		OracleText: oracle,
		TypeLine:   "Sorcery",
		CMC:        2,
	}
}

func TestCommanderBrackets_Bracket1_NoSignals(t *testing.T) {
	cfg := domain.DefaultCommanderBracketConfig()
	uc := NewCommanderBracketUseCase(&cfg)

	cards := []*domain.Card{
		{ID: "a", Name: "Austere Command", OracleText: "Choose two.", TypeLine: "Sorcery", CMC: 6},
		{ID: "b", Name: "Merciless Eviction", OracleText: "Choose one.", TypeLine: "Sorcery", CMC: 6},
		{ID: "c", Name: "Sun Titan", OracleText: "Whenever Sun Titan enters the battlefield or attacks, return target permanent card with mana value 3 or less from your graveyard to the battlefield.", TypeLine: "Creature — Giant", CMC: 6},
	}
	qty := map[string]int{"a": 1, "b": 1, "c": 1}

	got := uc.Evaluate(cards, qty)
	if got == nil {
		t.Fatalf("expected assessment, got nil")
	}
	if got.Bracket != 1 {
		t.Fatalf("expected bracket 1, got %d", got.Bracket)
	}
}

func TestCommanderBrackets_Bracket2_ModestOptimizationNoForbidden(t *testing.T) {
	cfg := domain.DefaultCommanderBracketConfig()
	uc := NewCommanderBracketUseCase(&cfg)

	cards := []*domain.Card{
		mkCard("a", "Eladamri's Call", "Search your library for a creature card, reveal it, put it into your hand."),
		mkCard("b", "Arcane Signet", "{T}: Add one mana of any color."),
	}
	qty := map[string]int{"a": 1, "b": 1}

	got := uc.Evaluate(cards, qty)
	if got == nil {
		t.Fatalf("expected assessment, got nil")
	}
	if got.Bracket != 2 {
		t.Fatalf("expected bracket 2, got %d", got.Bracket)
	}
}

func TestCommanderBrackets_Bracket3_UpToThreeGameChangers_NoForbidden(t *testing.T) {
	cfg := domain.DefaultCommanderBracketConfig()
	uc := NewCommanderBracketUseCase(&cfg)

	cards := []*domain.Card{
		mkCard("a", "Rhystic Study", "Whenever an opponent casts a spell, you may draw a card unless that player pays {1}."),
		mkCard("b", "Mystic Remora", "Cumulative upkeep {1}."),
		mkCard("c", "Mana Crypt", "At the beginning of your upkeep, flip a coin."),
	}
	qty := map[string]int{"a": 1, "b": 1, "c": 1}

	got := uc.Evaluate(cards, qty)
	if got == nil {
		t.Fatalf("expected assessment, got nil")
	}
	if got.Bracket != 3 {
		t.Fatalf("expected bracket 3, got %d", got.Bracket)
	}
}

func TestCommanderBrackets_Bracket4_ForbiddenPatternPresent(t *testing.T) {
	cfg := domain.DefaultCommanderBracketConfig()
	uc := NewCommanderBracketUseCase(&cfg)

	cards := []*domain.Card{
		mkCard("a", "Armageddon", "Destroy all lands."),
		mkCard("b", "Sol Ring", "{T}: Add {C}{C}."),
	}
	qty := map[string]int{"a": 1, "b": 1}

	got := uc.Evaluate(cards, qty)
	if got == nil {
		t.Fatalf("expected assessment, got nil")
	}
	if got.Bracket != 4 {
		t.Fatalf("expected bracket 4, got %d", got.Bracket)
	}
}

func TestCommanderBrackets_Bracket5_CompetitiveDensity(t *testing.T) {
	cfg := domain.DefaultCommanderBracketConfig()
	cfg.CedhTutorThreshold = 2
	cfg.CedhFastManaThreshold = 1
	cfg.CedhComboThreshold = 2
	uc := NewCommanderBracketUseCase(&cfg)

	cards := []*domain.Card{
		mkCard("a", "Demonic Tutor", "Search your library for a card, put it into your hand."),
		mkCard("b", "Vampiric Tutor", "Search your library for a card, then shuffle and put that card on top."),
		mkCard("c", "Mana Crypt", "{T}: Add {C}{C}."),
		mkCard("d", "Thassa's Oracle", "When Thassa's Oracle enters the battlefield, look at the top X cards..."),
		mkCard("e", "Demonic Consultation", "Name a card. Exile the top six cards of your library..."),
	}
	qty := map[string]int{"a": 1, "b": 1, "c": 1, "d": 1, "e": 1}

	got := uc.Evaluate(cards, qty)
	if got == nil {
		t.Fatalf("expected assessment, got nil")
	}
	if got.Bracket != 5 {
		t.Fatalf("expected bracket 5, got %d", got.Bracket)
	}
}

func TestCommanderBrackets_ValueRecursionDeck_StaysBracket2(t *testing.T) {
	cfg := domain.DefaultCommanderBracketConfig()
	uc := NewCommanderBracketUseCase(&cfg)

	cards := []*domain.Card{
		mkCard("a", "Sol Ring", "{T}: Add {C}{C}.") ,
		mkCard("b", "Wayfarer's Bauble", "{2}, {T}, Sacrifice Wayfarer's Bauble: Search your library for a basic land card, put that card onto the battlefield tapped, then shuffle."),
		mkCard("c", "Commander's Sphere", "{T}: Add one mana of any color.") ,
		mkCard("d", "Reanimate", "Put target creature card from a graveyard onto the battlefield under your control. You lose life equal to its mana value."),
		mkCard("e", "Rise of the Dark Realms", "Put all creature cards from all graveyards onto the battlefield under your control."),
		mkCard("f", "Ruinous Ultimatum", "Destroy all nonland permanents your opponents control."),
		mkCard("g", "Morbid Opportunist", "Whenever one or more other creatures die, draw a card. This ability triggers only once each turn."),
	}
	qty := map[string]int{"a": 1, "b": 1, "c": 1, "d": 1, "e": 1, "f": 1, "g": 1}

	got := uc.Evaluate(cards, qty)
	if got == nil {
		t.Fatalf("expected assessment, got nil")
	}
	if got.Bracket != 2 {
		t.Fatalf("expected bracket 2, got %d (%s; signals=%+v)", got.Bracket, got.Label, got.Signals)
	}
}
