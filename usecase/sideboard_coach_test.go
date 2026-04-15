package usecase_test

import (
	"context"
	"testing"

	"github.com/gigliofr/mana-wise/usecase"
)

func TestSideboardCoach_ControlPlanHasInsOuts(t *testing.T) {
	uc := usecase.NewSideboardCoachUseCase(nil)

	main := "4 Lightning Strike\n4 Negate\n4 Impulse\n4 Brotherhood's End\n4 Go for the Throat\n4 Island\n4 Mountain\n4 Swamp"
	side := "2 Duress\n2 Disdainful Stroke\n2 Abrade\n2 Unlicensed Hearse\n2 Lithomantic Barrage"

	plan, err := uc.Execute(context.Background(), usecase.SideboardPlanRequest{
		MainDecklist:      main,
		SideboardDecklist: side,
		OpponentArchetype: "control",
		Format:            "standard",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if plan.Matchup != "control" {
		t.Fatalf("expected matchup control, got %s", plan.Matchup)
	}
	if len(plan.Ins) == 0 {
		t.Fatal("expected non-empty sideboard ins")
	}
	if len(plan.Outs) == 0 {
		t.Fatal("expected non-empty sideboard outs")
	}
}

func TestSideboardCoach_EmptyInputsValidation(t *testing.T) {
	uc := usecase.NewSideboardCoachUseCase(nil)

	_, err := uc.Execute(context.Background(), usecase.SideboardPlanRequest{
		MainDecklist:      "",
		SideboardDecklist: "2 Duress",
		OpponentArchetype: "control",
	})
	if err == nil {
		t.Fatal("expected error for empty main deck")
	}

	_, err = uc.Execute(context.Background(), usecase.SideboardPlanRequest{
		MainDecklist:      "4 Island",
		SideboardDecklist: "",
		OpponentArchetype: "control",
	})
	if err == nil {
		t.Fatal("expected error for empty sideboard")
	}
}
