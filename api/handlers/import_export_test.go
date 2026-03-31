package handlers

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/manawise/api/domain"
)

// Test parsers
func TestArenaParser_OK(t *testing.T) {
	input := `2 Solitude (SIR) 100
1 Fury (TSR) 200

Sideboard
3 Surgical Extraction (NPH) 300`

	parser := &ArenaParser{}
	entries, warnings, err := parser.Parse(input)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(entries) != 3 {
		t.Errorf("expected 3 entries, got %d", len(entries))
	}

	if entries[0].CardName != "Solitude" || entries[0].Quantity != 2 || entries[0].IsSideboard {
		t.Errorf("first card: expected '2 Solitude', got '%d %s sideboard=%v'", entries[0].Quantity, entries[0].CardName, entries[0].IsSideboard)
	}

	if entries[1].CardName != "Fury" || entries[1].Quantity != 1 {
		t.Errorf("second card: expected '1 Fury', got '%d %s'", entries[1].Quantity, entries[1].CardName)
	}

	if !entries[2].IsSideboard {
		t.Errorf("third card should be sideboard entry")
	}

	if entries[2].CardName != "Surgical Extraction" {
		t.Errorf("sideboard card: expected 'Surgical Extraction', got '%s'", entries[2].CardName)
	}

	if len(warnings) > 0 {
		t.Logf("warnings: %v", warnings)
	}
}

func TestMTGOParser_OK(t *testing.T) {
	input := `2x Solitude
3x Fury

Sideboard
1x Surgical Extraction`

	parser := &MTGOParser{}
	entries, warnings, err := parser.Parse(input)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(entries) != 3 {
		t.Errorf("expected 3 entries, got %d; entries=%+v", len(entries), entries)
	}

	if len(entries) > 0 && entries[0].Quantity != 2 {
		t.Errorf("first quantity: expected 2, got %d", entries[0].Quantity)
	}

	if len(entries) > 2 && !entries[2].IsSideboard {
		t.Errorf("third card should be sideboard (got main=%v)", !entries[2].IsSideboard)
	}

	if len(entries) > 2 && entries[2].CardName != "Surgical Extraction" {
		t.Errorf("sideboard card: expected 'Surgical Extraction', got '%s'", entries[2].CardName)
	}

	_ = warnings
}

func TestTextParser_OK(t *testing.T) {
	input := `3 Mountain
2 Island
1 Solitude

Sideboard
2 Mystical Dispute`

	parser := &TextParser{}
	entries, warnings, err := parser.Parse(input)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(entries) != 4 {
		t.Errorf("expected 4 entries (3 main + 1 sideboard), got %d; entries=%+v", len(entries), entries)
	}

	if len(entries) > 0 && entries[0].CardName != "Mountain" {
		t.Errorf("first card: expected 'Mountain', got '%s'", entries[0].CardName)
	}

	if len(entries) > 0 && entries[0].Quantity != 3 {
		t.Errorf("first quantity: expected 3, got %d", entries[0].Quantity)
	}

	if len(entries) > 3 && !entries[3].IsSideboard {
		t.Errorf("fourth card should be sideboard")
	}

	_ = warnings
}

func TestParser_IgnoresComments(t *testing.T) {
	input := `// This is a comment
2 Solitude
// Another comment
1 Fury`

	parser := &TextParser{}
	entries, warnings, err := parser.Parse(input)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(entries) != 2 {
		t.Errorf("expected 2 entries (ignoring comments), got %d", len(entries))
	}

	_ = warnings
}

func TestParser_EmptyInput(t *testing.T) {
	input := ""

	parser := &TextParser{}
	entries, _, err := parser.Parse(input)

	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if len(entries) != 0 {
		t.Errorf("expected 0 entries, got %d", len(entries))
	}
}

// Test exporters
func TestArenaExporter_OK(t *testing.T) {
	cards := []domain.DeckCard{
		{CardName: "Solitude", Quantity: 2, IsSideboard: false},
		{CardName: "Mountain", Quantity: 3, IsSideboard: false},
		{CardName: "Surgical Extraction", Quantity: 1, IsSideboard: true},
	}

	exporter := &ArenaExporter{}
	output := exporter.Export(cards, false)

	if !containsSubstring(output, "2 Solitude") {
		t.Errorf("expected '2 Solitude' in output")
	}

	if !containsSubstring(output, "3 Mountain") {
		t.Errorf("expected '3 Mountain' in output")
	}

	if !containsSubstring(output, "Sideboard") {
		t.Errorf("expected 'Sideboard' marker in output")
	}

	if !containsSubstring(output, "1 Surgical Extraction") {
		t.Errorf("expected '1 Surgical Extraction' in sideboard section")
	}
}

func TestMTGOExporter_OK(t *testing.T) {
	cards := []domain.DeckCard{
		{CardName: "Solitude", Quantity: 2, IsSideboard: false},
		{CardName: "Fury", Quantity: 1, IsSideboard: false},
		{CardName: "Mystical Dispute", Quantity: 2, IsSideboard: true},
	}

	exporter := &MTGOExporter{}
	output := exporter.Export(cards, false)

	if !containsSubstring(output, "2x Solitude") {
		t.Errorf("expected '2x Solitude' in output")
	}

	if !containsSubstring(output, "1x Fury") {
		t.Errorf("expected '1x Fury' in output")
	}

	if !containsSubstring(output, "Sideboard") {
		t.Errorf("expected 'Sideboard' marker in output")
	}
}

func TestTextExporter_OK(t *testing.T) {
	cards := []domain.DeckCard{
		{CardName: "Mountain", Quantity: 3, IsSideboard: false},
		{CardName: "Island", Quantity: 2, IsSideboard: false},
		{CardName: "Unlicensed Hearse", Quantity: 2, IsSideboard: true},
	}

	exporter := &TextExporter{}
	output := exporter.Export(cards, false)

	if !containsSubstring(output, "3 Mountain") {
		t.Errorf("expected '3 Mountain' in output")
	}

	if !containsSubstring(output, "2 Island") {
		t.Errorf("expected '2 Island' in output")
	}

	if !containsSubstring(output, "Sideboard") {
		t.Errorf("expected 'Sideboard' marker")
	}
}

func TestExporter_NoSideboard(t *testing.T) {
	cards := []domain.DeckCard{
		{CardName: "Solitude", Quantity: 2, IsSideboard: false},
		{CardName: "Mountain", Quantity: 3, IsSideboard: false},
	}

	exporter := &TextExporter{}
	output := exporter.Export(cards, false)

	if containsSubstring(output, "Sideboard") {
		t.Errorf("should not include 'Sideboard' marker when no sideboard cards")
	}
}

func TestGetParserForFormat(t *testing.T) {
	tests := []struct {
		format   string
		expected string
	}{
		{"arena", "*handlers.ArenaParser"},
		{"mtgo", "*handlers.MTGOParser"},
		{"text", "*handlers.TextParser"},
		{"moxfield", "*handlers.TextParser"},
		{"archidekt", "*handlers.ArchidektParser"},
		{"unknown", "*handlers.TextParser"}, // Default
	}

	for _, tc := range tests {
		parser := GetParserForFormat(tc.format)
		received := typeString(parser)
		if received != tc.expected {
			t.Errorf("GetParserForFormat(%s): expected %s, got %s", tc.format, tc.expected, received)
		}
	}
}

func TestGetExporterForFormat(t *testing.T) {
	tests := []struct {
		format   string
		expected string
	}{
		{"arena", "*handlers.ArenaExporter"},
		{"mtgo", "*handlers.MTGOExporter"},
		{"text", "*handlers.TextExporter"},
		{"moxfield", "*handlers.TextExporter"},
		{"unknown", "*handlers.TextExporter"}, // Default
	}

	for _, tc := range tests {
		exporter := GetExporterForFormat(tc.format)
		received := typeString(exporter)
		if received != tc.expected {
			t.Errorf("GetExporterForFormat(%s): expected %s, got %s", tc.format, tc.expected, received)
		}
	}
}

// Helper for round-trip: import and export should preserve deck structure
func TestImportExportRoundTrip(t *testing.T) {
	originalInput := `2 Solitude
3 Mountain
1 Fury

Sideboard
2 Surgical Extraction
1 Unlicensed Hearse`

	parser := &TextParser{}
	entries, _, err := parser.Parse(originalInput)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}

	// Convert entries to DeckCards for export
	cards := make([]domain.DeckCard, len(entries))
	for i, e := range entries {
		cards[i] = domain.DeckCard{
			CardName:    e.CardName,
			Quantity:    e.Quantity,
			IsSideboard: e.IsSideboard,
		}
	}

	exporter := &TextExporter{}
	exported := exporter.Export(cards, false)

	// Re-parse exported deck
	entries2, _, err := parser.Parse(exported)
	if err != nil {
		t.Fatalf("re-parse error: %v", err)
	}

	if len(entries) != len(entries2) {
		t.Errorf("round-trip length mismatch: %d -> %d", len(entries), len(entries2))
	}

	for i, e := range entries {
		if i >= len(entries2) {
			break
		}
		e2 := entries2[i]
		if e.CardName != e2.CardName || e.Quantity != e2.Quantity || e.IsSideboard != e2.IsSideboard {
			t.Errorf("round-trip mismatch at index %d: %+v -> %+v", i, e, e2)
		}
	}
}

// Helper function to check substring presence
func containsSubstring(haystack, needle string) bool {
	return strings.Contains(haystack, needle)
}

// Simple type introspection helper
func typeString(v interface{}) string {
	t := "unknown"
	switch v.(type) {
	case *ArenaParser:
		t = "*handlers.ArenaParser"
	case *MTGOParser:
		t = "*handlers.MTGOParser"
	case *TextParser:
		t = "*handlers.TextParser"
	case *ArchidektParser:
		t = "*handlers.ArchidektParser"
	case *ArenaExporter:
		t = "*handlers.ArenaExporter"
	case *MTGOExporter:
		t = "*handlers.MTGOExporter"
	case *TextExporter:
		t = "*handlers.TextExporter"
	}
	return t
}

// Mock for handler-level tests
type mockDeckRepo struct {
	decks map[string]*domain.Deck
}

func (m *mockDeckRepo) Create(ctx context.Context, deck *domain.Deck) error {
	if m.decks == nil {
		m.decks = make(map[string]*domain.Deck)
	}
	m.decks[deck.ID] = deck
	return nil
}

func (m *mockDeckRepo) GetByID(ctx context.Context, id string) (*domain.Deck, error) {
	if d, ok := m.decks[id]; ok {
		return d, nil
	}
	return nil, errors.New("deck not found")
}

func (m *mockDeckRepo) FindByID(ctx context.Context, id string) (*domain.Deck, error) {
	if d, ok := m.decks[id]; ok {
		return d, nil
	}
	return nil, errors.New("deck not found")
}

func (m *mockDeckRepo) List(ctx context.Context, userID string) ([]domain.Deck, error) {
	// Not implemented for this test
	return nil, nil
}

func (m *mockDeckRepo) FindByUserID(ctx context.Context, userID string) ([]*domain.Deck, error) {
	// Not implemented for this test
	return nil, nil
}

func (m *mockDeckRepo) Update(ctx context.Context, deck *domain.Deck) error {
	if m.decks == nil {
		m.decks = make(map[string]*domain.Deck)
	}
	m.decks[deck.ID] = deck
	return nil
}

func (m *mockDeckRepo) Delete(ctx context.Context, id string) error {
	delete(m.decks, id)
	return nil
}

// Mock for ResolveCardByNameUseCase
type mockResolveCardUC struct {
	cards map[string]*domain.Card
}

func (m *mockResolveCardUC) Resolve(ctx context.Context, name string) (*domain.Card, error) {
	if card, ok := m.cards[name]; ok {
		return card, nil
	}
	// Return a synthetic card for testing
	return &domain.Card{
		ID:   "synthetic-" + name,
		Name: name,
	}, nil
}
