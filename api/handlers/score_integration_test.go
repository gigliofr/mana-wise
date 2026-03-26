package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/manawise/api/domain"
	"github.com/manawise/api/usecase"
)

// TestScoreEndpointIntegration validates the POST /score endpoint returns properly structured ScoreDetail.
func TestScoreEndpointIntegration(t *testing.T) {
	// Create test cards with realistic data
	cards := []*domain.Card{
		{
			ID:         "test-card-1",
			Name:       "Test Land",
			CMC:        0,
			TypeLine:   "Basic Land",
			EdhrecRank: 100,
			PriceHistory: []domain.PriceSnapshot{
				{
					Date: time.Now(),
					USD:  0.50,
				},
			},
		},
		{
			ID:         "test-card-2",
			Name:       "Test Spell",
			CMC:        3.0,
			TypeLine:   "Creature — Test",
			OracleText: "Test ability",
			EdhrecRank: 500,
			PriceHistory: []domain.PriceSnapshot{
				{
					Date: time.Now(),
					USD:  5.00,
				},
			},
		},
	}

	// Build request
	reqBody := map[string]interface{}{
		"decklist": cards,
		"format":   "commander",
	}

	reqBytes, _ := json.Marshal(reqBody)
	req := httptest.NewRequest("POST", "/score", bytes.NewReader(reqBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	// Set up use cases
	impactUC := &usecase.ImpactScoreUseCase{
		Weights: domain.DefaultImpactWeights(),
	}
	powerLevelUC := &usecase.PowerLevelUseCase{}
	scoreUC := usecase.NewScoreUseCase(impactUC, powerLevelUC)

	// Create handler
	handler := newScoreHandler(scoreUC)

	// Execute
	handler.ServeHTTP(w, req)

	// Assertions
	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	// Validate response structure
	scoreDetail, ok := response["score_detail"].(map[string]interface{})
	if !ok {
		t.Fatal("Missing or invalid score_detail in response")
	}

	// Check required fields
	requiredFields := []string{"score", "total_impact", "tipping_point", "impact_by_cmc", "mana_screw_pct", "mana_flood_pct", "sweet_spot_pct", "card_impacts"}
	for _, field := range requiredFields {
		if _, exists := scoreDetail[field]; !exists {
			t.Errorf("Missing required field: %s", field)
		}
	}

	// Validate score is in range [0, 10]
	score, ok := scoreDetail["score"].(float64)
	if !ok {
		t.Fatal("score is not a float64")
	}
	if score < 0 || score > 10 {
		t.Errorf("Score out of range [0,10]: %f", score)
	}

	// Validate mana probabilities are in range [0, 100]
	manaScrew, ok := scoreDetail["mana_screw_pct"].(float64)
	if !ok {
		t.Fatal("mana_screw_pct is not a float64")
	}
	if manaScrew < 0 || manaScrew > 100 {
		t.Errorf("Mana Screw out of range [0,100]: %f", manaScrew)
	}

	manaFlood, ok := scoreDetail["mana_flood_pct"].(float64)
	if !ok {
		t.Fatal("mana_flood_pct is not a float64")
	}
	if manaFlood < 0 || manaFlood > 100 {
		t.Errorf("Mana Flood out of range [0,100]: %f", manaFlood)
	}

	sweetSpot, ok := scoreDetail["sweet_spot_pct"].(float64)
	if !ok {
		t.Fatal("sweet_spot_pct is not a float64")
	}
	if sweetSpot < 0 || sweetSpot > 100 {
		t.Errorf("Sweet Spot out of range [0,100]: %f", sweetSpot)
	}

	// Log for visual inspection
	t.Logf("✓ Score: %.2f", score)
	t.Logf("✓ ManaScrew: %.2f%%", manaScrew)
	t.Logf("✓ ManaFlood: %.2f%%", manaFlood)
	t.Logf("✓ SweetSpot: %.2f%%", sweetSpot)
	t.Logf("✓ TippingPoint: %v", scoreDetail["tipping_point"])
	t.Logf("✓ Response structure is valid for frontend consumption")
}

// newScoreHandler creates a handler for testing; would normally be injected.
func newScoreHandler(scoreUC *usecase.ScoreUseCase) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			Decklist []*domain.Card       `json:"decklist"`
			Format   string               `json:"format"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request", http.StatusBadRequest)
			return
		}

		// Build quantities map
		quantities := make(map[string]int)
		for _, card := range req.Decklist {
			quantities[card.ID] = 1 // Simplified: assume 1 copy per card for testing
		}

		// Execute score use case
		scoreResult, err := scoreUC.Execute(context.Background(), req.Decklist, quantities)
		if err != nil {
			http.Error(w, "Failed to calculate score", http.StatusInternalServerError)
			return
		}

		// Convert to domain.ScoreDetail for JSON response
		scoreDetail := &domain.ScoreDetail{
			Score:        scoreResult.Score,
			TotalImpact:  scoreResult.TotalImpact,
			TippingPoint: scoreResult.TippingPoint,
			ImpactByCMC:  scoreResult.ImpactByCMC,
			ManaScrew:    scoreResult.ManaAnalysis.ManaScrew,
			ManaFlood:    scoreResult.ManaAnalysis.ManaFlood,
			SweetSpot:    scoreResult.ManaAnalysis.SweetSpot,
			CardImpacts:  scoreResult.CardImpacts,
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"score_detail": scoreDetail,
			"latency_ms":   0,
		})
	}
}
