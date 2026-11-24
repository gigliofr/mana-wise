package domain

import "strings"

// CommanderBracketConfig stores the rule sets used to classify a Commander deck.
//
// The classification follows the official Commander Brackets (beta) criteria as
// updated on 21 October 2025 and 9 February 2026:
//   - Game Changers: none in Brackets 1-2, up to 3 in Bracket 3, unlimited in 4-5
//   - mass land denial: only from Bracket 4
//   - chained extra turns: only from Bracket 4
//   - two-card infinite combos: none in Brackets 1-2, only late-game in Bracket 3
//   - tutors are no longer restricted (October 2025), so they are reported but do
//     not raise the bracket
//
// Bracket 1 (Exhibition) and 5 (cEDH) depend on the deck's intent, which a list
// alone cannot prove: the estimate never returns 1, and returns 5 only when the
// competitive-density thresholds below are all met.
type CommanderBracketConfig struct {
	Enabled bool `json:"enabled"`

	// GameChangers is the official Game Changers list. The default is the
	// February 2026 list (53 cards); it can be refreshed from Scryfall
	// (is:gamechanger) at runtime.
	GameChangers []string `json:"game_changers"`
	// TwoCardCombos lists known two-card infinite or game-winning combos.
	TwoCardCombos []TwoCardCombo `json:"two_card_combos"`
	// EarlyComboMaxManaValue: a combo whose pieces add up to at most this mana
	// value is considered able to fire in the early game (Bracket 4).
	EarlyComboMaxManaValue float64 `json:"early_combo_max_mana_value"`

	ExtraTurnCards         []string `json:"extra_turn_cards"`
	MassLandDenialCards    []string `json:"mass_land_denial_cards"`
	TutorKeywords          []string `json:"tutor_keywords"`
	ExtraTurnKeywords      []string `json:"extra_turn_keywords"`
	MassLandDenialKeywords []string `json:"mass_land_denial_keywords"`
	FastManaKeywords       []string `json:"fast_mana_keywords"`

	// Bracket3MaxGameChangers is the official cap for Bracket 3.
	Bracket3MaxGameChangers int `json:"bracket3_max_game_changers"`
	// Bracket3MaxExtraTurns: a few non-chained extra turns are fine up to
	// Bracket 3; more than this many are treated as a likely turn chain (Bracket 4).
	Bracket3MaxExtraTurns int `json:"bracket3_max_extra_turns"`

	// cEDH heuristics: all must be met to suggest Bracket 5.
	CedhGameChangerThreshold int `json:"cedh_game_changer_threshold"`
	CedhEarlyComboThreshold  int `json:"cedh_early_combo_threshold"`
	CedhFastManaThreshold    int `json:"cedh_fast_mana_threshold"`
	CedhTutorThreshold       int `json:"cedh_tutor_threshold"`
}

// TwoCardCombo is a pair of cards that together win the game or go infinite.
type TwoCardCombo struct {
	Cards  [2]string `json:"cards"`
	Result string    `json:"result"`
}

// DetectedCombo is a two-card combo found in a deck.
type DetectedCombo struct {
	Cards     []string `json:"cards"`
	Result    string   `json:"result"`
	ManaValue float64  `json:"mana_value"`
	Early     bool     `json:"early"`
}

// DefaultCommanderBracketConfig returns the default configuration.
func DefaultCommanderBracketConfig() CommanderBracketConfig {
	return CommanderBracketConfig{
		Enabled:                  true,
		GameChangers:             DefaultGameChangers(),
		TwoCardCombos:            DefaultTwoCardCombos(),
		EarlyComboMaxManaValue:   6,
		ExtraTurnCards:           DefaultCommanderExtraTurnCards(),
		MassLandDenialCards:      DefaultCommanderMassLandDenialCards(),
		TutorKeywords:            []string{"search your library for", "tutor"},
		ExtraTurnKeywords:        []string{"take an extra turn after this one", "take an extra turn"},
		MassLandDenialKeywords:   []string{"destroy all lands", "each player sacrifices all lands", "lands don't untap", "players can't play lands"},
		FastManaKeywords:         []string{"add two mana", "add three mana", "add four mana", "add {c}{c}", "mox", "lotus"},
		Bracket3MaxGameChangers:  3,
		Bracket3MaxExtraTurns:    2,
		CedhGameChangerThreshold: 8,
		CedhEarlyComboThreshold:  1,
		CedhFastManaThreshold:    5,
		CedhTutorThreshold:       6,
	}
}

// DefaultGameChangers returns the official Game Changers list as of the
// 9 February 2026 update (53 cards), cross-checked on 23 November 2026 against
// two independent copies of the list that are verified against Scryfall's
// is:gamechanger query.
func DefaultGameChangers() []string {
	return []string{
		// White
		"Drannith Magistrate", "Enlightened Tutor", "Farewell", "Humility",
		"Serra's Sanctum", "Smothering Tithe", "Teferi's Protection",
		// Blue
		"Consecrated Sphinx", "Cyclonic Rift", "Fierce Guardianship", "Force of Will",
		"Gifts Ungiven", "Intuition", "Mystical Tutor", "Narset, Parter of Veils",
		"Rhystic Study", "Thassa's Oracle",
		// Black
		"Ad Nauseam", "Bolas's Citadel", "Braids, Cabal Minion", "Demonic Tutor",
		"Imperial Seal", "Necropotence", "Opposition Agent", "Orcish Bowmasters",
		"Tergrid, God of Fright", "Vampiric Tutor",
		// Red
		"Gamble", "Jeska's Will", "Underworld Breach",
		// Green
		"Biorhythm", "Crop Rotation", "Gaea's Cradle", "Natural Order",
		"Seedborn Muse", "Survival of the Fittest", "Worldly Tutor",
		// Multicolored
		"Aura Shards", "Coalition Victory", "Grand Arbiter Augustin IV", "Notion Thief",
		// Colorless
		"Ancient Tomb", "Chrome Mox", "Field of the Dead", "Glacial Chasm",
		"Grim Monolith", "Lion's Eye Diamond", "Mana Vault", "Mishra's Workshop",
		"Mox Diamond", "Panoptic Mirror", "The One Ring", "The Tabernacle at Pendrell Vale",
	}
}

// DefaultTwoCardCombos returns a small curated set of well-known two-card
// combos. It is a starting point until a full combo database (e.g. Commander
// Spellbook) is integrated.
func DefaultTwoCardCombos() []TwoCardCombo {
	return []TwoCardCombo{
		{Cards: [2]string{"Thassa's Oracle", "Demonic Consultation"}, Result: "win"},
		{Cards: [2]string{"Thassa's Oracle", "Tainted Pact"}, Result: "win"},
		{Cards: [2]string{"Heliod, Sun-Crowned", "Walking Ballista"}, Result: "infinite damage"},
		{Cards: [2]string{"Devoted Druid", "Vizier of Remedies"}, Result: "infinite mana"},
		{Cards: [2]string{"Isochron Scepter", "Dramatic Reversal"}, Result: "infinite mana (with mana rocks)"},
		{Cards: [2]string{"Dualcaster Mage", "Twinflame"}, Result: "infinite creatures"},
		{Cards: [2]string{"Dualcaster Mage", "Heat Shimmer"}, Result: "infinite creatures"},
		{Cards: [2]string{"Food Chain", "Squee, the Immortal"}, Result: "infinite creature mana"},
		{Cards: [2]string{"Power Artifact", "Grim Monolith"}, Result: "infinite mana"},
		{Cards: [2]string{"Power Artifact", "Basalt Monolith"}, Result: "infinite mana"},
		{Cards: [2]string{"Earthcraft", "Squirrel Nest"}, Result: "infinite squirrels"},
		{Cards: [2]string{"Painter's Servant", "Grindstone"}, Result: "mill opponent's library"},
		{Cards: [2]string{"Splinter Twin", "Pestermite"}, Result: "infinite creatures"},
		{Cards: [2]string{"Splinter Twin", "Deceiver Exarch"}, Result: "infinite creatures"},
		{Cards: [2]string{"Kiki-Jiki, Mirror Breaker", "Pestermite"}, Result: "infinite creatures"},
		{Cards: [2]string{"Kiki-Jiki, Mirror Breaker", "Deceiver Exarch"}, Result: "infinite creatures"},
		{Cards: [2]string{"Kiki-Jiki, Mirror Breaker", "Zealous Conscripts"}, Result: "infinite creatures"},
		{Cards: [2]string{"Kiki-Jiki, Mirror Breaker", "Restoration Angel"}, Result: "infinite creatures"},
		{Cards: [2]string{"Niv-Mizzet, Parun", "Curiosity"}, Result: "infinite damage"},
		{Cards: [2]string{"Sanguine Bond", "Exquisite Blood"}, Result: "infinite life drain"},
		{Cards: [2]string{"Mikaeus, the Unhallowed", "Triskelion"}, Result: "infinite damage"},
		{Cards: [2]string{"Peregrine Drake", "Deadeye Navigator"}, Result: "infinite mana"},
	}
}

// DefaultCommanderExtraTurnCards returns common extra-turn effects.
func DefaultCommanderExtraTurnCards() []string {
	return []string{
		"Time Warp",
		"Temporal Manipulation",
		"Capture of Jingzhou",
		"Nexus of Fate",
		"Expropriate",
		"Temporal Mastery",
		"Walk the Aeons",
	}
}

// DefaultCommanderMassLandDenialCards returns common mass land denial cards.
func DefaultCommanderMassLandDenialCards() []string {
	return []string{
		"Armageddon",
		"Ravages of War",
		"Jokulhaups",
		"Cataclysm",
		"Ruination",
		"Winter Orb",
		"Static Orb",
	}
}

// CommanderBracketSignals collects the concrete signals found while analyzing a deck.
type CommanderBracketSignals struct {
	GameChangerCards    []string        `json:"game_changer_cards,omitempty"`
	TwoCardCombos       []DetectedCombo `json:"two_card_combos,omitempty"`
	ExtraTurnCards      []string        `json:"extra_turn_cards,omitempty"`
	MassLandDenialCards []string        `json:"mass_land_denial_cards,omitempty"`
	BannedCards         []string        `json:"banned_cards,omitempty"`
	// Informational only: tutors and fast mana do not raise the bracket on
	// their own, but feed the cEDH heuristic.
	TutorCards    []string `json:"tutor_cards,omitempty"`
	FastManaCards []string `json:"fast_mana_cards,omitempty"`
	AverageCMC    float64  `json:"average_cmc,omitempty"`
	LandCount     int      `json:"land_count,omitempty"`
	NonLandCount  int      `json:"non_land_count,omitempty"`
}

// CommanderBracketAssessment is the output of bracket evaluation.
// Bracket is the minimum bracket the list qualifies for.
type CommanderBracketAssessment struct {
	Bracket int                     `json:"bracket"`
	Label   string                  `json:"label"`
	Reason  string                  `json:"reason"`
	Signals CommanderBracketSignals `json:"signals"`
	Notes   []string                `json:"notes,omitempty"`
	// Legal is false when the deck contains cards banned in Commander.
	Legal bool `json:"legal"`
}

// CommanderBracketLabel returns the official name of a bracket.
func CommanderBracketLabel(bracket int) string {
	switch bracket {
	case 1:
		return "Exhibition"
	case 2:
		return "Core"
	case 3:
		return "Upgraded"
	case 4:
		return "Optimized"
	case 5:
		return "cEDH"
	default:
		return "Unknown"
	}
}

// NormalizeCommanderCardName returns the canonical key used for lookups.
func NormalizeCommanderCardName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}
