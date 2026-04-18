package domain

import "strings"

// CommanderBracketConfig stores the rule sets used to classify a Commander deck.
type CommanderBracketConfig struct {
	Enabled                bool     `json:"enabled"`
	DecisiveCards          []string `json:"decisive_cards"`
	ComboCards             []string `json:"combo_cards"`
	ExtraTurnCards         []string `json:"extra_turn_cards"`
	MassLandDenialCards    []string `json:"mass_land_denial_cards"`
	TutorKeywords          []string `json:"tutor_keywords"`
	ExtraTurnKeywords      []string `json:"extra_turn_keywords"`
	MassLandDenialKeywords []string `json:"mass_land_denial_keywords"`
	ComboKeywords          []string `json:"combo_keywords"`
	FastManaKeywords       []string `json:"fast_mana_keywords"`
	Bracket1MaxSignals     int      `json:"bracket1_max_signals"`
	Bracket2MaxSignals     int      `json:"bracket2_max_signals"`
	Bracket3MaxDecisive    int      `json:"bracket3_max_decisive"`
	Bracket3MaxSignals     int      `json:"bracket3_max_signals"`
	Bracket4MaxSignals     int      `json:"bracket4_max_signals"`
	CedhTutorThreshold     int      `json:"cedh_tutor_threshold"`
	CedhComboThreshold     int      `json:"cedh_combo_threshold"`
	CedhFastManaThreshold  int      `json:"cedh_fast_mana_threshold"`
	CedhDecisiveThreshold  int      `json:"cedh_decisive_threshold"`
}

// DefaultCommanderBracketConfig returns a conservative, spec-oriented default configuration.
func DefaultCommanderBracketConfig() CommanderBracketConfig {
	return CommanderBracketConfig{
		Enabled:                true,
		DecisiveCards:          DefaultCommanderDecisiveCards(),
		ComboCards:             DefaultCommanderComboCards(),
		ExtraTurnCards:         DefaultCommanderExtraTurnCards(),
		MassLandDenialCards:    DefaultCommanderMassLandDenialCards(),
		TutorKeywords:          []string{"search your library for", "tutor", "demonic", "vampiric", "enlightened", "worldly", "mystical"},
		ExtraTurnKeywords:      []string{"take an extra turn after this one", "take an extra turn"},
		MassLandDenialKeywords: []string{"destroy all lands", "each player sacrifices all lands", "lands don't untap", "players can't play lands"},
		ComboKeywords:          nil,
		FastManaKeywords:       []string{"add two mana", "add three mana", "add four mana", "add {c}{c}", "treasure token", "mana crypt", "sol ring", "chrome mox", "mox", "lotus"},
		Bracket1MaxSignals:     0,
		Bracket2MaxSignals:     2,
		Bracket3MaxDecisive:    3,
		Bracket3MaxSignals:     4,
		Bracket4MaxSignals:     7,
		CedhTutorThreshold:     8,
		CedhComboThreshold:     2,
		CedhFastManaThreshold:  3,
		CedhDecisiveThreshold:  4,
	}
}

// DefaultCommanderComboCards returns a conservative list of cards strongly associated with fast two-card combo shells.
func DefaultCommanderComboCards() []string {
	return []string{
		"Thassa's Oracle",
		"Demonic Consultation",
		"Tainted Pact",
		"Underworld Breach",
		"Dramatic Reversal",
		"Dualcaster Mage",
		"Food Chain",
		"Heliod, Sun-Crowned",
		"Walking Ballista",
		"Sanguine Bond",
		"Exquisite Blood",
	}
}

// DefaultCommanderExtraTurnCards returns common extra-turn effects used for turn chains.
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

// DefaultCommanderDecisiveCards is the built-in decisive-card list used as a starting point.
func DefaultCommanderDecisiveCards() []string {
	return []string{
		"Ad Nauseam",
		"Bolas's Citadel",
		"Consultation",
		"Demonic Consultation",
		"Dockside Extortionist",
		"Dramatic Reversal",
		"Dualcaster Mage",
		"Food Chain",
		"Gaea's Cradle",
		"Hermit Druid",
		"Jeweled Lotus",
		"Kinnan, Bonder Prodigy",
		"Lion's Eye Diamond",
		"Mana Crypt",
		"Mana Vault",
		"Mox Amber",
		"Mox Diamond",
		"Mystic Remora",
		"Necropotence",
		"Notion Thief",
		"Oracle of Mul Daya",
		"Orcish Bowmasters",
		"Rhystic Study",
		"Seedborn Muse",
		"Sensei's Divining Top",
		"The One Ring",
		"Tainted Pact",
		"Thassa's Oracle",
		"Underworld Breach",
		"Vampiric Tutor",
		"Worldly Tutor",
		"Enlightened Tutor",
		"Mystical Tutor",
	}
}

// CommanderBracketSignals collects the concrete signals found while analyzing a deck.
type CommanderBracketSignals struct {
	DecisiveCards       []string `json:"decisive_cards,omitempty"`
	TutorCards          []string `json:"tutor_cards,omitempty"`
	ExtraTurnCards      []string `json:"extra_turn_cards,omitempty"`
	MassLandDenialCards []string `json:"mass_land_denial_cards,omitempty"`
	ComboCards          []string `json:"combo_cards,omitempty"`
	FastManaCards       []string `json:"fast_mana_cards,omitempty"`
	AverageCMC          float64  `json:"average_cmc,omitempty"`
	LandCount           int      `json:"land_count,omitempty"`
	NonLandCount        int      `json:"non_land_count,omitempty"`
}

// CommanderBracketAssessment is the output of bracket evaluation.
type CommanderBracketAssessment struct {
	Bracket int                     `json:"bracket"`
	Label   string                  `json:"label"`
	Reason  string                  `json:"reason"`
	Signals CommanderBracketSignals `json:"signals"`
	Notes   []string                `json:"notes,omitempty"`
}

// NormalizeCommanderCardName returns the canonical key used for lookups.
func NormalizeCommanderCardName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}
