package llm

import (
	"context"
	"testing"

	"github.com/gigliofr/mana-wise/domain"
)

func TestConnectorSuggestions_NilReceiver_ReturnsError(t *testing.T) {
	var c *Connector

	_, err := c.Suggestions(context.Background(), "hash", "decklist", "standard", "en", &domain.AnalysisResult{})
	if err == nil {
		t.Fatal("expected error for nil connector receiver")
	}
}

func TestConnectorEmbedText_NilReceiver_ReturnsError(t *testing.T) {
	var c *Connector

	_, err := c.EmbedText(context.Background(), "Lightning Bolt")
	if err == nil {
		t.Fatal("expected error for nil connector receiver")
	}
}
