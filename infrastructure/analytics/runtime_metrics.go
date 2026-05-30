package analytics

import (
	"context"
	"strings"
	"sync"
	"time"

	"github.com/gigliofr/mana-wise/domain"
)

// RuntimeMetricsTracker records lightweight in-process metrics while forwarding events to a delegate tracker.
type RuntimeMetricsTracker struct {
	next domain.AnalyticsTracker

	mu sync.RWMutex

	totalEvents       int64
	eventCounts       map[string]int64
	analysisFallbacks int64
	analysisBySource  map[string]int64
	forwardingErrors  int64
	lastEventAt       time.Time
	cacheHits         int64
	cacheMisses       int64
	cbOpenCount       int64
	cbHalfOpenCount   int64
	cbCloseCount      int64
}

// NewRuntimeMetricsTracker wraps an existing tracker and adds runtime counters.
func NewRuntimeMetricsTracker(next domain.AnalyticsTracker) *RuntimeMetricsTracker {
	if next == nil {
		next = domain.NoopAnalyticsTracker{}
	}
	return &RuntimeMetricsTracker{
		next:             next,
		eventCounts:      map[string]int64{},
		analysisBySource: map[string]int64{},
	}
}

// Track records metrics and forwards the event to the wrapped tracker.
func (t *RuntimeMetricsTracker) Track(ctx context.Context, distinctID, event string, properties map[string]interface{}) error {
	normalizedEvent := strings.TrimSpace(event)
	if normalizedEvent == "" {
		normalizedEvent = "unknown"
	}

	t.mu.Lock()
	t.totalEvents++
	t.eventCounts[normalizedEvent]++
	t.lastEventAt = time.Now().UTC()

	if normalizedEvent == "analysis_completed" {
		source := "unknown"
		if properties != nil {
			if rawSource, ok := properties["ai_source"]; ok {
				if s, ok := rawSource.(string); ok && strings.TrimSpace(s) != "" {
					source = strings.TrimSpace(s)
				}
			}
			t.analysisBySource[source]++
			if rawFallback, ok := properties["ai_fallback"]; ok {
				if fb, ok := rawFallback.(bool); ok && fb {
					t.analysisFallbacks++
				}
			}
		} else {
			t.analysisBySource[source]++
		}
	}
	t.mu.Unlock()

	if err := t.next.Track(ctx, distinctID, event, properties); err != nil {
		t.mu.Lock()
		t.forwardingErrors++
		t.mu.Unlock()
		return err
	}
	return nil
}

// RecordCacheHit increments the in-process cache hit counter.
func (t *RuntimeMetricsTracker) RecordCacheHit() {
	t.mu.Lock()
	t.cacheHits++
	t.mu.Unlock()
}

// RecordCacheMiss increments the in-process cache miss counter.
func (t *RuntimeMetricsTracker) RecordCacheMiss() {
	t.mu.Lock()
	t.cacheMisses++
	t.mu.Unlock()
}

// RecordCircuitBreakerOpen increments the counter for CB open transitions.
func (t *RuntimeMetricsTracker) RecordCircuitBreakerOpen() {
	t.mu.Lock()
	t.cbOpenCount++
	t.mu.Unlock()
}

// RecordCircuitBreakerHalfOpen increments the counter for CB half-open transitions.
func (t *RuntimeMetricsTracker) RecordCircuitBreakerHalfOpen() {
	t.mu.Lock()
	t.cbHalfOpenCount++
	t.mu.Unlock()
}

// RecordCircuitBreakerClosed increments the counter for CB closed transitions.
func (t *RuntimeMetricsTracker) RecordCircuitBreakerClosed() {
	t.mu.Lock()
	t.cbCloseCount++
	t.mu.Unlock()
}

// Snapshot returns a copy of the current runtime counters.
func (t *RuntimeMetricsTracker) Snapshot() domain.AnalyticsMetricsSnapshot {
	t.mu.RLock()
	defer t.mu.RUnlock()

	eventCounts := make(map[string]int64, len(t.eventCounts))
	for k, v := range t.eventCounts {
		eventCounts[k] = v
	}

	analysisBySource := make(map[string]int64, len(t.analysisBySource))
	for k, v := range t.analysisBySource {
		analysisBySource[k] = v
	}

	lastEventMs := int64(0)
	if !t.lastEventAt.IsZero() {
		lastEventMs = t.lastEventAt.UnixMilli()
	}

	return domain.AnalyticsMetricsSnapshot{
		TotalEvents:          t.totalEvents,
		EventCounts:          eventCounts,
		AnalysisFallbacks:    t.analysisFallbacks,
		AnalysisByAISource:   analysisBySource,
		ForwardingErrors:     t.forwardingErrors,
		LastEventAtUnixMilli: lastEventMs,
		CacheHits:            t.cacheHits,
		CacheMisses:          t.cacheMisses,
		// Circuit breaker counters
		CircuitBreakerOpenCount:     t.cbOpenCount,
		CircuitBreakerHalfOpenCount: t.cbHalfOpenCount,
		CircuitBreakerCloseCount:    t.cbCloseCount,
	}
}
