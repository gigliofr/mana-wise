package domain

// CMCBucket holds the count of cards at a given converted mana cost.
type CMCBucket struct {
	CMC   int `json:"cmc"`
	Count int `json:"count"`
}

// ManaCurveSuggestion represents a single adjustment suggestion.
type ManaCurveSuggestion struct {
	Type    string `json:"type"` // "add" | "remove"
	CMC     int    `json:"cmc"`
	Reason  string `json:"reason"`
	Urgency string `json:"urgency"` // "critical" | "moderate" | "minor"
}

// ColorSourceRequirement compares required coloured sources vs currently estimated ones.
type ColorSourceRequirement struct {
	Color    string `json:"color"`
	Required int    `json:"required"`
	Current  int    `json:"current"`
	Gap      int    `json:"gap"`
}

// CardTypeDistribution summarizes non-land card counts by macro type buckets.
type CardTypeDistribution struct {
	Creature       int `json:"creature"`
	Spell          int `json:"spell"`
	EnchantArtifact int `json:"enchant_artifact"`
	Planeswalker   int `json:"planeswalker"`
}

// ManaAnalysis is the output of the deterministic mana-curve analysis.
type ManaAnalysis struct {
	Format            string                `json:"format"`
	TotalCards        int                   `json:"total_cards"`
	AverageCMC        float64               `json:"average_cmc"`
	Distribution      []CMCBucket           `json:"distribution"`
	TypeDistribution  CardTypeDistribution  `json:"type_distribution"`
	LandCount         int                   `json:"land_count"`
	IdealLandCount    int                   `json:"ideal_land_count"`
	ManaProducerCount int                   `json:"mana_producer_count,omitempty"`
	CurrentTotalSources int                 `json:"current_total_sources,omitempty"`
	TargetTotalSources  int                 `json:"target_total_sources,omitempty"`
	TotalSourceGap      int                 `json:"total_source_gap,omitempty"`
	ManaScrewChance     float64             `json:"mana_screw_chance,omitempty"`
	ManaFloodChance     float64             `json:"mana_flood_chance,omitempty"`
	SweetSpotChance     float64             `json:"sweet_spot_chance,omitempty"`
	LandSampleDraws     int                 `json:"land_sample_draws,omitempty"`
	SweetSpotMinLands   int                 `json:"sweet_spot_min_lands,omitempty"`
	SweetSpotMaxLands   int                 `json:"sweet_spot_max_lands,omitempty"`
	ColorDistribution map[string]int        `json:"color_distribution"`
	// PipDistribution counts how many coloured pips appear across all non-land mana costs.
	// Keys are single-letter colour codes: W, U, B, R, G, C.
	// Example: 16 white pips means the deck demands T white sources early.
	PipDistribution   map[string]int        `json:"pip_distribution,omitempty"`
	SourceRequirements []ColorSourceRequirement `json:"source_requirements,omitempty"`
	Suggestions       []ManaCurveSuggestion `json:"suggestions"`
}

// DeckArchetype represents the detected play style of a deck.
type DeckArchetype string

const (
	ArchetypeAggro    DeckArchetype = "aggro"
	ArchetypeControl  DeckArchetype = "control"
	ArchetypeRamp     DeckArchetype = "ramp"
	ArchetypeMidrange DeckArchetype = "midrange"
	ArchetypeUnknown  DeckArchetype = "unknown"
)

// InteractionCategory describes a type of interactive spell.
type InteractionCategory string

const (
	InteractionRemoval    InteractionCategory = "removal"
	InteractionCounter    InteractionCategory = "counter"
	InteractionDraw       InteractionCategory = "draw"
	InteractionRamp       InteractionCategory = "ramp"
	InteractionProtection InteractionCategory = "protection"
	InteractionDiscard    InteractionCategory = "discard"
)

// InteractionBreakdown holds the count and weighted score for a category.
type InteractionBreakdown struct {
	Category InteractionCategory `json:"category"`
	Count    int                 `json:"count"`
	Weight   float64             `json:"weight"`
	Score    float64             `json:"score"`
	Ideal    int                 `json:"ideal"`
	Delta    int                 `json:"delta"` // positive = over, negative = under
}

// InteractionAnalysis is the output of the deterministic interaction-density analysis.
type InteractionAnalysis struct {
	Format      string                 `json:"format"`
	Archetype   string                 `json:"archetype"`
	TotalScore  float64                `json:"total_score"`
	Breakdowns  []InteractionBreakdown `json:"breakdowns"`
	Suggestions []string               `json:"suggestions"`
}

// AnalysisResult aggregates all deterministic analyses for a deck.
type AnalysisResult struct {
	DeckID      string              `json:"deck_id,omitempty"`
	Format      string              `json:"format"`
	Mana        ManaAnalysis        `json:"mana"`
	Interaction InteractionAnalysis `json:"interaction"`
	ScoreDetail *ScoreDetail        `json:"score_detail,omitempty" bson:"score_detail,omitempty"`
	LatencyMs   int64               `json:"latency_ms"`
}
