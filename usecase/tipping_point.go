package usecase

import "github.com/gigliofr/mana-wise/domain"

// TippingPointResult contiene l'analisi del punto di svolta della curva.
type TippingPointResult struct {
	TippingPoint int            `json:"tipping_point"`
	ImpactByCMC  map[int]float64 `json:"impact_by_cmc"`
}

// CalculateTippingPoint calcola il CMC con il picco di impatto totale.
func CalculateTippingPoint(cards []*domain.Card, quantities map[string]int, impacts []domain.CardImpact) TippingPointResult {
	impactMap := make(map[string]float64)
	for _, c := range impacts {
		impactMap[c.CardID] = c.ImpactScore
	}

	byCMC := make(map[int]float64)
	for _, card := range cards {
		if card.IsBasicLand() {
			continue
		}

		cmc := int(card.CMC)
		qty := quantities[card.ID]
		byCMC[cmc] += impactMap[card.ID] * float64(qty)
	}

	// Trova il CMC con il maximum impact (in caso di parità, minore vince)
	tippingPoint := 0
	maxImpact := -1.0
	for cmc := 0; cmc <= 20; cmc++ {
		if impact, ok := byCMC[cmc]; ok {
			if impact > maxImpact || (impact == maxImpact && cmc < tippingPoint) {
				maxImpact = impact
				tippingPoint = cmc
			}
		}
	}

	return TippingPointResult{
		TippingPoint: tippingPoint,
		ImpactByCMC:  byCMC,
	}
}
