package domain

import "context"

// AnalyticsTracker records product events (freemium funnel, usage, upgrades).
type AnalyticsTracker interface {
	Track(ctx context.Context, distinctID, event string, properties map[string]interface{}) error
}

// NoopAnalyticsTracker is a disabled tracker implementation.
type NoopAnalyticsTracker struct{}

// Track implements AnalyticsTracker and intentionally does nothing.
func (NoopAnalyticsTracker) Track(ctx context.Context, distinctID, event string, properties map[string]interface{}) error {
	return nil
}
