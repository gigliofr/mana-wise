package usecase

import (
	"context"
	"errors"
	"sync"
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

func mkCardMV(id, name string, mv float64) *domain.Card {
	c := mkCard(id, name, "")
	c.CMC = mv
	return c
}

func evaluate(t *testing.T, cfg domain.CommanderBracketConfig, cards ...*domain.Card) *domain.CommanderBracketAssessment {
	t.Helper()
	qty := map[string]int{}
	for _, c := range cards {
		qty[c.ID] = 1
	}
	got := NewCommanderBracketUseCase(&cfg).Evaluate(cards, qty)
	if got == nil {
		t.Fatal("expected assessment, got nil")
	}
	return got
}

func TestDefaultGameChangers_Has53UniqueCards(t *testing.T) {
	list := domain.DefaultGameChangers()
	if len(list) != 53 {
		t.Fatalf("expected 53 Game Changers (February 2026 list), got %d", len(list))
	}
	seen := map[string]bool{}
	for _, n := range list {
		k := domain.NormalizeCommanderCardName(n)
		if seen[k] {
			t.Fatalf("duplicate Game Changer %q", n)
		}
		seen[k] = true
	}
	for _, banned := range []string{"Mana Crypt", "Jeweled Lotus", "Dockside Extortionist", "Food Chain"} {
		if seen[domain.NormalizeCommanderCardName(banned)] {
			t.Fatalf("%q is not on the current Game Changers list", banned)
		}
	}
}

func TestCommanderBrackets_NoSignals_IsCoreNotExhibition(t *testing.T) {
	got := evaluate(t, domain.DefaultCommanderBracketConfig(),
		&domain.Card{ID: "a", Name: "Austere Command", OracleText: "Choose two.", TypeLine: "Sorcery", CMC: 6},
		&domain.Card{ID: "b", Name: "Sun Titan", OracleText: "Vigilance", TypeLine: "Creature — Giant", CMC: 6},
	)
	if got.Bracket != 2 || got.Label != "Core" {
		t.Fatalf("expected bracket 2 Core, got %d %s", got.Bracket, got.Label)
	}
}

// October 2025: tutors no longer raise the bracket.
func TestCommanderBrackets_TutorsDoNotRaiseBracket(t *testing.T) {
	got := evaluate(t, domain.DefaultCommanderBracketConfig(),
		mkCard("a", "Eladamri's Call", "Search your library for a creature card, reveal it, put it into your hand."),
		mkCard("b", "Diabolic Tutor", "Search your library for a card, put that card into your hand, then shuffle."),
		mkCard("c", "Arcane Signet", "{T}: Add one mana of any color."),
	)
	if got.Bracket != 2 {
		t.Fatalf("expected bracket 2, got %d (%s)", got.Bracket, got.Reason)
	}
	if len(got.Signals.TutorCards) != 2 {
		t.Fatalf("tutors should still be reported, got %v", got.Signals.TutorCards)
	}
}

func TestCommanderBrackets_UpToThreeGameChangers_IsUpgraded(t *testing.T) {
	got := evaluate(t, domain.DefaultCommanderBracketConfig(),
		mkCard("a", "Rhystic Study", ""),
		mkCard("b", "Smothering Tithe", ""),
		mkCard("c", "Cyclonic Rift", ""),
		mkCard("d", "Mystic Remora", ""), // removed from the list in October 2025
	)
	if got.Bracket != 3 || got.Label != "Upgraded" {
		t.Fatalf("expected bracket 3 Upgraded, got %d %s (%v)", got.Bracket, got.Label, got.Signals.GameChangerCards)
	}
	if len(got.Signals.GameChangerCards) != 3 {
		t.Fatalf("expected 3 Game Changers, got %v", got.Signals.GameChangerCards)
	}
}

func TestCommanderBrackets_FourGameChangers_IsOptimized(t *testing.T) {
	got := evaluate(t, domain.DefaultCommanderBracketConfig(),
		mkCard("a", "Rhystic Study", ""),
		mkCard("b", "Smothering Tithe", ""),
		mkCard("c", "Cyclonic Rift", ""),
		mkCard("d", "Demonic Tutor", "Search your library for a card, put that card into your hand, then shuffle."),
	)
	if got.Bracket != 4 {
		t.Fatalf("expected bracket 4, got %d (%s)", got.Bracket, got.Reason)
	}
}

// Several Game Changers are lands: they must be counted.
func TestCommanderBrackets_GameChangerLandsAreCounted(t *testing.T) {
	cradle := &domain.Card{ID: "l", Name: "Gaea's Cradle", TypeLine: "Legendary Land", CMC: 0}
	got := evaluate(t, domain.DefaultCommanderBracketConfig(), cradle, mkCard("x", "Llanowar Elves", "{T}: Add {G}."))
	if got.Bracket != 3 || len(got.Signals.GameChangerCards) != 1 {
		t.Fatalf("expected Gaea's Cradle to count as Game Changer, got bracket %d signals %v", got.Bracket, got.Signals.GameChangerCards)
	}
}

// The Scryfall game_changer flag is honoured even if the name is not in the list.
func TestCommanderBrackets_ScryfallFlagCountsAsGameChanger(t *testing.T) {
	newGC := mkCard("n", "Some Future Card", "")
	newGC.GameChanger = true
	got := evaluate(t, domain.DefaultCommanderBracketConfig(), newGC)
	if got.Bracket != 3 {
		t.Fatalf("expected bracket 3, got %d", got.Bracket)
	}
}

func TestCommanderBrackets_MassLandDenial_IsOptimized(t *testing.T) {
	got := evaluate(t, domain.DefaultCommanderBracketConfig(),
		mkCard("a", "Armageddon", "Destroy all lands."),
		mkCard("b", "Sol Ring", "{T}: Add {C}{C}."),
	)
	if got.Bracket != 4 {
		t.Fatalf("expected bracket 4, got %d", got.Bracket)
	}
}

func TestCommanderBrackets_ExtraTurns_FewAllowedChainIsOptimized(t *testing.T) {
	cfg := domain.DefaultCommanderBracketConfig()
	few := evaluate(t, cfg, mkCard("a", "Time Warp", "Target player takes an extra turn after this one."))
	if few.Bracket != 2 {
		t.Fatalf("one extra turn: expected bracket 2, got %d", few.Bracket)
	}
	chain := evaluate(t, cfg,
		mkCard("a", "Time Warp", "Target player takes an extra turn after this one."),
		mkCard("b", "Temporal Manipulation", "Take an extra turn after this one."),
		mkCard("c", "Capture of Jingzhou", "Take an extra turn after this one."),
	)
	if chain.Bracket != 4 {
		t.Fatalf("three extra turns: expected bracket 4, got %d", chain.Bracket)
	}
}

func TestCommanderBrackets_EarlyTwoCardCombo_IsOptimized(t *testing.T) {
	got := evaluate(t, domain.DefaultCommanderBracketConfig(),
		mkCardMV("a", "Devoted Druid", 2),
		mkCardMV("b", "Vizier of Remedies", 2),
	)
	if got.Bracket != 4 {
		t.Fatalf("expected bracket 4, got %d (%s)", got.Bracket, got.Reason)
	}
	if len(got.Signals.TwoCardCombos) != 1 || !got.Signals.TwoCardCombos[0].Early {
		t.Fatalf("expected one early combo, got %+v", got.Signals.TwoCardCombos)
	}
}

func TestCommanderBrackets_LateTwoCardCombo_IsUpgraded(t *testing.T) {
	got := evaluate(t, domain.DefaultCommanderBracketConfig(),
		mkCardMV("a", "Sanguine Bond", 5),
		mkCardMV("b", "Exquisite Blood", 5),
	)
	if got.Bracket != 3 {
		t.Fatalf("expected bracket 3, got %d (%s)", got.Bracket, got.Reason)
	}
	if len(got.Signals.TwoCardCombos) != 1 || got.Signals.TwoCardCombos[0].Early {
		t.Fatalf("expected one late combo, got %+v", got.Signals.TwoCardCombos)
	}
}

func TestCommanderBrackets_HalfACombo_IsNotDetected(t *testing.T) {
	got := evaluate(t, domain.DefaultCommanderBracketConfig(), mkCardMV("a", "Walking Ballista", 0))
	if len(got.Signals.TwoCardCombos) != 0 || got.Bracket != 2 {
		t.Fatalf("a single combo piece must not count, got %+v bracket %d", got.Signals.TwoCardCombos, got.Bracket)
	}
}

func TestCommanderBrackets_BannedCardsAreFlaggedNotCounted(t *testing.T) {
	crypt := mkCard("a", "Mana Crypt", "{T}: Add {C}{C}.")
	crypt.Legalities = map[string]string{"commander": "banned"}
	got := evaluate(t, domain.DefaultCommanderBracketConfig(), crypt)
	if got.Legal {
		t.Fatal("deck with a banned card must not be legal")
	}
	if len(got.Signals.BannedCards) != 1 || len(got.Signals.GameChangerCards) != 0 {
		t.Fatalf("expected Mana Crypt as banned only, got banned=%v gc=%v", got.Signals.BannedCards, got.Signals.GameChangerCards)
	}
}

func TestCommanderBrackets_CompetitiveDensity_IsCEDH(t *testing.T) {
	cfg := domain.DefaultCommanderBracketConfig()
	cfg.CedhGameChangerThreshold = 2
	cfg.CedhFastManaThreshold = 1
	cfg.CedhTutorThreshold = 2
	got := evaluate(t, cfg,
		mkCard("a", "Demonic Tutor", "Search your library for a card, put that card into your hand, then shuffle."),
		mkCard("b", "Vampiric Tutor", "Search your library for a card, then shuffle and put that card on top."),
		mkCard("c", "Chrome Mox", "Imprint. {T}: Add one mana of any of the exiled card's colors. mox"),
		mkCardMV("d", "Thassa's Oracle", 2),
		mkCardMV("e", "Demonic Consultation", 1),
	)
	if got.Bracket != 5 || got.Label != "cEDH" {
		t.Fatalf("expected bracket 5 cEDH, got %d %s (%s)", got.Bracket, got.Label, got.Reason)
	}
}

func TestCommanderBrackets_ConfigIsConcurrencySafe(t *testing.T) {
	uc := NewCommanderBracketUseCase(nil)
	cards := []*domain.Card{mkCard("a", "Rhystic Study", "")}
	qty := map[string]int{"a": 1}
	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(2)
		go func() { defer wg.Done(); uc.Evaluate(cards, qty) }()
		go func() { defer wg.Done(); uc.SetGameChangers(domain.DefaultGameChangers()) }()
	}
	wg.Wait()
}

type fakeGCSource struct {
	names []string
	err   error
}

func (f fakeGCSource) GameChangerNames(context.Context) ([]string, error) { return f.names, f.err }

func TestSyncGameChangers_AppliesPlausibleListAndKeepsCurrentOtherwise(t *testing.T) {
	uc := NewCommanderBracketUseCase(nil)

	if _, err := SyncGameChangers(context.Background(), fakeGCSource{err: errors.New("down")}, uc); err == nil {
		t.Fatal("expected error from failing source")
	}
	if _, err := SyncGameChangers(context.Background(), fakeGCSource{names: []string{"Only One"}}, uc); err == nil {
		t.Fatal("expected implausible size to be rejected")
	}
	if got := len(uc.Config().GameChangers); got != 53 {
		t.Fatalf("current list must be kept, got %d cards", got)
	}

	fresh := append(domain.DefaultGameChangers(), "Brand New Game Changer")
	n, err := SyncGameChangers(context.Background(), fakeGCSource{names: fresh}, uc)
	if err != nil || n != 54 {
		t.Fatalf("expected sync of 54 cards, got n=%d err=%v", n, err)
	}
	got := evaluate(t, uc.Config(), mkCard("z", "Brand New Game Changer", ""))
	if got.Bracket != 3 {
		t.Fatalf("synced card must count as Game Changer, got bracket %d", got.Bracket)
	}
}
