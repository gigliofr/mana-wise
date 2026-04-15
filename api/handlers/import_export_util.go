package handlers

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/gigliofr/mana-wise/domain"
)

// DecklistParser handles parsing various decklist formats.
type DecklistParser interface {
	Parse(input string) ([]decklistEntry, []string, error) // entries, warnings, error
}

// decklistEntry represents a parsed card entry before resolution.
type decklistEntry struct {
	CardName   string
	Quantity   int
	IsSideboard bool
	SetCode    string // Optional: expansion set code
}

// ArenaParser parses Magic: The Gathering Arena export format.
// Format: "2 Solitude (STX) 100"
type ArenaParser struct{}

func (p *ArenaParser) Parse(input string) ([]decklistEntry, []string, error) {
	return parseGenericFormat(input, `^(\d+)\s+(.+?)\s*(?:\(([A-Z0-9]+)\))?(?:\s+\d+)?$`, false)
}

// MTGOParser parses Magic Online export format.
// Format: "2x Solitude (STX) #123"
type MTGOParser struct{}

func (p *MTGOParser) Parse(input string) ([]decklistEntry, []string, error) {
	return parseGenericFormat(input, `^(\d+)x?\s+(.+?)\s*(?:\(([A-Z0-9]+)\))?(?:\s+#\d+)?$`, false)
}

// TextParser parses simple text format (Moxfield / MTGGoldfish style).
// Format: "2 Solitude"
type TextParser struct{}

func (p *TextParser) Parse(input string) ([]decklistEntry, []string, error) {
	return parseGenericFormat(input, `^(\d+)\s+(.+)$`, false)
}

// ArchidektParser parses Archidekt JSON-encoded format (simplified).
// For now, treat as generic text format.
type ArchidektParser struct{}

func (p *ArchidektParser) Parse(input string) ([]decklistEntry, []string, error) {
	return parseGenericFormat(input, `^(\d+)\s+(.+)$`, false)
}

// parseGenericFormat is the core parser for most Magic formats.
func parseGenericFormat(input string, linePattern string, isSideboard bool) ([]decklistEntry, []string, error) {
	entries := []decklistEntry{}
	warnings := []string{}

	re, err := regexp.Compile(linePattern)
	if err != nil {
		return nil, warnings, err
	}

	lines := strings.Split(input, "\n")
	// Track if we've entered sideboard section (for formats that separate main from sideboard)
	enteringSideboard := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}

		// Sideboard marker detection (generic format) - CHECK BEFORE PARSING
		if strings.ToLower(line) == "sideboard:" || strings.ToLower(line) == "sb:" || strings.ToLower(line) == "sideboard" {
			enteringSideboard = true
			continue
		}

		// Try to match card line
		matches := re.FindStringSubmatch(line)
		if len(matches) < 3 {
			if !strings.HasPrefix(line, "//") && line != "" {
				warnings = append(warnings, "Could not parse line: "+line)
			}
			continue
		}

		qty := 0
		if qtyStr := strings.TrimSpace(matches[1]); qtyStr != "" {
			if parsedQty, err := strconv.Atoi(qtyStr); err == nil {
				qty = parsedQty
			}
		}
		cardName := strings.TrimSpace(matches[2])
		setCode := ""
		if len(matches) > 3 && matches[3] != "" {
			setCode = strings.TrimSpace(matches[3])
		}

		if qty > 0 && cardName != "" {
			entries = append(entries, decklistEntry{
				CardName:    cardName,
				Quantity:    qty,
				IsSideboard: enteringSideboard || isSideboard,
				SetCode:     setCode,
			})
		}
	}

	return entries, warnings, nil
}

// =============================================================================
// Export formatters
// =============================================================================

// DecklistExporter handles exporting deck to various formats.
type DecklistExporter interface {
	Export(cards []domain.DeckCard, includeSetCode bool) string
}

// ArenaExporter exports to Arena format.
type ArenaExporter struct{}

func (e *ArenaExporter) Export(cards []domain.DeckCard, includeSetCode bool) string {
	/*
	   Arena format:
	   1 Island (M21) 1
	   1 Mountain (M21) 2
	   ...
	   Sideboard (optional)
	   1 Surgical Extraction (XXX) 123
	*/
	mainboard := make([]string, 0)
	sideboard := make([]string, 0)

	for _, card := range cards {
		line := formatArenaCardLine(card)
		if card.IsSideboard {
			sideboard = append(sideboard, line)
		} else {
			mainboard = append(mainboard, line)
		}
	}

	result := strings.Join(mainboard, "\n")
	if len(sideboard) > 0 {
		result += "\n\nSideboard\n"
		result += strings.Join(sideboard, "\n")
	}
	return result
}

func formatArenaCardLine(card domain.DeckCard) string {
	// Format: "2 Solitude (STX) 100"
	return strconv.Itoa(card.Quantity) + " " + card.CardName
	// Note: In production, set code and collector number would come from card metadata
}

// MTGOExporter exports to MTGO format.
type MTGOExporter struct{}

func (e *MTGOExporter) Export(cards []domain.DeckCard, includeSetCode bool) string {
	/*
	   MTGO format:
	   2x Solitude (STX) #123
	   ...
	   Sideboard
	   1x Surgical Extraction (XXX) #456
	*/
	mainboard := make([]string, 0)
	sideboard := make([]string, 0)

	for _, card := range cards {
		qty := card.Quantity
		line := strconv.Itoa(qty) + "x " + card.CardName
		if card.IsSideboard {
			sideboard = append(sideboard, line)
		} else {
			mainboard = append(mainboard, line)
		}
	}

	result := strings.Join(mainboard, "\n")
	if len(sideboard) > 0 {
		result += "\n\nSideboard\n"
		result += strings.Join(sideboard, "\n")
	}
	return result
}

// TextExporter exports to simple text format (Moxfield-compatible).
type TextExporter struct{}

func (e *TextExporter) Export(cards []domain.DeckCard, includeSetCode bool) string {
	/*
	   Text format (Moxfield):
	   2 Solitude
	   3 Mountain
	   ...
	   Sideboard
	   1 Surgical Extraction
	*/
	mainboard := make([]string, 0)
	sideboard := make([]string, 0)

	for _, card := range cards {
		qty := card.Quantity
		line := strconv.Itoa(qty) + " " + card.CardName
		if card.IsSideboard {
			sideboard = append(sideboard, line)
		} else {
			mainboard = append(mainboard, line)
		}
	}

	result := strings.Join(mainboard, "\n")
	if len(sideboard) > 0 {
		result += "\n\nSideboard\n"
		result += strings.Join(sideboard, "\n")
	}
	return result
}

// GetParserForFormat returns the appropriate parser for a given format.
func GetParserForFormat(format string) DecklistParser {
	switch strings.ToLower(format) {
	case "arena":
		return &ArenaParser{}
	case "mtgo":
		return &MTGOParser{}
	case "moxfield", "text":
		return &TextParser{}
	case "archidekt":
		return &ArchidektParser{}
	default:
		return &TextParser{} // Default to text format
	}
}

// GetExporterForFormat returns the appropriate exporter for a given format.
func GetExporterForFormat(format string) DecklistExporter {
	switch strings.ToLower(format) {
	case "arena":
		return &ArenaExporter{}
	case "mtgo":
		return &MTGOExporter{}
	case "moxfield", "text":
		return &TextExporter{}
	default:
		return &TextExporter{} // Default to text format
	}
}
