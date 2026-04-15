package usecase

import (
	"math"

	"github.com/gigliofr/mana-wise/domain"
)

// ImpactScoreUseCase calcola l'Impact Score per una lista di carte.
// L'Impact Score misura la "rilevanza" di una carta nel formato Commander
// in base al prezzo di mercato e alla popolarità (EDHREC rank).
type ImpactScoreUseCase struct {
	Weights domain.ImpactWeights
}

// Calculate calcola l'Impact Score normalizzato (0-10) per ogni carta.
// La normalizzazione viene effettuata con min-max su tutto il set di carte.
//
// Formula:
//   rawScore = w_price * log(price+1) + w_edhrec * edhrecScore(rank)
//   normalizedScore = ((rawScore - min) / (max - min)) * 10
//
// Se tutte le carte hanno lo stesso rawScore, viene assegnato 5.0 a tutte.
func (uc *ImpactScoreUseCase) Calculate(cards []domain.CardImpact) []domain.CardImpact {
	if len(cards) == 0 {
		return cards
	}

	// 1. Calcolo raw score per ogni carta
	rawScores := make([]float64, len(cards))
	for i, c := range cards {
		priceComponent := math.Log1p(c.PriceUSD) // log per smorzare outlier di prezzo
		edhrecComponent := edhrecScore(c.EdhrecRank)
		rawScores[i] = uc.Weights.Price*priceComponent + uc.Weights.EdhrecRank*edhrecComponent
	}

	// 2. Normalizzazione min-max → range 0–10
	min, max := rawScores[0], rawScores[0]
	for _, v := range rawScores {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}

	// 3. Assegnazione dello score normalizzato
	for i := range cards {
		if max == min {
			// Se tutti i raw score sono uguali, assegna il valore medio (5.0)
			cards[i].ImpactScore = 5.0
		} else {
			cards[i].ImpactScore = round2(((rawScores[i] - min) / (max - min)) * 10)
		}
	}

	return cards
}

// edhrecScore converte il rank EDHREC in un punteggio.
// Ranking inferiore (più popolare) → punteggio più alto.
// Formula: 1 / log(rank + 1) per smorzare i ranking di nicchia.
func edhrecScore(rank int) float64 {
	if rank <= 0 {
		return 0
	}
	return 1.0 / math.Log1p(float64(rank))
}
