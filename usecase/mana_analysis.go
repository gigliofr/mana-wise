package usecase

import (
	"math/big"
)

// ManaAnalysisInput contains parameters for mana distribution analysis
type ManaAnalysisInput struct {
	LandCount      int // Total lands in deck (including MDFC-lands)
	DeckSize       int // Total cards in deck (default: 100 for EDH)
	HandSize       int // Initial hand size (default: 7)
	TargetTurn     int // Target turn for analysis (based on tipping point)
	MinLandsTarget int // Minimum lands desired by TargetTurn
	MaxLandsTarget int // Maximum lands acceptable by TargetTurn
}

// ManaAnalysisResult contains probabilities for mana scenarios
type ManaAnalysisResult struct {
	ManaScrew    float64 `json:"mana_screw_pct"`   // Probability of mana screw (%)
	ManaFlood    float64 `json:"mana_flood_pct"`   // Probability of mana flood (%)
	SweetSpot    float64 `json:"sweet_spot_pct"`   // Probability of optimal mana (%)
	LandCount    int     `json:"land_count"`
	NonLandCount int     `json:"non_land_count"`
}

// AnalyzeMana calculates the probability of Mana Screw, Flood, and Sweet Spot
// using the hypergeometric distribution.
//
// The number of cards drawn is: HandSize + (TargetTurn - 1)
// This represents the initial hand plus cards drawn until reaching target turn.
//
// Mana Screw: P(lands < MinLandsTarget)
// Mana Flood: P(lands > MaxLandsTarget)
// Sweet Spot: P(MinLandsTarget <= lands <= MaxLandsTarget)
func AnalyzeMana(input ManaAnalysisInput) ManaAnalysisResult {
	N := input.DeckSize         // Total deck size
	K := input.LandCount        // Total lands in deck
	n := input.HandSize + input.TargetTurn - 1 // Cards drawn by target turn
	if n > N {
		n = N // Clamp to deck size
	}

	// Calculate probabilities using hypergeometric distribution
	var screw, flood float64

	// Mana Screw: P(X < min_lands_target)
	for k := 0; k < input.MinLandsTarget; k++ {
		screw += Hypergeometric(N, K, n, k)
	}

	// Mana Flood: P(X > max_lands_target)
	for k := input.MaxLandsTarget + 1; k <= n && k <= K; k++ {
		flood += Hypergeometric(N, K, n, k)
	}

	// Sweet Spot: 1 - Screw - Flood
	sweet := 1.0 - screw - flood
	if sweet < 0 {
		sweet = 0 // Handle floating-point rounding errors
	}

	return ManaAnalysisResult{
		ManaScrew:    round2(screw * 100),
		ManaFlood:    round2(flood * 100),
		SweetSpot:    round2(sweet * 100),
		LandCount:    K,
		NonLandCount: N - K,
	}
}

// Hypergeometric calculates the probability mass function of the hypergeometric distribution.
//
// Parameters:
// N: total population size (deck size)
// K: number of success states in population (lands)
// n: number of draws (cards drawn)
// k: number of observed successes (lands drawn)
//
// Formula: P(X = k) = C(K, k) × C(N-K, n-k) / C(N, n)
// where C(n, k) is the binomial coefficient "n choose k"
func Hypergeometric(N, K, n, k int) float64 {
	// Boundary checks
	if k < 0 || k > K || k > n {
		return 0
	}
	if n-k > N-K {
		return 0
	}
	if n > N {
		return 0
	}

	// Use big.Int for numerator to avoid overflow
	num := new(big.Int).Mul(binom(K, k), binom(N-K, n-k))
	den := binom(N, n)

	if den.Sign() == 0 {
		return 0
	}

	// Convert to float64
	numFloat := new(big.Float).SetInt(num)
	denFloat := new(big.Float).SetInt(den)
	result := new(big.Float).Quo(numFloat, denFloat)

	f, _ := result.Float64()
	return f
}

// binom calculates the binomial coefficient C(n, k) = n! / (k! * (n-k)!)
// Uses big.Int to prevent overflow on large factorials
func binom(n, k int) *big.Int {
	if k < 0 || k > n {
		return big.NewInt(0)
	}
	if k == 0 || k == n {
		return big.NewInt(1)
	}
	// Optimize by using the smaller of k or n-k
	if k > n-k {
		k = n - k
	}

	// Go 1.21+ has math/big.Int.Binomial()
	result := new(big.Int).Binomial(int64(n), int64(k))
	return result
}
