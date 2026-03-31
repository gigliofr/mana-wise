package domain

import (
	"strings"
	"time"
)

// DeckCard represents a card entry within a deck.
type DeckCard struct {
	CardID      string `bson:"card_id"  json:"card_id"`
	CardName    string `bson:"card_name" json:"card_name"`
	Quantity    int    `bson:"quantity" json:"quantity"`
	IsSideboard bool   `bson:"is_sideboard" json:"is_sideboard"`
	IsCommander bool   `bson:"is_commander" json:"is_commander"`
}

// DeckChange represents a card delta between two deck versions.
type DeckChange struct {
	Op   string `bson:"op"   json:"op"` // add|remove
	Card string `bson:"card" json:"card"`
	Qty  int    `bson:"qty"  json:"qty"`
}

// DeckVersion stores one historical snapshot and its summary diff.
type DeckVersion struct {
	V        int         `bson:"v" json:"v"`
	Date     time.Time   `bson:"date" json:"date"`
	Changes  []DeckChange `bson:"changes,omitempty" json:"changes,omitempty"`
	Note     string      `bson:"note,omitempty" json:"note,omitempty"`
	Snapshot []DeckCard  `bson:"snapshot" json:"snapshot"`
}

// Deck represents a player's deck.
type Deck struct {
	ID          string     `bson:"_id"       json:"id"`
	UserID      string     `bson:"user_id"   json:"user_id"`
	Name        string     `bson:"name"      json:"name"`
	Format      string     `bson:"format"    json:"format"`
	Cards       []DeckCard `bson:"cards"   json:"cards"`
	Version     int        `bson:"version,omitempty" json:"version,omitempty"`
	History     []DeckVersion `bson:"history,omitempty" json:"history,omitempty"`
	Description string     `bson:"description" json:"description"`
	IsPublic    bool       `bson:"is_public" json:"is_public"`
	CreatedAt   time.Time  `bson:"created_at" json:"created_at"`
	UpdatedAt   time.Time  `bson:"updated_at" json:"updated_at"`
}

// MainboardCards returns only mainboard cards (non-sideboard).
func (d *Deck) MainboardCards() []DeckCard {
	result := make([]DeckCard, 0, len(d.Cards))
	for _, c := range d.Cards {
		if !c.IsSideboard {
			result = append(result, c)
		}
	}
	return result
}

// TotalCards returns the total card count (mainboard).
func (d *Deck) TotalCards() int {
	total := 0
	for _, c := range d.MainboardCards() {
		total += c.Quantity
	}
	return total
}

// SupportedFormats is the list of MTG formats supported by ManaWise.
var SupportedFormats = []string{
	"standard",
	"pioneer",
	"modern",
	"legacy",
	"vintage",
	"commander",
	"pauper",
}

var formatAliases = map[string]string{
	"edh":       "commander",
	"commander": "commander",
	"std":       "standard",
	"standard":  "standard",
	"pio":       "pioneer",
	"pioneer":   "pioneer",
	"modern":    "modern",
	"legacy":    "legacy",
	"vintage":   "vintage",
	"pauper":    "pauper",
}

// NormalizeFormat maps aliases to canonical format names.
func NormalizeFormat(format string) string {
	key := strings.ToLower(strings.TrimSpace(format))
	if canonical, ok := formatAliases[key]; ok {
		return canonical
	}
	return key
}

// IsValidFormat returns true if the format name is supported.
func IsValidFormat(format string) bool {
	format = NormalizeFormat(format)
	for _, f := range SupportedFormats {
		if f == format {
			return true
		}
	}
	return false
}
