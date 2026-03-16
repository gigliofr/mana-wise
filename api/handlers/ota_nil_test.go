package handlers

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestOTAHandler_NotConfigured(t *testing.T) {
	h := NewOTAHandler(nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/ota/manifest", nil)
	rr := httptest.NewRecorder()
	h.Manifest(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503 when ota handler not configured, got %d", rr.Code)
	}
}
