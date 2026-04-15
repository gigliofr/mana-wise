package handlers

import (
	"math"
	"testing"
	"time"

	"github.com/gigliofr/mana-wise/domain"
)

func TestComputePriceTrend_StrictLookbackWindows(t *testing.T) {
	now := time.Date(2026, 3, 14, 12, 0, 0, 0, time.UTC)
	card := &domain.Card{
		ID:   "c1",
		Name: "Test Card",
		PriceHistory: []domain.PriceSnapshot{
			{Date: now.AddDate(0, 0, -95), USD: 8},  // older than 90d target fallback
			{Date: now.AddDate(0, 0, -31), USD: 10}, // 30d snapshot source
			{Date: now.AddDate(0, 0, -8), USD: 12},  // 7d snapshot source
			{Date: now, USD: 15},                    // latest
		},
	}

	got := computePriceTrend(card)
	if got.Change7d == nil || got.Change30d == nil || got.Change90d == nil {
		t.Fatalf("expected all windows to be computed, got %+v", got)
	}

	if !almostEqual(*got.Change7d, 25.0) {
		t.Fatalf("expected 7d=25.0, got %.2f", *got.Change7d)
	}
	if !almostEqual(*got.Change30d, 50.0) {
		t.Fatalf("expected 30d=50.0, got %.2f", *got.Change30d)
	}
	if !almostEqual(*got.Change90d, 87.5) {
		t.Fatalf("expected 90d=87.5, got %.2f", *got.Change90d)
	}
}

func TestComputePriceTrend_NoPastSnapshotYieldsNil(t *testing.T) {
	now := time.Date(2026, 3, 14, 12, 0, 0, 0, time.UTC)
	card := &domain.Card{
		ID:   "c2",
		Name: "Recent Only",
		PriceHistory: []domain.PriceSnapshot{
			{Date: now, USD: 10},
			{Date: now.AddDate(0, 0, -2), USD: 9},
		},
	}

	got := computePriceTrend(card)
	if got.Change7d != nil || got.Change30d != nil || got.Change90d != nil {
		t.Fatalf("expected nil windows with no snapshots old enough, got %+v", got)
	}
}

func TestComputePriceTrend_SpikeAlert(t *testing.T) {
	now := time.Date(2026, 3, 14, 12, 0, 0, 0, time.UTC)
	card := &domain.Card{
		ID:   "c3",
		Name: "Spike Card",
		PriceHistory: []domain.PriceSnapshot{
			{Date: now.AddDate(0, 0, -7), USD: 10},
			{Date: now, USD: 13},
		},
	}

	got := computePriceTrend(card)
	if got.Change7d == nil || !got.SpikeAlert {
		t.Fatalf("expected spike alert for >20%% increase, got %+v", got)
	}
}

func TestCosineSimilarity(t *testing.T) {
	a := []float64{1, 0, 0}
	b := []float64{1, 0, 0}
	c := []float64{0, 1, 0}
	if !almostEqual(cosineSimilarity(a, b), 1.0) {
		t.Fatalf("expected cosine=1 for identical vectors")
	}
	if !almostEqual(cosineSimilarity(a, c), 0.0) {
		t.Fatalf("expected cosine=0 for orthogonal vectors")
	}
}

func TestTopK_SortedDesc(t *testing.T) {
	tk := newTopK(3)
	tk.Push(ranked{Score: 0.1})
	tk.Push(ranked{Score: 0.9})
	tk.Push(ranked{Score: 0.5})
	tk.Push(ranked{Score: 0.8})
	tk.Push(ranked{Score: 0.2})
	out := tk.SortedDesc()

	if len(out) != 3 {
		t.Fatalf("expected top 3, got %d", len(out))
	}
	if !(out[0].Score >= out[1].Score && out[1].Score >= out[2].Score) {
		t.Fatalf("expected descending order, got %+v", out)
	}
	if !almostEqual(out[0].Score, 0.9) || !almostEqual(out[1].Score, 0.8) || !almostEqual(out[2].Score, 0.5) {
		t.Fatalf("unexpected top values: %+v", out)
	}
}

func almostEqual(a, b float64) bool {
	return math.Abs(a-b) < 0.0001
}
