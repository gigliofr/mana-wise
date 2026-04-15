package domain

import "context"

// AnalyticsTracker records product events (freemium funnel, usage, upgrades).
type AnalyticsTracker interface {
	Track(ctx context.Context, distinctID, event string, properties map[string]interface{}) error
}

// AnalyticsMetricsSnapshot is a compact in-memory view of product funnel metrics.
type AnalyticsMetricsSnapshot struct {
	TotalEvents          int64            `json:"total_events"`
	EventCounts          map[string]int64 `json:"event_counts"`
	AnalysisFallbacks    int64            `json:"analysis_fallbacks"`
	AnalysisByAISource   map[string]int64 `json:"analysis_by_ai_source"`
	ForwardingErrors     int64            `json:"forwarding_errors"`
	LastEventAtUnixMilli int64            `json:"last_event_at_unix_ms"`
}

// AnalyticsMetricsProvider exposes runtime metrics snapshots for admin/ops endpoints.
type AnalyticsMetricsProvider interface {
	Snapshot() AnalyticsMetricsSnapshot
}

// NoopAnalyticsTracker is a disabled tracker implementation.
type NoopAnalyticsTracker struct{}

// Track implements AnalyticsTracker and intentionally does nothing.
func (NoopAnalyticsTracker) Track(ctx context.Context, distinctID, event string, properties map[string]interface{}) error {
	return nil
}
