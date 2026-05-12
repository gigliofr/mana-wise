package usecase

import (
    "testing"
    "time"

    "github.com/gigliofr/mana-wise/domain"
    "github.com/gigliofr/mana-wise/infrastructure/cache"
)

func TestLegalityEvaluator_CacheHitAndClone(t *testing.T) {
    c := cache.New()
    eval := NewLegalityEvaluator(c, 10*time.Minute)

    cards := []*domain.Card{{ID: "c1", Name: "C1", OracleText: "", Legalities: map[string]string{"standard": "legal", "commander": "legal"}}}
    quantities := map[string]int{"c1": 4}

    first := eval.AllFormats(cards, quantities)
    if first == nil {
        t.Fatal("expected non-nil result")
    }

    // Mutate the returned map and slices to ensure clones are returned.
    if rf, ok := first["standard"]; ok {
        rf.Issues = append(rf.Issues, "mutated")
        if len(rf.Issues) == 0 {
            t.Fatal("sanity: mutation failed")
        }
        first["standard"] = rf
    }

    // Second retrieval should be unaffected by mutations above.
    second := eval.AllFormats(cards, quantities)
    if second == nil {
        t.Fatal("expected non-nil second result")
    }
    if secStd, ok := second["standard"]; ok {
        for _, issue := range secStd.Issues {
            if issue == "mutated" {
                t.Fatalf("cache returned a reference instead of a clone; found mutation in cached value")
            }
        }
    }
}
