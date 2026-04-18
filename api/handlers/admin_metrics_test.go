package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gigliofr/mana-wise/domain"
)

type stubMetricsProvider struct {
	snapshot domain.AnalyticsMetricsSnapshot
}

func (s stubMetricsProvider) Snapshot() domain.AnalyticsMetricsSnapshot {
	return s.snapshot
}

func TestAdminHandler_FunnelMetrics_Success(t *testing.T) {
	h := NewAdminHandler(nil, stubMetricsProvider{snapshot: domain.AnalyticsMetricsSnapshot{
		TotalEvents:       7,
		EventCounts:       map[string]int64{"analysis_completed": 3},
		AnalysisFallbacks: 1,
	}}, nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/metrics/funnel", nil)
	rr := httptest.NewRecorder()
	h.FunnelMetrics(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d body=%s", rr.Code, rr.Body.String())
	}
}

func TestAdminHandler_FunnelMetrics_NoProvider(t *testing.T) {
	h := NewAdminHandler(nil, nil, nil)
	req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/metrics/funnel", nil)
	rr := httptest.NewRecorder()
	h.FunnelMetrics(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d body=%s", rr.Code, rr.Body.String())
	}
}
