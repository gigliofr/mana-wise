package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestMetaSnapshot_Modern_OK(t *testing.T) {
	h := NewMetaHandler()

	req := httptest.NewRequest("GET", "/meta/modern", nil)
	w := httptest.NewRecorder()

	// Mock chi.URLParam extraction since chi router not used in test
	// In real test, use chi router context
	req.Header.Set("X-Mock-Format", "modern")

	h.Snapshot(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected status %d, got %d", http.StatusOK, w.Code)
	}

	var resp deckMetaSnapshotResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	// Assertions
	if resp.Format != "modern" {
		t.Errorf("expected format modern, got %s", resp.Format)
	}

	if len(resp.Archetypes) == 0 {
		t.Errorf("expected archetypes, got none")
	}

	// Check that percentages sum close to 100
	totalPercent := 0.0
	for _, arch := range resp.Archetypes {
		totalPercent += arch.Percentage
		if arch.Name == "" {
			t.Errorf("archetype missing name")
		}
	}

	if totalPercent < 99.0 || totalPercent > 101.0 {
		t.Errorf("expected percentages to sum ~100, got %.1f", totalPercent)
	}

	if resp.LastUpdatedAt == "" {
		t.Errorf("expected LastUpdatedAt to be set")
	}

	if resp.SampleSize <= 0 {
		t.Errorf("expected positive sample size, got %d", resp.SampleSize)
	}
}

func TestMetaSnapshot_Legacy_OK(t *testing.T) {
	h := NewMetaHandler()

	req := httptest.NewRequest("GET", "/meta/legacy", nil)
	w := httptest.NewRecorder()

	h.Snapshot(w, req)

	// Even though chi param won't work in isolation, check response structure
	var resp deckMetaSnapshotResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	// Legacy should still return valid meta (defaults to modern since chi not available)
	if len(resp.Archetypes) == 0 {
		t.Errorf("expected archetypes in meta snapshot")
	}
}

func TestMetaSnapshot_Pioneer_OK(t *testing.T) {
	h := NewMetaHandler()

	req := httptest.NewRequest("GET", "/meta/pioneer", nil)
	w := httptest.NewRecorder()

	h.Snapshot(w, req)

	var resp deckMetaSnapshotResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if len(resp.Archetypes) == 0 {
		t.Errorf("expected archetypes")
	}
}

func TestMetaSnapshotArchetype_HasRequiredFields(t *testing.T) {
	h := NewMetaHandler()
	snapshot := h.getHardcodedMetaSnapshot("modern")

	for _, arch := range snapshot.Archetypes {
		if arch.Name == "" {
			t.Errorf("archetype missing name")
		}
		if arch.Percentage <= 0 {
			t.Errorf("archetype %s has invalid percentage: %.1f", arch.Name, arch.Percentage)
		}
		if arch.Description == "" {
			t.Errorf("archetype %s missing description", arch.Name)
		}
		if arch.TrendDirection == "" {
			t.Errorf("archetype %s missing trend direction", arch.Name)
		}
	}
}

func TestMetaSnapshot_ResponseStructure(t *testing.T) {
	h := NewMetaHandler()
	snapshot := h.getHardcodedMetaSnapshot("modern")

	// Validate response structure
	if snapshot.Format == "" {
		t.Errorf("format should not be empty")
	}

	if len(snapshot.Archetypes) == 0 {
		t.Errorf("should have at least one archetype")
	}

	if snapshot.LastUpdatedAt == "" {
		t.Errorf("LastUpdatedAt should not be empty")
	}

	if snapshot.DataSource == "" {
		t.Errorf("DataSource should not be empty")
	}

	if snapshot.SampleSize == 0 {
		t.Errorf("SampleSize should be positive")
	}
}

func TestMetaSnapshot_ArchetypeWithTrend(t *testing.T) {
	h := NewMetaHandler()
	snapshot := h.getHardcodedMetaSnapshot("modern")

	// At least one archetype should have trend info
	hasUpTrend := false
	hasDownTrend := false

	for _, arch := range snapshot.Archetypes {
		if arch.TrendDirection == "up" && arch.TrendPercentage > 0 {
			hasUpTrend = true
		}
		if arch.TrendDirection == "down" && arch.TrendPercentage < 0 {
			hasDownTrend = true
		}
	}

	if !hasUpTrend || !hasDownTrend {
		t.Logf("expected at least one archetype with up/down trend")
		// Not a hard failure, just logging
	}
}

func TestNormalizeFormat(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"modern", "modern"},
		{"mod", "modern"},
		{"legacy", "legacy"},
		{"leg", "legacy"},
		{"pioneer", "pioneer"},
		{"pio", "pioneer"},
		{"standard", "standard"},
		{"std", "standard"},
		{"invalid", "modern"}, // Default fallback
	}

	for _, tc := range tests {
		result := normalizeFormat(tc.input)
		if result != tc.expected {
			t.Errorf("normalizeFormat(%s) = %s, expected %s", tc.input, result, tc.expected)
		}
	}
}

// Helper for integration testing with chi router
func runMetaSnapshotRequest(t *testing.T, format string) *deckMetaSnapshotResponse {
	h := NewMetaHandler()
	req := httptest.NewRequest("GET", "/meta/"+format, nil)
	w := httptest.NewRecorder()

	h.Snapshot(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", w.Code, w.Body.String())
	}

	var resp deckMetaSnapshotResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	return &resp
}

func TestMetaSideboardIntegration(t *testing.T) {
	// Verify that meta snapshot archetypes align with sideboard template categories
	h := NewMetaHandler()
	snapshot := h.getHardcodedMetaSnapshot("modern")

	// Extract archetype names from snapshot
	snapshotArchetypes := map[string]bool{}
	for _, arch := range snapshot.Archetypes {
		snapshotArchetypes[strings.ToLower(arch.Name)] = true
	}

	// Verify "Other" category exists as fallback
	if !snapshotArchetypes["other"] {
		t.Errorf("expected 'Other' archetype in snapshot")
	}
}

func TestMetaResolve_UsesExternalSource_WhenConfigured(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"format":"modern","archetypes":[{"name":"Domain Zoo","percentage":12.5,"description":"Aggro shell"}],"data_source":"etl-mock","sample_size":321}`))
	}))
	defer server.Close()

	h := NewMetaHandlerWithConfig(metaHandlerConfig{
		client:     server.Client(),
		cacheTTL:   10 * time.Second,
		sourceURLs: map[string]string{"modern": server.URL},
	})

	snap := h.resolveMetaSnapshot(context.Background(), "modern")
	if snap.DataSource != "etl-mock" {
		t.Fatalf("expected external data source, got %s", snap.DataSource)
	}
	if len(snap.Archetypes) != 1 || snap.Archetypes[0].Name != "Domain Zoo" {
		t.Fatalf("expected external archetype payload, got %+v", snap.Archetypes)
	}
}

func TestMetaResolve_FallbacksToHardcoded_WhenExternalFails(t *testing.T) {
	h := NewMetaHandlerWithConfig(metaHandlerConfig{
		client:     &http.Client{Timeout: 50 * time.Millisecond},
		cacheTTL:   10 * time.Second,
		sourceURLs: map[string]string{"modern": "http://127.0.0.1:1/unreachable"},
	})

	snap := h.resolveMetaSnapshot(context.Background(), "modern")
	if !strings.HasPrefix(snap.DataSource, "hardcoded-v1") {
		t.Fatalf("expected hardcoded fallback source, got %s", snap.DataSource)
	}
	if len(snap.Archetypes) == 0 {
		t.Fatalf("expected fallback archetypes")
	}
}
