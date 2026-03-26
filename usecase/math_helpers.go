package usecase

import "math"

// round2 rounds a float64 to 2 decimal places.
// This is a shared utility used by multiple analysis modules.
func round2(v float64) float64 {
	return math.Round(v*100) / 100
}

// max returns the larger of two integers.
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// min returns the smaller of two integers.
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
