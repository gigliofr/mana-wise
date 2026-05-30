package usecase

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/gigliofr/mana-wise/domain"
	"github.com/gigliofr/mana-wise/infrastructure/cache"
)

// LegalityEvaluator caches legality reports for repeated deck snapshots.
type LegalityEvaluator struct {
	cache *cache.Cache
	ttl   time.Duration
}

// NewLegalityEvaluator creates a cache-backed legality evaluator.
func NewLegalityEvaluator(c *cache.Cache, ttl time.Duration) *LegalityEvaluator {
	return &LegalityEvaluator{cache: c, ttl: ttl}
}

// AllFormats returns legality for every supported format, using cache when available.
func (e *LegalityEvaluator) AllFormats(cards []*domain.Card, quantities map[string]int) map[string]DeckLegalityResult {
	if e == nil {
		return DetermineDeckLegalityAllFormats(cards, quantities)
	}
	if cached, ok := e.get(allFormatsKey(cards, quantities)); ok {
		return cached
	}
	result := DetermineDeckLegalityAllFormats(cards, quantities)
	e.set(allFormatsKey(cards, quantities), result)
	return cloneLegalityMap(result)
}

func (e *LegalityEvaluator) get(key string) (map[string]DeckLegalityResult, bool) {
	if e == nil || e.cache == nil {
		return nil, false
	}
	value, ok := e.cache.Get(key)
	if !ok {
		return nil, false
	}
	cached, ok := value.(map[string]DeckLegalityResult)
	if !ok {
		return nil, false
	}
	return cloneLegalityMap(cached), true
}

func (e *LegalityEvaluator) set(key string, value map[string]DeckLegalityResult) {
	if e == nil || e.cache == nil || e.ttl <= 0 {
		return
	}
	e.cache.Set(key, cloneLegalityMap(value), e.ttl)
}

func allFormatsKey(cards []*domain.Card, quantities map[string]int) string {
	h := sha256.New()
	entries := uniqueCardsByID(cards)
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].ID < entries[j].ID
	})
	for _, card := range entries {
		fmt.Fprintf(h, "%s|%s|%s|", card.ID, card.Name, strings.TrimSpace(card.OracleText))
	}
	keys := make([]string, 0, len(quantities))
	for id := range quantities {
		keys = append(keys, id)
	}
	sort.Strings(keys)
	for _, id := range keys {
		fmt.Fprintf(h, "%s=%d|", id, quantities[id])
	}
	return fmt.Sprintf("legality:%x", h.Sum(nil))
}

func cloneLegalityMap(src map[string]DeckLegalityResult) map[string]DeckLegalityResult {
	if src == nil {
		return nil
	}
	out := make(map[string]DeckLegalityResult, len(src))
	for k, v := range src {
		v.Issues = append([]string(nil), v.Issues...)
		if len(v.IllegalCards) > 0 {
			v.IllegalCards = append([]IllegalCardIssue(nil), v.IllegalCards...)
		}
		out[k] = v
	}
	return out
}
