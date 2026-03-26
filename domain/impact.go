package domain

// CardImpact rappresenta il punteggio di impatto di una singola carta nel formato Commander.
type CardImpact struct {
	CardID      string  `json:"card_id" bson:"card_id"`
	CardName    string  `json:"card_name" bson:"card_name"`
	PriceUSD    float64 `json:"price_usd" bson:"price_usd"`
	EdhrecRank  int     `json:"edhrec_rank" bson:"edhrec_rank"`
	ImpactScore float64 `json:"impact_score" bson:"impact_score"` // 0.0-10.0
}

// ImpactWeights definisce i pesi per il calcolo dell'Impact Score.
type ImpactWeights struct {
	Price      float64 `json:"price"`
	EdhrecRank float64 `json:"edhrec_rank"`
	Reprint    float64 `json:"reprint"`
}

// DefaultImpactWeights restituisce i pesi di default.
// I pesi sono calibrati empiricamente e sommano a 1.0 per una distribuzione normalizzata.
func DefaultImpactWeights() ImpactWeights {
	return ImpactWeights{
		Price:      0.4,
		EdhrecRank: 0.5,
		Reprint:    0.1,
	}
}
