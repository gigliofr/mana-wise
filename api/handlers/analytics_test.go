package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/manawise/api/api/middleware"
	"github.com/manawise/api/domain"
)

type mockTracker struct {
	distinctID string
	event      string
	properties map[string]interface{}
	calls      int
}

func (m *mockTracker) Track(ctx context.Context, distinctID, event string, properties map[string]interface{}) error {
	m.distinctID = distinctID
	m.event = event
	m.properties = properties
	m.calls++
	return nil
}

func TestAnalyticsHandler_UpgradeClick_WithSource(t *testing.T) {
	tracker := &mockTracker{}
	h := NewAnalyticsHandler(tracker)

	body := map[string]string{"source": "analyzer_banner"}
	b, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/analytics/upgrade-click", bytes.NewReader(b))
	ctx := context.WithValue(req.Context(), middleware.ContextKeyUserID, "u-123")
	ctx = context.WithValue(ctx, middleware.ContextKeyPlan, "free")
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	h.UpgradeClick(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	if tracker.calls != 1 {
		t.Fatalf("expected 1 tracking call, got %d", tracker.calls)
	}
	if tracker.distinctID != "u-123" {
		t.Fatalf("unexpected distinct id: %s", tracker.distinctID)
	}
	if tracker.event != "upgrade_clicked" {
		t.Fatalf("unexpected event: %s", tracker.event)
	}
	if got, _ := tracker.properties["source"].(string); got != "analyzer_banner" {
		t.Fatalf("unexpected source: %v", tracker.properties["source"])
	}
	if got, _ := tracker.properties["plan"].(string); got != "free" {
		t.Fatalf("unexpected plan: %v", tracker.properties["plan"])
	}
}

func TestAnalyticsHandler_UpgradeClick_DefaultSource(t *testing.T) {
	tracker := &mockTracker{}
	h := NewAnalyticsHandler(tracker)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/analytics/upgrade-click", bytes.NewReader([]byte(`{}`)))
	ctx := context.WithValue(req.Context(), middleware.ContextKeyUserID, "u-999")
	ctx = context.WithValue(ctx, middleware.ContextKeyPlan, string(domain.PlanPro))
	req = req.WithContext(ctx)

	rr := httptest.NewRecorder()
	h.UpgradeClick(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", rr.Code)
	}
	if got, _ := tracker.properties["source"].(string); got != "unknown" {
		t.Fatalf("expected default source unknown, got %v", tracker.properties["source"])
	}
}
