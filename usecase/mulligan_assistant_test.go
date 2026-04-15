package usecase_test

import (
	"context"
	"testing"

	"github.com/gigliofr/mana-wise/usecase"
)

func TestMulliganAssistant_BasicSimulation(t *testing.T) {
	uc := usecase.NewMulliganAssistantUseCase(nil)

	deck := ""
	for i := 0; i < 24; i++ {
		deck += "1 Mountain\n"
	}
	for i := 0; i < 36; i++ {
		deck += "1 Lightning Strike\n"
	}

	res, err := uc.Execute(context.Background(), usecase.MulliganSimulationRequest{
		Decklist:   deck,
		Format:     "standard",
		Archetype:  "aggro",
		Iterations: 200,
		OnPlay:     true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(res.Summaries) != 3 {
		t.Fatalf("expected 3 summaries, got %d", len(res.Summaries))
	}
	for _, s := range res.Summaries {
		if s.KeepRate < 0 || s.KeepRate > 1 {
			t.Fatalf("invalid keep rate for hand %d: %f", s.HandSize, s.KeepRate)
		}
	}
}

func TestMulliganAssistant_EmptyDecklist(t *testing.T) {
	uc := usecase.NewMulliganAssistantUseCase(nil)
	_, err := uc.Execute(context.Background(), usecase.MulliganSimulationRequest{})
	if err == nil {
		t.Fatal("expected error for empty decklist")
	}
}
