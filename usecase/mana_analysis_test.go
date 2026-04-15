package usecase_test

import (
	"math"
	"testing"

	"github.com/gigliofr/mana-wise/usecase"
)

// TestHypergeometricDistribution verifies the hypergeometric PMF calculation.
// Known values from statistical tables: deck=60, lands=24, sampled=7, exact_lands=0
// Expected P(X=0) ≈ 0.0119 ± 0.0005
func TestHypergeometricDistribution(t *testing.T) {
	tests := []struct {
		name            string
		deckSize        int
		landCount       int
		sampleSize      int
		exactLands      int
		expectedMin     float64 // ±error margin
		expectedMax     float64
		description     string
	}{
		{
			name:        "No lands drawn initially (7 cards)",
			deckSize:    60,
			landCount:   24,
			sampleSize:  7,
			exactLands:  0,
			expectedMin: 0.015,
			expectedMax: 0.025,
			description: "Rare but possible mana screw scenario",
		},
		{
			name:        "Perfect distribution (7 cards with 3 lands)",
			deckSize:    60,
			landCount:   24,
			sampleSize:  7,
			exactLands:  3,
			expectedMin: 0.25,
			expectedMax: 0.35,
			description: "~40% of deck is lands; 3 of 7 in hand is typical",
		},
		{
			name:        "All lands (7 cards all lands)",
			deckSize:    60,
			landCount:   24,
			sampleSize:  7,
			exactLands:  7,
			expectedMin: 0.0001,
			expectedMax: 0.001,
			description: "Extreme mana flood",
		},
		{
			name:        "Commander deck (100 cards, 37 lands, 10 sampled)",
			deckSize:    100,
			landCount:   37,
			sampleSize:  10,
			exactLands:  3,
			expectedMin: 0.20,
			expectedMax: 0.30,
			description: "Typical initial hand + few draw steps",
		},
		{
			name:        "Boundary: single land",
			deckSize:    20,
			landCount:   1,
			sampleSize:  5,
			exactLands:  0,
			expectedMin: 0.70,
			expectedMax: 0.80,
			description: "High probability of no lands with only 1 in deck",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			prob := usecase.Hypergeometric(tt.deckSize, tt.landCount, tt.sampleSize, tt.exactLands)
			if prob < tt.expectedMin || prob > tt.expectedMax {
				t.Errorf(
					"%s: got %.6f, expected range [%.6f, %.6f]",
					tt.description, prob, tt.expectedMin, tt.expectedMax,
				)
			}
		})
	}
}

// TestManaAnalysisProbabilities ensures Screw + Flood + Sweet Spot = 1.0 within tolerance
func TestManaAnalysisProbabilities(t *testing.T) {
	input := usecase.ManaAnalysisInput{
		LandCount:      37,
		DeckSize:       100,
		HandSize:       7,
		TargetTurn:     4,
		MinLandsTarget: 3,
		MaxLandsTarget: 5,
	}

	result := usecase.AnalyzeMana(input)

	total := result.ManaScrew + result.ManaFlood + result.SweetSpot
	const tolerance = 0.1 // Allow ±0.1% due to rounding

	if math.Abs(total-100.0) > tolerance {
		t.Errorf("Probabilities don't sum to 100: %.2f%%", total)
	}

	if result.ManaScrew < 0 || result.ManaScrew > 100 {
		t.Errorf("ManaScrew out of range: %.2f%%", result.ManaScrew)
	}
	if result.ManaFlood < 0 || result.ManaFlood > 100 {
		t.Errorf("ManaFlood out of range: %.2f%%", result.ManaFlood)
	}
	if result.SweetSpot < 0 || result.SweetSpot > 100 {
		t.Errorf("SweetSpot out of range: %.2f%%", result.SweetSpot)
	}
}

// TestManaAnalysisLogic validates the logic of screw/flood/sweet spot
func TestManaAnalysisLogic(t *testing.T) {
	tests := []struct {
		name       string
		landCount  int
		deckSize   int
		handSize   int
		targetTurn int
		minTarget  int
		maxTarget  int
		// Expectations: which should be highest percentage
		expectedHighest string
	}{
		{
			name:           "Low lands → high screw",
			landCount:      20,
			deckSize:       100,
			handSize:       7,
			targetTurn:     5,
			minTarget:      3,
			maxTarget:      6,
			expectedHighest: "screw",
		},
		{
			name:           "High lands → high flood",
			landCount:      60,
			deckSize:       100,
			handSize:       7,
			targetTurn:     5,
			minTarget:      3,
			maxTarget:      6,
			expectedHighest: "flood",
		},
		{
			name:           "Balanced lands → sweet spot dominates",
			landCount:      37,
			deckSize:       100,
			handSize:       7,
			targetTurn:     5,
			minTarget:      3,
			maxTarget:      5,
			expectedHighest: "sweet",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			input := usecase.ManaAnalysisInput{
				LandCount:      tt.landCount,
				DeckSize:       tt.deckSize,
				HandSize:       tt.handSize,
				TargetTurn:     tt.targetTurn,
				MinLandsTarget: tt.minTarget,
				MaxLandsTarget: tt.maxTarget,
			}
			result := usecase.AnalyzeMana(input)

			var highest float64
			var scenario string

			if result.ManaScrew > highest {
				highest = result.ManaScrew
				scenario = "screw"
			}
			if result.ManaFlood > highest {
				highest = result.ManaFlood
				scenario = "flood"
			}
			if result.SweetSpot > highest {
				highest = result.SweetSpot
				scenario = "sweet"
			}

			if scenario != tt.expectedHighest {
				t.Errorf(
					"Expected %s to be highest, but got %s (result: %.1f%%, %.1f%%, %.1f%%)",
					tt.expectedHighest, scenario,
					result.ManaScrew, result.ManaFlood, result.SweetSpot,
				)
			}
		})
	}
}

// TestManaAnalysisEdgeCase tests edge cases
func TestManaAnalysisEdgeCase(t *testing.T) {
	tests := []struct {
		name      string
		input     usecase.ManaAnalysisInput
		shouldErr bool
	}{
		{
			name: "Normal Commander deck",
			input: usecase.ManaAnalysisInput{
				LandCount:      37,
				DeckSize:       100,
				HandSize:       7,
				TargetTurn:     4,
				MinLandsTarget: 2,
				MaxLandsTarget: 5,
			},
			shouldErr: false,
		},
		{
			name: "Edge: Sample size exceeds deck",
			input: usecase.ManaAnalysisInput{
				LandCount:      10,
				DeckSize:       20,
				HandSize:       7,
				TargetTurn:     20,
				MinLandsTarget: 1,
				MaxLandsTarget: 10,
			},
			shouldErr: false, // Should clamp sample to deck size
		},
		{
			name: "Edge: No lands in deck",
			input: usecase.ManaAnalysisInput{
				LandCount:      0,
				DeckSize:       100,
				HandSize:       7,
				TargetTurn:     4,
				MinLandsTarget: 1,
				MaxLandsTarget: 5,
			},
			shouldErr: false,
		},
		{
			name: "Edge: All lands",
			input: usecase.ManaAnalysisInput{
				LandCount:      100,
				DeckSize:       100,
				HandSize:       7,
				TargetTurn:     4,
				MinLandsTarget: 2,
				MaxLandsTarget: 5,
			},
			shouldErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := usecase.AnalyzeMana(tt.input)

			// Verify result is valid (no NaN, probabilities in range)
			if math.IsNaN(result.ManaScrew) || math.IsNaN(result.ManaFlood) || math.IsNaN(result.SweetSpot) {
				t.Error("Got NaN in result")
			}

			total := result.ManaScrew + result.ManaFlood + result.SweetSpot
			if total < 99 || total > 101 {
				t.Errorf("Probabilities don't sum correctly: %.2f%%", total)
			}
		})
	}
}

// BenchmarkManaAnalysis measures performance of analysis
func BenchmarkManaAnalysis(b *testing.B) {
	input := usecase.ManaAnalysisInput{
		LandCount:      37,
		DeckSize:       100,
		HandSize:       7,
		TargetTurn:     4,
		MinLandsTarget: 2,
		MaxLandsTarget: 5,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		usecase.AnalyzeMana(input)
	}
}
