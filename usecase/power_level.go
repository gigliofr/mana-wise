package usecase

import "github.com/manawise/api/domain"

// PowerLevelResult contiene il punteggio aggregato del mazzo.
type PowerLevelResult struct {
	Score       float64             `json:"score"`
	TotalImpact float64             `json:"total_impact"`
	CardImpacts []domain.CardImpact `json:"card_impacts"`
}

// PowerLevelUseCase calcola il punteggio aggregato.
type PowerLevelUseCase struct {
	ImpactUC *ImpactScoreUseCase
}

// NewPowerLevelUseCase crea una nuova istanza.
func NewPowerLevelUseCase(impactUC *ImpactScoreUseCase) *PowerLevelUseCase {
	return &PowerLevelUseCase{ImpactUC: impactUC}
}

// Calculate calcola il score aggregato per un mazzo.
func (uc *PowerLevelUseCase) Calculate(cards []*domain.Card, quantities map[string]int, impacts []domain.CardImpact) PowerLevelResult {
	var totalImpact float64
	nonlandCount := 0

	impactMap := make(map[string]float64)
	for _, c := range impacts {
		impactMap[c.CardID] = c.ImpactScore
	}

	// Itera sulle carte
	for _, card := range cards {
		if card.IsBasicLand() {
			continue
		}

		score := impactMap[card.ID]
		qty := quantities[card.ID]

		totalImpact += score * float64(qty)
		nonlandCount += qty
	}

	avgScore := 0.0
	if nonlandCount > 0 {
		avgScore = round2(totalImpact / float64(nonlandCount))
	}

	// Clamp 0-10
	if avgScore > 10 {
		avgScore = 10
	}
	if avgScore < 0 {
		avgScore = 0
	}

	return PowerLevelResult{
		Score:       avgScore,
		TotalImpact: round2(totalImpact),
		CardImpacts: impacts,
	}
}
