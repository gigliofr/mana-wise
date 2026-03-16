package usecase_test

import (
	"context"
	"testing"

	"github.com/manawise/api/usecase"
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

func TestMatchupSimulator_EmptyDecklist(t *testing.T) {
	uc := usecase.NewMatchupSimulatorUseCase(nil)

	_, err := uc.Execute(context.Background(), usecase.MatchupSimulationRequest{})
	if err == nil {
		t.Fatalf("expected error for empty decklist")
	}
}
