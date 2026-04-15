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
	}
}
