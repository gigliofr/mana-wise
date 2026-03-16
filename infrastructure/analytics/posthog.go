package analytics

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// PostHogTracker sends product analytics events to PostHog.
type PostHogTracker struct {
	apiKey string
	host   string
	client *http.Client
}

// NewPostHogTracker creates a PostHog tracker.
func NewPostHogTracker(apiKey, host string) *PostHogTracker {
	host = strings.TrimRight(strings.TrimSpace(host), "/")
	if host == "" {
		host = "https://app.posthog.com"
	}
	return &PostHogTracker{
		apiKey: strings.TrimSpace(apiKey),
		host:   host,
		client: &http.Client{Timeout: 5 * time.Second},
	}
}

// Enabled reports whether tracking can be used.
func (t *PostHogTracker) Enabled() bool {
	return t != nil && t.apiKey != ""
}

// Track sends an event to PostHog.
func (t *PostHogTracker) Track(ctx context.Context, distinctID, event string, properties map[string]interface{}) error {
	if !t.Enabled() {
		return nil
	}
	if properties == nil {
		properties = map[string]interface{}{}
	}
	properties["distinct_id"] = distinctID

	payload := map[string]interface{}{
		"api_key":     t.apiKey,
		"event":       event,
		"distinct_id": distinctID,
		"properties":  properties,
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("marshal analytics payload: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, t.host+"/capture/", bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("build analytics request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := t.client.Do(req)
	if err != nil {
		return fmt.Errorf("send analytics request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode >= 300 {
		return fmt.Errorf("analytics http status %d", resp.StatusCode)
	}
	return nil
}
