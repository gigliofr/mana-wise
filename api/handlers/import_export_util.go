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
	CardName    string
	Quantity    int
	IsSideboard bool
	IsCommander bool
	SetCode     string // Optional: expansion set code
}

// ArenaParser parses Magic: The Gathering Arena export format.
// Format: "2 Solitude (STX) 100"
type ArenaParser struct{}

func (p *ArenaParser) Parse(input string) ([]decklistEntry, []string, error) {
	return parseGenericFormat(input, `^(\d+)x?\s+(.+)$`, false)
}

// MTGOParser parses Magic Online export format.
// Format: "2x Solitude (STX) #123"
type MTGOParser struct{}

func (p *MTGOParser) Parse(input string) ([]decklistEntry, []string, error) {
	return parseGenericFormat(input, `^(\d+)x?\s+(.+)$`, false)
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
	enteringCommander := false

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}

		// Sideboard marker detection (generic format) - CHECK BEFORE PARSING
		switch strings.ToLower(line) {
		case "sideboard:", "sb:", "sideboard":
			enteringSideboard, enteringCommander = true, false
			continue
		case "commander", "commander:":
			enteringCommander, enteringSideboard = true, false
			continue
		case "deck", "deck:", "main", "mainboard", "mainboard:":
			enteringCommander, enteringSideboard = false, false
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
		rawName := strings.TrimSpace(matches[2])
		isCommander := enteringCommander || importedCommanderTagRe.MatchString(rawName)
		cardName := normalizeImportedCardName(rawName)
		setCode := ""
		if m := importedSetCodeRe.FindStringSubmatch(rawName); len(m) == 2 {
			setCode = strings.ToUpper(m[1])
		}

		if qty > 0 && cardName != "" {
			entries = append(entries, decklistEntry{
				CardName:    cardName,
				Quantity:    qty,
				IsSideboard: enteringSideboard || isSideboard,
				IsCommander: isCommander,
				SetCode:     setCode,
			})
		}
	}

	return entries, warnings, nil
}

var importedDeckTagRe = regexp.MustCompile(`\s*\[[^\]]+\]\s*$`)
var importedFoilMarkerRe = regexp.MustCompile(`(?i)\s*\*(?:f|e|foil|etched)\*\s*$`)

// "(SET) 123", including collector numbers like "247p", "D25", "M15-7".
var importedSetSuffixRe = regexp.MustCompile(`(?i)\s*\([a-z0-9]{2,10}\)(?:\s+[a-z0-9][a-z0-9★†-]*)?\s*$`)
var importedSetCodeRe = regexp.MustCompile(`(?i)\(([a-z0-9]{2,10})\)`)
var importedCommanderTagRe = regexp.MustCompile(`(?i)\[[^\]]*commander[^\]]*\]`)
var importedSingleSlashRe = regexp.MustCompile(`\s+/\s+`)

// normalizeImportedCardName removes tags ("[Commander]"), foil markers ("*F*")
// and set/collector suffixes, in any order, and turns "Ready / Willing" into
// the Scryfall name "Ready // Willing".
func normalizeImportedCardName(raw string) string {
	name := strings.TrimSpace(raw)
	for name != "" {
		next := importedDeckTagRe.ReplaceAllString(name, "")
		next = importedFoilMarkerRe.ReplaceAllString(next, "")
		next = strings.TrimSpace(importedSetSuffixRe.ReplaceAllString(next, ""))
		if next == name {
			break
		}
		name = next
	}
	name = strings.Join(strings.Fields(name), " ")
	if !strings.Contains(name, "//") {
		name = importedSingleSlashRe.ReplaceAllString(name, " // ")
	}
	return name
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
