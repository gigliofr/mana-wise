package usecase_test

import (
	"testing"

	"github.com/gigliofr/mana-wise/domain"
	"github.com/gigliofr/mana-wise/usecase"
)

// TestImpactScoreCalculation verifies the Impact Score normalization
func TestImpactScoreCalculation(t *testing.T) {
	tests := []struct {
		name      string
		cards     []domain.CardImpact
		weights   domain.ImpactWeights
		// Expectations: at least the min/max should be in the expected output
		expectedMin float64
		expectedMax float64
	}{
		{
			name: "Single card gets score 5 (average when alone)",
			cards: []domain.CardImpact{
				{CardID: "1", CardName: "Sol Ring", PriceUSD: 10.0, EdhrecRank: 1},
			},
			weights: domain.DefaultImpactWeights(),
			expectedMin: 4.5,
			expectedMax: 5.5,
		},
		{
			name: "Price variation creates spread",
			cards: []domain.CardImpact{
				{CardID: "1", CardName: "Sol Ring", PriceUSD: 50.0, EdhrecRank: 1},
				{CardID: "2", CardName: "Island", PriceUSD: 0.10, EdhrecRank: 30000},
			},
			weights: domain.DefaultImpactWeights(),
			expectedMin: 0.0,
			expectedMax: 10.0,
		},
		{
			name: "EDHREC rank variation creates spread",
			cards: []domain.CardImpact{
				{CardID: "1", CardName: "Popular Card", PriceUSD: 5.0, EdhrecRank: 1},
				{CardID: "2", CardName: "Niche Card", PriceUSD: 5.0, EdhrecRank: 100000},
			},
			weights: domain.DefaultImpactWeights(),
			expectedMin: 0.0,
			expectedMax: 10.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &usecase.ImpactScoreUseCase{Weights: tt.weights}
			result := uc.Calculate(tt.cards)

			if len(result) != len(tt.cards) {
				t.Errorf("expected %d cards returned, got %d", len(tt.cards), len(result))
			}

			for i, card := range result {
				if card.ImpactScore < tt.expectedMin || card.ImpactScore > tt.expectedMax {
					t.Errorf(
						"card %d: impact score %.2f out of expected range [%.2f, %.2f]",
						i, card.ImpactScore, tt.expectedMin, tt.expectedMax,
					)
				}
			}
		})
	}
}

// TestImpactScoreEdgeCases tests edge cases
func TestImpactScoreEdgeCases(t *testing.T) {
	tests := []struct {
		name      string
		cards     []domain.CardImpact
		weights   domain.ImpactWeights
		shouldErr bool
	}{
		{
			name:      "Empty list",
			cards:     []domain.CardImpact{},
			weights:   domain.DefaultImpactWeights(),
			shouldErr: false,
		},
		{
			name: "All cards with same price and rank",
			cards: []domain.CardImpact{
				{CardID: "1", CardName: "Card A", PriceUSD: 5.0, EdhrecRank: 100},
				{CardID: "2", CardName: "Card B", PriceUSD: 5.0, EdhrecRank: 100},
				{CardID: "3", CardName: "Card C", PriceUSD: 5.0, EdhrecRank: 100},
			},
			weights:   domain.DefaultImpactWeights(),
			shouldErr: false,
		},
		{
			name: "Zero price handling",
			cards: []domain.CardImpact{
				{CardID: "1", CardName: "Card", PriceUSD: 0.0, EdhrecRank: 100},
				{CardID: "2", CardName: "Expensive", PriceUSD: 20.0, EdhrecRank: 100},
			},
			weights:   domain.DefaultImpactWeights(),
			shouldErr: false,
		},
		{
			name: "High rank number handling",
			cards: []domain.CardImpact{
				{CardID: "1", CardName: "Popular", PriceUSD: 5.0, EdhrecRank: 1},
				{CardID: "2", CardName: "Unknown", PriceUSD: 5.0, EdhrecRank: 999999},
			},
			weights:   domain.DefaultImpactWeights(),
			shouldErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			uc := &usecase.ImpactScoreUseCase{Weights: tt.weights}
			result := uc.Calculate(tt.cards)

			if len(result) != len(tt.cards) {
				t.Errorf("expected %d cards returned, got %d", len(tt.cards), len(result))
			}

			// Verify all scores are in valid range [0, 10]
			for i, card := range result {
				if card.ImpactScore < 0 || card.ImpactScore > 10 {
					t.Errorf(
						"card %d: impact score %.2f out of valid range [0.0, 10.0]",
						i, card.ImpactScore,
					)
				}
			}
		})
	}
}

// TestDefaultImpactWeights verifies weights sum to approximately 1.0
func TestDefaultImpactWeights(t *testing.T) {
	weights := domain.DefaultImpactWeights()
	sum := weights.Price + weights.EdhrecRank + weights.Reprint
	const tolerance = 0.01
	if sum < 1.0-tolerance || sum > 1.0+tolerance {
		t.Errorf("Impact weights don't sum to 1.0: %.2f", sum)
	}
}
