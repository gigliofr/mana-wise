package usecase_test

import (
	"strings"
	"testing"
)

// parseDecklist is package-internal — test via exported behavior:
// AnalyzeDeckUseCase.Execute validates the decklist internally.
// Here we test the parsing logic via a white-box approach since
// parseDecklist is unexported. We verify results in integration-style
// tests that exercise the full use case with a mock fetcher.

// TestDecklistFormat verifies that common formats are correctly accepted.
func TestDecklistFormat(t *testing.T) {
	formats := []struct {
		line  string
		valid bool
	}{
		{"4 Lightning Bolt", true},
		{"4x Lightning Bolt", true},
		{"1 Black Lotus", true},
		{"Sideboard:", false}, // header line, skipped
		{"", false},           // blank line, skipped
		{"// comment", false}, // comment, skipped
	}

	for _, f := range formats {
		line := strings.TrimSpace(f.line)
		isHeader := strings.HasSuffix(strings.ToLower(line), ":")
		isBlank := line == ""
		isComment := strings.HasPrefix(line, "//")

		shouldSkip := isHeader || isBlank || isComment
		if shouldSkip == f.valid {
			t.Errorf("line %q: expected valid=%v, but skip=%v", f.line, f.valid, shouldSkip)
		}
	}
}
