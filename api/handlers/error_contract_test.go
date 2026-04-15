package handlers

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestJSONError_ContractShape(t *testing.T) {
	rr := httptest.NewRecorder()
	jsonError(rr, "invalid request body", http.StatusBadRequest)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", rr.Code)
	}

	assertErrorEnvelope(t, rr.Body.Bytes(), "invalid request body", "bad_request", http.StatusBadRequest)
}

func TestAnalyzeHandler_InvalidJSON_UsesStandardErrorContract(t *testing.T) {
	h := NewAnalyzeHandler(nil, nil, nil, nil)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/analyze", bytes.NewBufferString("{"))
	rr := httptest.NewRecorder()

	h.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d body=%s", rr.Code, rr.Body.String())
	}
	assertErrorEnvelope(t, rr.Body.Bytes(), "invalid request body", "bad_request", http.StatusBadRequest)
}

func TestOTAHandler_InvalidChecksum_UsesStandardErrorContract(t *testing.T) {
	h, _ := newOTAHandlerForTest()
	payload := `{"version":"1.0.0","platform":"esp32","binary_base64":"Zg==","sha256":"deadbeef"}`
	req := httptest.NewRequest(http.MethodPost, "/api/v1/ota/release", bytes.NewBufferString(payload))
	rr := httptest.NewRecorder()

	h.PublishRelease(rr, req)

	if rr.Code != http.StatusUnprocessableEntity {
		t.Fatalf("expected status 422, got %d body=%s", rr.Code, rr.Body.String())
	}
	assertErrorEnvelope(t, rr.Body.Bytes(), "", "unprocessable_entity", http.StatusUnprocessableEntity)
}

func assertErrorEnvelope(t *testing.T, body []byte, expectedErrorContains, expectedCode string, expectedStatus int) {
	t.Helper()

	var got map[string]interface{}
	if err := json.Unmarshal(body, &got); err != nil {
		t.Fatalf("decode error envelope: %v; body=%s", err, string(body))
	}

	errMsg, ok := got["error"].(string)
	if !ok || errMsg == "" {
		t.Fatalf("expected non-empty string field error, got %v", got["error"])
	}
	if expectedErrorContains != "" && errMsg != expectedErrorContains {
		t.Fatalf("expected error=%q, got %q", expectedErrorContains, errMsg)
	}

	code, ok := got["code"].(string)
	if !ok || code != expectedCode {
		t.Fatalf("expected code=%q, got %v", expectedCode, got["code"])
	}

	status, ok := got["status"].(float64)
	if !ok || int(status) != expectedStatus {
		t.Fatalf("expected status=%d, got %v", expectedStatus, got["status"])
	}
}
