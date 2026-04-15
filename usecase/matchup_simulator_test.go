package usecase_test

import (
	"context"
	"strings"
	"testing"

	"github.com/gigliofr/mana-wise/usecase"
)

func TestMatchupSimulator_Basic(t *testing.T) {
	uc := usecase.NewMatchupSimulatorUseCase(nil)

	deck := "4 Consider\n4 Opt\n4 Negate\n4 Go for the Throat\n4 Sheoldred, the Apocalypse\n24 Island\n16 Swamp"
	res, err := uc.Execute(context.Background(), usecase.MatchupSimulationRequest{
		Decklist: deck,
		Format:   "standard",
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(res.Matchups) != 4 {
		t.Fatalf("expected 4 default matchups, got %d", len(res.Matchups))
	}
	if res.PlayerArchetype == "" {
		t.Fatalf("expected inferred player archetype")
	}
	for _, m := range res.Matchups {
		if m.WinRate <= 0 || m.WinRate >= 1 {
			t.Fatalf("expected winrate in (0,1), got %.2f", m.WinRate)
		}
		if m.Confidence <= 0 || m.Confidence > 1 {
			t.Fatalf("expected confidence in (0,1], got %.2f", m.Confidence)
		}
	}
}

func TestMatchupSimulator_CustomOpponentsAndArchetype(t *testing.T) {
	uc := usecase.NewMatchupSimulatorUseCase(nil)

	deck := "4 Monastery Swiftspear\n4 Lightning Strike\n4 Play with Fire\n4 Kumano Faces Kakkazan\n20 Mountain"
	res, err := uc.Execute(context.Background(), usecase.MatchupSimulationRequest{
		Decklist:        deck,
		Format:          "standard",
		PlayerArchetype: "aggro",
		Opponents:       []string{"control", "combo", "control"},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if res.PlayerArchetype != "aggro" {
		t.Fatalf("expected archetype aggro, got %s", res.PlayerArchetype)
	}
	if len(res.Matchups) != 2 {
		t.Fatalf("expected deduplicated opponents, got %d", len(res.Matchups))
	}
}

func TestMatchupSimulator_PostBoardAdjustment(t *testing.T) {
	uc := usecase.NewMatchupSimulatorUseCase(nil)

	deck := "4 Consider\n4 Opt\n4 Negate\n4 Go for the Throat\n4 Sheoldred, the Apocalypse\n24 Island\n16 Swamp"
	side := "3 Duress\n2 Negate\n2 Brotherhood's End\n2 Unlicensed Hearse"
	res, err := uc.Execute(context.Background(), usecase.MatchupSimulationRequest{
		Decklist:          deck,
		SideboardDecklist: side,
		Format:            "standard",
		Opponents:         []string{"aggro", "combo"},
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(res.Matchups) != 2 {
		t.Fatalf("expected 2 matchups, got %d", len(res.Matchups))
	}
	for _, m := range res.Matchups {
		if m.PostBoardWinRate < m.WinRate {
			t.Fatalf("expected post-board >= pre-board for %s, got %.2f < %.2f", m.OpponentArchetype, m.PostBoardWinRate, m.WinRate)
		}
		if len(m.SuggestedIns) == 0 {
			t.Fatalf("expected suggested_ins for %s", m.OpponentArchetype)
		}
		if len(m.SuggestedOuts) == 0 {
			t.Fatalf("expected suggested_outs for %s", m.OpponentArchetype)
		}
	}
}

func TestMatchupSimulator_WeaknessDiagnosis(t *testing.T) {
	uc := usecase.NewMatchupSimulatorUseCase(nil)

	// Aggro deck: few counters/discard → should be unfavoured vs control/combo
	deck := "4 Monastery Swiftspear\n4 Lightning Strike\n4 Play with Fire\n4 Kumano Faces Kakkazan\n4 Goblin Guide\n20 Mountain"
	res, err := uc.Execute(context.Background(), usecase.MatchupSimulationRequest{
		Decklist:        deck,
		Format:          "standard",
		PlayerArchetype: "aggro",
		Opponents:       []string{"aggro", "control", "combo"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Weaknesses should only include matchups with WR < 0.50
	for _, w := range res.Weaknesses {
		var found bool
		for _, m := range res.Matchups {
			if m.OpponentArchetype == w.Opponent {
				if m.WinRate >= 0.50 {
					t.Errorf("weakness listed for %s but WR=%.2f >= 0.50", w.Opponent, m.WinRate)
				}
				found = true
				break
			}
		}
		if !found {
			t.Errorf("weakness opponent %q not found in matchups", w.Opponent)
		}
		if w.Severity == "" {
			t.Errorf("expected non-empty severity for %s", w.Opponent)
		}
		if len(w.Gaps) == 0 {
			t.Errorf("expected at least one gap for %s", w.Opponent)
		}
		if len(w.Remedies) == 0 {
			t.Errorf("expected at least one remedy for %s", w.Opponent)
		}
	}

	// Summary should mention unfavored matchup count when weaknesses exist
	if len(res.Weaknesses) > 0 && !strings.Contains(res.Summary, "unfavored matchup") {
		t.Errorf("expected summary to mention unfavored matchups, got: %s", res.Summary)
	}
}

func TestMatchupSimulator_MetaWeightedWinRate(t *testing.T) {
	uc := usecase.NewMatchupSimulatorUseCase(nil)

	deck := "4 Consider\n4 Opt\n4 Negate\n4 Go for the Throat\n4 Sheoldred, the Apocalypse\n24 Island\n16 Swamp"
	// default 4 opponents, standard format
	res, err := uc.Execute(context.Background(), usecase.MatchupSimulationRequest{
		Decklist: deck,
		Format:   "standard",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if res.MetaWeightedWinRate <= 0 || res.MetaWeightedWinRate >= 1 {
		t.Fatalf("meta_weighted_win_rate out of range: %.4f", res.MetaWeightedWinRate)
	}
	// All meta shares must sum to ~1.0
	total := 0.0
	for _, m := range res.Matchups {
		if m.MetaShare <= 0 {
			t.Fatalf("meta_share must be > 0 for %s, got %.4f", m.OpponentArchetype, m.MetaShare)
		}
		total += m.MetaShare
	}
	if total < 0.99 || total > 1.01 {
		t.Fatalf("meta shares should sum to ~1.0, got %.4f", total)
	}
	// Custom meta shares override: favor combo-heavy meta
	resCombo, err := uc.Execute(context.Background(), usecase.MatchupSimulationRequest{
		Decklist: deck,
		Format:   "standard",
		MetaShares: map[string]float64{"aggro": 0.05, "midrange": 0.10, "control": 0.15, "combo": 0.70},
	})
	if err != nil {
		t.Fatalf("unexpected error with custom meta shares: %v", err)
	}
	// With 70% combo weight the meta-weighted WR should differ from equal-weight result
	if res.MetaWeightedWinRate == resCombo.MetaWeightedWinRate {
		t.Fatalf("expected different meta-weighted WR with custom meta shares")
	}
	// Summary must mention meta-weighted percentage
	if !strings.Contains(resCombo.Summary, "meta-weighted") {
		t.Fatalf("expected summary to contain 'meta-weighted', got: %s", resCombo.Summary)
	}
}

func TestMatchupSimulator_OnPlay(t *testing.T) {
	uc := usecase.NewMatchupSimulatorUseCase(nil)

	deck := "4 Monastery Swiftspear\n4 Lightning Strike\n4 Kumano Faces Kakkazan\n20 Mountain\n4 Play with Fire\n4 Goblin Guide"
	opponents := []string{"control", "combo"}

	resPlay, err := uc.Execute(context.Background(), usecase.MatchupSimulationRequest{
		Decklist:        deck,
		Format:          "standard",
		PlayerArchetype: "aggro",
		Opponents:       opponents,
		OnPlay:          true,
	})
	if err != nil {
		t.Fatalf("on_play=true error: %v", err)
	}
	resDraw, err := uc.Execute(context.Background(), usecase.MatchupSimulationRequest{
		Decklist:        deck,
		Format:          "standard",
		PlayerArchetype: "aggro",
		Opponents:       opponents,
		OnPlay:          false,
	})
	if err != nil {
		t.Fatalf("on_play=false error: %v", err)
	}

	if !resPlay.OnPlay {
		t.Fatal("expected OnPlay=true reflected in result")
	}
	if resDraw.OnPlay {
		t.Fatal("expected OnPlay=false reflected in result")
	}

	// On-play aggro should have higher win rate than on-draw across all matchups
	for i, mp := range resPlay.Matchups {
		md := resDraw.Matchups[i]
		if mp.WinRate <= md.WinRate {
			t.Errorf("expected on_play WR > on_draw WR for %s: %.3f <= %.3f", mp.OpponentArchetype, mp.WinRate, md.WinRate)
		}
	}
}

func TestMatchupSimulator_EmptyDecklist(t *testing.T) {
	uc := usecase.NewMatchupSimulatorUseCase(nil)

	_, err := uc.Execute(context.Background(), usecase.MatchupSimulationRequest{})
	if err == nil {
		t.Fatalf("expected error for empty decklist")
	}
}
