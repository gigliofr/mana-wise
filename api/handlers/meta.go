package handlers

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
)

// MetaHandler handles meta snapshot and archetype distribution queries.
type MetaHandler struct{}

// NewMetaHandler creates a MetaHandler.
func NewMetaHandler() *MetaHandler {
	return &MetaHandler{}
}

type metaArchetype struct {
	Name              string   `json:"name"`
	Percentage        float64  `json:"percentage"`
	Description       string   `json:"description"`
	SideboardSample   []string `json:"sideboard_sample,omitempty"`
	TrendDirection    string   `json:"trend_direction,omitempty"` // "up", "down", "stable"
	TrendPercentage   float64  `json:"trend_percentage,omitempty"`
	PopularCards      []string `json:"popular_cards,omitempty"`
}

type deckMetaSnapshotResponse struct {
	Format           string            `json:"format"`
	Archetypes       []metaArchetype   `json:"archetypes"`
	LastUpdatedAt    string            `json:"last_updated_at"`
	DataSource       string            `json:"data_source"`
	SampleSize       int               `json:"sample_size"`
}

// Snapshot returns the current meta distribution for a given format.
// GET /api/v1/meta/{format}
func (h *MetaHandler) Snapshot(w http.ResponseWriter, r *http.Request) {
	format := chi.URLParam(r, "format")
	if format == "" {
		format = "modern"
	}
	format = normalizeFormat(format)

	// V1: Hardcoded meta snapshot with realistic distribution.
	// In future iterations, this will be sourced from MTGGoldfish/MTGTOP8 ETL.
	snapshot := h.getMetaSnapshot(format)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(snapshot)
}

func (h *MetaHandler) getMetaSnapshot(format string) deckMetaSnapshotResponse {
	// Realistic Modern meta distribution (March 31, 2026 baseline).
	// Percentages reflect typical metagame composition.
	archetypes := []metaArchetype{}

	switch format {
	case "modern":
		archetypes = []metaArchetype{
			{
				Name:              "Scam",
				Percentage:        18.5,
				Description:       "Rakdos tempo with Fury + Counterspell interactive shell",
				TrendDirection:    "stable",
				TrendPercentage:   0.2,
				SideboardSample:   []string{"Temporary Lockdown", "Zealous Persecution", "Unholy Heat"},
				PopularCards:      []string{"Fury", "Murktide", "Dress Down", "Counterspell", "Subtlety"},
			},
			{
				Name:              "Rhinos",
				Percentage:        16.2,
				Description:       "Cascade-focused Temur ramp with Subtlety+Spell Pierce",
				TrendDirection:    "up",
				TrendPercentage:   1.5,
				SideboardSample:   []string{"Teferi, Time Raveler", "Mystical Dispute", "Engineered Explosives"},
				PopularCards:      []string{"Subtlety", "Fury", "Dress Down", "Snapcaster Mage", "Solitude"},
			},
			{
				Name:              "Murktide",
				Percentage:        14.8,
				Description:       "Izzet tempo control with Murktide + Counterspell stack",
				TrendDirection:    "stable",
				TrendPercentage:   -0.3,
				SideboardSample:   []string{"Mystical Dispute", "Dress Down", "Engineered Explosives"},
				PopularCards:      []string{"Counterspell", "Murktide", "Dregscape Zombie", "Ledger Shredder", "Snapcaster"},
			},
			{
				Name:              "Hammer Time",
				Percentage:        11.3,
				Description:       "Hardened scales aggro with construct equipment synergies",
				TrendDirection:    "down",
				TrendPercentage:   -2.1,
				SideboardSample:   []string{"Damping Sphere", "Surgical Extraction", "Auriok Sentinel"},
				PopularCards:      []string{"Stoneforge Mystic", "Sigil of Stagmire", "Hammer of Bogardan", "Urza's Saga"},
			},
			{
				Name:              "Hardened Scales",
				Percentage:        10.5,
				Description:       "Mono-green +1/+1 counters with Scales synergy creatures",
				TrendDirection:    "up",
				TrendPercentage:   0.8,
				SideboardSample:   []string{"Damping Sphere", "Mystical Dispute", "Blood Moon"},
				PopularCards:      []string{"Hardened Scales", "Saga", "Murktide", "Dress Down", "Counterspell"},
			},
			{
				Name:              "4c Control",
				Percentage:        9.2,
				Description:       "Esper-based control with Subtlety + creature synergies",
				TrendDirection:    "stable",
				TrendPercentage:   0.1,
				SideboardSample:   []string{"Endurance", "Teferi", "Narset", "Unlicensed Hearse", "Dress Down"},
				PopularCards:      []string{"Counterspell", "Snapcaster Mage", "Dress Down", "Teferi", "Murktide"},
			},
			{
				Name:              "Living End",
				Percentage:        7.4,
				Description:       "Cascade combo with Living End game-ender",
				TrendDirection:    "down",
				TrendPercentage:   -0.9,
				SideboardSample:   []string{"Surgical Extraction", "Weather the Storm", "Shadow of Doubt"},
				PopularCards:      []string{"Living End", "Grief", "Fury", "Dress Down", "Subtlety"},
			},
			{
				Name:              "Murktide Murktide",
				Percentage:        6.8,
				Description:       "Mix of tempo + Murktide shell (secondary variants)",
				TrendDirection:    "stable",
				TrendPercentage:   0.0,
				SideboardSample:   []string{"Engineered Explosives", "Subtlety", "Snapcaster"},
				PopularCards:      []string{"Murktide", "Subtlety", "Counterspell", "Dress Down", "Engineered Explosives"},
			},
			{
				Name:              "Other",
				Percentage:        5.3,
				Description:       "Misc archetypes: Blitz, Pox, Elementals, Golgari variants",
				TrendDirection:    "stable",
				TrendPercentage:   0.0,
				SideboardSample:   []string{},
				PopularCards:      []string{},
			},
		}

	case "legacy":
		// Legacy meta (March 31, 2026 baseline)
		archetypes = []metaArchetype{
			{
				Name:              "Murktide",
				Percentage:        22.1,
				Description:       "Izzet tempo with Counterspell + Murktide",
				TrendDirection:    "up",
				TrendPercentage:   1.2,
				SideboardSample:   []string{"Subtlety", "Mystical Dispute", "Dress Down", "Engineered Explosives"},
				PopularCards:      []string{"Counterspell", "Murktide", "Force of Will", "Snapcaster Mage"},
			},
			{
				Name:              "4c Control",
				Percentage:        18.5,
				Description:       "Esper-based control with access to Abrupt Decay",
				TrendDirection:    "stable",
				TrendPercentage:   0.3,
				SideboardSample:   []string{"Teferi", "Carpet of Flowers", "Surgical Extraction"},
				PopularCards:      []string{"Counterspell", "Force of Will", "Counterspell", "Jace", "Teferi"},
			},
			{
				Name:              "Heezy",
				Percentage:        15.8,
				Description:       "Heliod-based combo with Ballista shutdown",
				TrendDirection:    "down",
				TrendPercentage:   -1.5,
				SideboardSample:   []string{"Surgical Extraction", "Nihil Rod", "Yavimaya Graveyard"},
				PopularCards:      []string{"Heliod", "Walking Ballista", "Archon of Redemption"},
			},
			{
				Name:              "Reanimator",
				Percentage:        12.3,
				Description:       "Grixis reanimator with Grief + Archon",
				TrendDirection:    "stable",
				TrendPercentage:   0.1,
				SideboardSample:   []string{"Surgical Extraction", "Endurance", "Dress Down"},
				PopularCards:      []string{"Grief", "Fury", "Exhume", "Dress Down"},
			},
			{
				Name:              "Omni-Tell",
				Percentage:        10.1,
				Description:       "Omniscience combo with Haul tellurion",
				TrendDirection:    "down",
				TrendPercentage:   -0.8,
				SideboardSample:   []string{"Flusterstorm", "Mystical Dispute"},
				PopularCards:      []string{"Omniscience", "Deep Thought", "Petals"},
			},
			{
				Name:              "Other",
				Percentage:        21.2,
				Description:       "Misc: Shops, ANT, TES, Elves, Ninjas, etc.",
				TrendDirection:    "stable",
				TrendPercentage:   0.0,
				SideboardSample:   []string{},
				PopularCards:      []string{},
			},
		}

	case "pioneer":
		// Pioneer meta (March 31, 2026 baseline)
		archetypes = []metaArchetype{
			{
				Name:              "Azorius",
				Percentage:        19.4,
				Description:       "Azorius control with Teferi + sweepers",
				TrendDirection:    "stable",
				TrendPercentage:   0.2,
				SideboardSample:   []string{"Teferi", "Jace", "Counterspell clones"},
				PopularCards:      []string{"Teferi", "Counterspell", "Dress Down", "Ethereal Valkyrie"},
			},
			{
				Name:              "Greasefang",
				Percentage:        16.7,
				Description:       "Mardu reanimator with Greasefang Okiba-Gang",
				TrendDirection:    "up",
				TrendPercentage:   1.8,
				SideboardSample:   []string{"Surgical Extraction", "Subtlety"},
				PopularCards:      []string{"Greasefang", "Archon of Cruelty", "Grief", "Dress Down"},
			},
			{
				Name:              "Rakdos Midrange",
				Percentage:        14.2,
				Description:       "Rakdos tempo creatures with discard",
				TrendDirection:    "stable",
				TrendPercentage:   -0.1,
				SideboardSample:   []string{"Aggressive discard"},
				PopularCards:      []string{"Fury", "Dreadhorde", "Thoughtseize", "Counterspell"},
			},
			{
				Name:              "Lotus Combo",
				Percentage:        11.8,
				Description:       "Lotus Field combo win conditions",
				TrendDirection:    "down",
				TrendPercentage:   -1.2,
				SideboardSample:   []string{"Subtlety", "Counterspell clones"},
				PopularCards:      []string{"Lotus Field", "Scute Swarm"},
			},
			{
				Name:              "Gruul Devotion",
				Percentage:        10.5,
				Description:       "Devotion-based greenred creatures",
				TrendDirection:    "stable",
				TrendPercentage:   0.0,
				SideboardSample:   []string{"Artifact hate", "Sweepers"},
				PopularCards:      []string{"Nykthos", "Embercleave"},
			},
			{
				Name:              "Other",
				Percentage:        27.4,
				Description:       "Misc: Abzan, Temur, Gruul Blitz, Spirits, etc.",
				TrendDirection:    "stable",
				TrendPercentage:   0.0,
				SideboardSample:   []string{},
				PopularCards:      []string{},
			},
		}

	default:
		// Fallback for unknown format: generic meta
		archetypes = []metaArchetype{
			{
				Name:              "Control",
				Percentage:        30.0,
				Description:       "Control shells with interactive elements",
				TrendDirection:    "stable",
				TrendPercentage:   0.0,
				PopularCards:      []string{},
			},
			{
				Name:              "Midrange",
				Percentage:        25.0,
				Description:       "Creature-based midrange threats",
				TrendDirection:    "stable",
				TrendPercentage:   0.0,
				PopularCards:      []string{},
			},
			{
				Name:              "Aggro",
				Percentage:        20.0,
				Description:       "Fast creature strategies",
				TrendDirection:    "stable",
				TrendPercentage:   0.0,
				PopularCards:      []string{},
			},
			{
				Name:              "Combo",
				Percentage:        15.0,
				Description:       "Combo-based decks",
				TrendDirection:    "stable",
				TrendPercentage:   0.0,
				PopularCards:      []string{},
			},
			{
				Name:              "Other",
				Percentage:        10.0,
				Description:       "Misc strategies",
				TrendDirection:    "stable",
				TrendPercentage:   0.0,
				PopularCards:      []string{},
			},
		}
	}

	return deckMetaSnapshotResponse{
		Format:        format,
		Archetypes:    archetypes,
		LastUpdatedAt: time.Now().UTC().Format(time.RFC3339),
		DataSource:    "hardcoded-v1-placeholder", // Will be replaced by MTGGoldfish/MTGTOP8 ETL in future
		SampleSize:    1000,                        // Placeholder sample size
	}
}

// normalizeFormat standardizes format names for routing.
func normalizeFormat(input string) string {
	switch input {
	case "mod", "modern":
		return "modern"
	case "leg", "legacy":
		return "legacy"
	case "pio", "pioneer":
		return "pioneer"
	case "std", "standard":
		return "standard"
	default:
		return "modern" // Default fallback
	}
}
