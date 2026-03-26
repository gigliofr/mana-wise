package domain

// ScoreDetail contiene un'analisi dettagliata del mazzo basata su impatto e curva.
type ScoreDetail struct {
	Score        float64   `json:"score" bson:"score"`                 // 0-10
	TotalImpact  float64   `json:"total_impact" bson:"total_impact"`
	TippingPoint int       `json:"tipping_point" bson:"tipping_point"`
	ImpactByCMC  map[int]float64 `json:"impact_by_cmc" bson:"impact_by_cmc"`

	// Mana distribution probabilities
	ManaScrew   float64 `json:"mana_screw_pct" bson:"mana_screw_pct"`
	ManaFlood   float64 `json:"mana_flood_pct" bson:"mana_flood_pct"`
	SweetSpot   float64 `json:"sweet_spot_pct" bson:"sweet_spot_pct"`

	// Card-level impact scores
	CardImpacts []CardImpact `json:"card_impacts" bson:"card_impacts"`
}
