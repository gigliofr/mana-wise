package scryfall

import (
	"net/http"
	"testing"
	"time"
)

func TestParseRetryAfter_Seconds(t *testing.T) {
	d, ok := parseRetryAfter("2")
	if !ok {
		t.Fatal("expected Retry-After seconds to parse")
	}
	if d != 2*time.Second {
		t.Fatalf("expected 2s, got %s", d)
	}
}

func TestParseRetryAfter_HTTPDateInPast(t *testing.T) {
	past := time.Now().Add(-1 * time.Minute).UTC().Format(http.TimeFormat)
	d, ok := parseRetryAfter(past)
	if !ok {
		t.Fatal("expected Retry-After http-date to parse")
	}
	if d != 0 {
		t.Fatalf("expected 0 for past date, got %s", d)
	}
}

func TestComputeRetryDelay_TooManyRequestsHasMinimumBackoff(t *testing.T) {
	d := computeRetryDelay(http.StatusTooManyRequests, 0, "")
	if d < minTooManyReqBackoff {
		t.Fatalf("expected minimum 429 backoff >= %s, got %s", minTooManyReqBackoff, d)
	}
}

func TestComputeRetryDelay_UsesRetryAfterWhenLarger(t *testing.T) {
	d := computeRetryDelay(http.StatusTooManyRequests, 0, "5")
	if d != 5*time.Second {
		t.Fatalf("expected Retry-After delay 5s, got %s", d)
	}
}

func TestComputeRetryDelay_CapsAtMaximum(t *testing.T) {
	d := computeRetryDelay(http.StatusServiceUnavailable, 10, "")
	if d != maxRetryBackoff {
		t.Fatalf("expected capped delay %s, got %s", maxRetryBackoff, d)
	}
}
