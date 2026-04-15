package analytics

import (
	"context"
	"errors"
	"testing"
)

type stubTracker struct {
	err error
}

func (s *stubTracker) Track(ctx context.Context, distinctID, event string, properties map[string]interface{}) error {
	return s.err
}

func TestRuntimeMetricsTracker_SnapshotCounts(t *testing.T) {
	tracker := NewRuntimeMetricsTracker(&stubTracker{})
	ctx := context.Background()

	_ = tracker.Track(ctx, "u1", "analysis_completed", map[string]interface{}{
		"ai_source":   "internal_rules",
		"ai_fallback": true,
	})
	_ = tracker.Track(ctx, "u1", "deck_saved", map[string]interface{}{"format": "modern"})

	s := tracker.Snapshot()
	if s.TotalEvents != 2 {
		t.Fatalf("expected total_events=2, got %d", s.TotalEvents)
	}
	if s.EventCounts["analysis_completed"] != 1 {
		t.Fatalf("expected analysis_completed=1, got %d", s.EventCounts["analysis_completed"])
	}
	if s.EventCounts["deck_saved"] != 1 {
		t.Fatalf("expected deck_saved=1, got %d", s.EventCounts["deck_saved"])
	}
	if s.AnalysisFallbacks != 1 {
		t.Fatalf("expected analysis_fallbacks=1, got %d", s.AnalysisFallbacks)
	}
	if s.AnalysisByAISource["internal_rules"] != 1 {
		t.Fatalf("expected analysis_by_ai_source.internal_rules=1, got %d", s.AnalysisByAISource["internal_rules"])
	}
	if s.LastEventAtUnixMilli == 0 {
		t.Fatalf("expected non-zero last_event_at_unix_ms")
	}
}

func TestRuntimeMetricsTracker_ForwardErrorCount(t *testing.T) {
	tracker := NewRuntimeMetricsTracker(&stubTracker{err: errors.New("forward failed")})
	err := tracker.Track(context.Background(), "u1", "upgrade_clicked", nil)
	if err == nil {
		t.Fatalf("expected forwarding error")
	}

	s := tracker.Snapshot()
	if s.TotalEvents != 1 {
		t.Fatalf("expected total_events=1, got %d", s.TotalEvents)
	}
	if s.ForwardingErrors != 1 {
		t.Fatalf("expected forwarding_errors=1, got %d", s.ForwardingErrors)
	}
}
