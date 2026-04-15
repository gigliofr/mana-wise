package usecase

import (
	"context"

	"github.com/gigliofr/mana-wise/domain"
)

// countLandCards conta le terre nel deck dato quantità per carta.
func countLandCards(cards []*domain.Card, quantities map[string]int) int {
	count := 0
	for _, card := range cards {
		if card.IsLand() {
			count += quantities[card.ID]
		}
	}
	return count
}

// countTotalCards conta il totale di carte nel deck dato le quantità.
func countTotalCards(quantities map[string]int) int {
	total := 0
	for _, qty := range quantities {
		total += qty
	}
	return total
}

// ScoreResult contiene tutti i risultati dell'analisi.
type ScoreResult struct {
	Score        float64             `json:"score"`
	TotalImpact  float64             `json:"total_impact"`
	TippingPoint int                 `json:"tipping_point"`
	ImpactByCMC  map[int]float64     `json:"impact_by_cmc"`
	ManaAnalysis ManaAnalysisResult  `json:"mana_analysis"`
	CardImpacts  []domain.CardImpact `json:"card_impacts"`
}

// ScoreUseCase orchestra il calcolo completo delle metriche.
type ScoreUseCase struct {
	ImpactUC     *ImpactScoreUseCase
	PowerLevelUC *PowerLevelUseCase
}

// NewScoreUseCase crea una nuova istanza.
func NewScoreUseCase(impactUC *ImpactScoreUseCase, powerLevelUC *PowerLevelUseCase) *ScoreUseCase {
	return &ScoreUseCase{
		ImpactUC:     impactUC,
		PowerLevelUC: powerLevelUC,
	}
}

// Execute calcola il score aggregato, tipping point e mana analysis.
func (uc *ScoreUseCase) Execute(ctx context.Context, cards []*domain.Card, quantities map[string]int) (*ScoreResult, error) {
	// 1 Build impacts from cards
	impacts := make([]domain.CardImpact, len(cards))
	for i, card := range cards {
		price := 0.0
		if card.LatestPrice() != nil {
			price = card.LatestPrice().USD
		}
		impacts[i] = domain.CardImpact{
			CardID:     card.ID,
			CardName:   card.Name,
			PriceUSD:   price,
			EdhrecRank: card.EdhrecRank,
		}
	}
	impacts = uc.ImpactUC.Calculate(impacts)

	// 2. Power Level
	plResult := uc.PowerLevelUC.Calculate(cards, quantities, impacts)

	// 3. Tipping Point
	tpResult := CalculateTippingPoint(cards, quantities, impacts)

	// 4. Mana Analysis
	landCount := countLandCards(cards, quantities)
	totalCards := countTotalCards(quantities)

	manaInput := ManaAnalysisInput{
		LandCount:      landCount,
		DeckSize:       totalCards,
		HandSize:       7,
		TargetTurn:     tpResult.TippingPoint,
		MinLandsTarget: max(1, tpResult.TippingPoint-1),
		MaxLandsTarget: tpResult.TippingPoint + 1,
	}
	if manaInput.TargetTurn <= 0 {
		manaInput.TargetTurn = 4
	}
	manaResult := AnalyzeMana(manaInput)

	return &ScoreResult{
		Score:        plResult.Score,
		TotalImpact:  plResult.TotalImpact,
		TippingPoint: tpResult.TippingPoint,
		ImpactByCMC:  tpResult.ImpactByCMC,
		ManaAnalysis: manaResult,
		CardImpacts:  impacts,
	}, nil
}
