package usecase_test

import (
	"testing"

	"github.com/manawise/api/domain"
	"github.com/manawise/api/usecase"
)

func makeCard(id string, cmc float64, typeL string, colors ...string) *domain.Card {
	return &domain.Card{
		ID:       id,
		Name:     id,
		CMC:      cmc,
		TypeLine: typeL,
		Colors:   colors,
	}
}

func makeLand(id, name string, identity ...string) *domain.Card {
	return &domain.Card{
		ID:            id,
		Name:          name,
		TypeLine:      "Land",
		ColorIdentity: identity,
	}
}

func TestAnalyzeManaCurve_BasicModern(t *testing.T) {
	// A typical 60-card modern-ish deck: 24 lands + 36 spells.
	cards := []*domain.Card{}
	qtys := map[string]int{}

	// Lands
	for i := 0; i < 24; i++ {
		id := "land-" + string(rune('A'+i))
		cards = append(cards, makeCard(id, 0, "Basic Land"))
		qtys[id] = 1
	}
	// 1-drop spells ×12
	for i := 0; i < 12; i++ {
		id := "one-" + string(rune('A'+i))
		cards = append(cards, makeCard(id, 1, "Instant", "R"))
		qtys[id] = 1
	}
	// 2-drop spells ×12
	for i := 0; i < 12; i++ {
		id := "two-" + string(rune('A'+i))
		cards = append(cards, makeCard(id, 2, "Creature", "W"))
		qtys[id] = 1
	}
	// 3-drop ×8, 4-drop ×4
	for i := 0; i < 8; i++ {
		id := "three-" + string(rune('A'+i))
		cards = append(cards, makeCard(id, 3, "Sorcery", "U"))
		qtys[id] = 1
	}
	for i := 0; i < 4; i++ {
		id := "four-" + string(rune('A'+i))
		cards = append(cards, makeCard(id, 4, "Creature", "G"))
		qtys[id] = 1
	}

	result := usecase.AnalyzeManaCurve(cards, qtys, "modern")

	if result.Format != "modern" {
		t.Errorf("expected format 'modern', got %q", result.Format)
	}
	if result.LandCount != 24 {
		t.Errorf("expected 24 lands, got %d", result.LandCount)
	}
	if result.TotalCards != 60 {
		t.Errorf("expected 60 total cards, got %d", result.TotalCards)
	}
	if result.AverageCMC <= 0 {
		t.Error("average CMC should be positive")
	}
	if len(result.Distribution) != 7 {
		t.Errorf("expected 7 CMC buckets (0–6+), got %d", len(result.Distribution))
	}
}

func TestAnalyzeManaCurve_IdealLandCount(t *testing.T) {
	cards := []*domain.Card{makeCard("l1", 0, "Basic Land")}
	qtys := map[string]int{"l1": 1}

	tests := []struct {
		format   string
		wantLow  int
		wantHigh int
	}{
		{"modern", 21, 25},
		{"commander", 35, 40},
		{"standard", 22, 26},
	}

	for _, tt := range tests {
		result := usecase.AnalyzeManaCurve(cards, qtys, tt.format)
		if result.IdealLandCount < tt.wantLow || result.IdealLandCount > tt.wantHigh {
			t.Errorf("format=%s: ideal land count %d not in [%d,%d]",
				tt.format, result.IdealLandCount, tt.wantLow, tt.wantHigh)
		}
	}
}

func TestAnalyzeManaCurve_LandSuggestion(t *testing.T) {
	// Deck with too few lands (only 10 for a 60-card modern deck).
	cards := []*domain.Card{}
	qtys := map[string]int{}
	for i := 0; i < 10; i++ {
		id := "land-" + string(rune('A'+i))
		cards = append(cards, makeCard(id, 0, "Basic Land"))
		qtys[id] = 1
	}
	for i := 0; i < 50; i++ {
		id := "spell-" + string(rune('A'+i%26)) + string(rune('a'+i/26))
		cards = append(cards, makeCard(id, 2, "Instant", "U"))
		qtys[id] = 1
	}

	result := usecase.AnalyzeManaCurve(cards, qtys, "modern")
	if len(result.Suggestions) == 0 {
		t.Error("expected at least one suggestion for a deck with too few lands")
	}
}

func TestAnalyzeManaCurve_UnknownFormatFallsBack(t *testing.T) {
	cards := []*domain.Card{makeCard("land", 0, "Basic Land")}
	qtys := map[string]int{"land": 1}
	result := usecase.AnalyzeManaCurve(cards, qtys, "freeform")
	// Should not panic and should return some ideal.
	if result.IdealLandCount == 0 {
		t.Error("expected non-zero ideal land count even for unknown format")
	}
}

func TestAnalyzeManaCurve_MDFCSpellLandCountedAsLand(t *testing.T) {
	mdfc := &domain.Card{
		ID:            "mdfc-1",
		Name:          "Shatterskull Smashing // Shatterskull, the Hammer Pass",
		CMC:           3,
		TypeLine:      "Sorcery",
		ManaCost:      "{X}{R}{R}",
		ColorIdentity: []string{"R"},
		Faces: []domain.CardFace{
			{Name: "Shatterskull Smashing", TypeLine: "Sorcery", ManaCost: "{X}{R}{R}", Colors: []string{"R"}},
			{Name: "Shatterskull, the Hammer Pass", TypeLine: "Land"},
		},
	}
	bolt := &domain.Card{ID: "bolt", Name: "Lightning Bolt", CMC: 1, TypeLine: "Instant", ManaCost: "{R}", Colors: []string{"R"}}

	result := usecase.AnalyzeManaCurve([]*domain.Card{mdfc, bolt}, map[string]int{"mdfc-1": 4, "bolt": 4}, "modern")

	if result.LandCount != 4 {
		t.Fatalf("expected MDFC copies to count as lands, got %d", result.LandCount)
	}
	if result.TotalCards != 8 {
		t.Fatalf("expected total cards 8, got %d", result.TotalCards)
	}
	if result.Distribution[3].Count != 0 {
		t.Fatalf("expected CMC 3 bucket to ignore MDFC land face, got %d", result.Distribution[3].Count)
	}
}

func TestAnalyzeManaCurve_SourceRequirementsExposeGap(t *testing.T) {
	cards := []*domain.Card{
		makeLand("island-1", "Island", "U"),
		makeLand("island-2", "Island", "U"),
		makeLand("island-3", "Island", "U"),
		makeLand("island-4", "Island", "U"),
		{ID: "cryptic", Name: "Cryptic Command", CMC: 4, TypeLine: "Instant", ManaCost: "{1}{U}{U}{U}", Colors: []string{"U"}},
	}
	qtys := map[string]int{"island-1": 1, "island-2": 1, "island-3": 1, "island-4": 1, "cryptic": 4}

	result := usecase.AnalyzeManaCurve(cards, qtys, "modern")

	if len(result.SourceRequirements) != 1 {
		t.Fatalf("expected single aggregate source requirement row, got %d", len(result.SourceRequirements))
	}
	total := result.SourceRequirements[0]
	if total.Color != "TOTAL" {
		t.Fatalf("expected TOTAL source requirement row, got %q", total.Color)
	}
	if total.Required <= total.Current {
		t.Fatalf("expected positive aggregate gap, got current=%d required=%d", total.Current, total.Required)
	}
}

func TestAnalyzeManaCurve_CountsItalianAndNonBasicLands(t *testing.T) {
	cards := []*domain.Card{
		{ID: "forest", Name: "Foresta", TypeLine: "Terra Base - Foresta"},
		{ID: "swamp", Name: "Palude", TypeLine: "Terra Base - Palude"},
		{ID: "gallery", Name: "Galleria di Fuga", TypeLine: "Terra"},
		{ID: "wilds", Name: "Terre Selvagge in Evoluzione", TypeLine: "Terra"},
		{ID: "elf", Name: "Elfi di Llanowar", CMC: 1, TypeLine: "Creatura - Elfo Druido", ManaCost: "{G}"},
	}

	q := map[string]int{
		"forest":  12,
		"swamp":  4,
		"gallery": 4,
		"wilds":  4,
		"elf":    4,
	}

	result := usecase.AnalyzeManaCurve(cards, q, "standard")

	if result.LandCount != 24 {
		t.Fatalf("expected 24 lands (12+4+4+4), got %d", result.LandCount)
	}
	if result.TotalCards != 28 {
		t.Fatalf("expected total cards 28, got %d", result.TotalCards)
	}
}

func TestAnalyzeManaCurve_FetchLandsCountAsFlexibleSources(t *testing.T) {
	cards := []*domain.Card{
		{ID: "swamp", Name: "Swamp", TypeLine: "Basic Land - Swamp"},
		{ID: "forest", Name: "Forest", TypeLine: "Basic Land - Forest"},
		{
			ID:         "wilds",
			Name:       "Evolving Wilds",
			TypeLine:   "Land",
			OracleText: "{T}, Sacrifice Evolving Wilds: Search your library for a basic land card, put it onto the battlefield tapped, then shuffle.",
		},
		{
			ID:       "bgspell",
			Name:     "BG Spell",
			CMC:      2,
			TypeLine: "Creature",
			ManaCost: "{B}{G}",
		},
	}

	q := map[string]int{
		"swamp":  4,
		"forest": 8,
		"wilds":  4,
		"bgspell": 4,
	}

	result := usecase.AnalyzeManaCurve(cards, q, "standard")

	if result.LandCount != 16 {
		t.Fatalf("expected 16 lands counted, got %d", result.LandCount)
	}
	if result.CurrentTotalSources != result.LandCount {
		t.Fatalf("expected current sources to match lands when no non-land producers exist, got current=%d lands=%d", result.CurrentTotalSources, result.LandCount)
	}
}

func TestAnalyzeManaCurve_IncludesManaProducingCreaturesInCurrentSources(t *testing.T) {
	cards := []*domain.Card{
		{ID: "forest", Name: "Forest", TypeLine: "Basic Land - Forest"},
		{ID: "swamp", Name: "Swamp", TypeLine: "Basic Land - Swamp"},
		{
			ID:         "llanowar",
			Name:       "Llanowar Elves",
			TypeLine:   "Creature — Elf Druid",
			CMC:        1,
			ManaCost:   "{G}",
			OracleText: "{T}: Add {G}.",
		},
	}

	q := map[string]int{
		"forest":   10,
		"swamp":    10,
		"llanowar": 4,
	}

	result := usecase.AnalyzeManaCurve(cards, q, "standard")

	if result.LandCount != 20 {
		t.Fatalf("expected 20 lands, got %d", result.LandCount)
	}
	if result.ManaProducerCount != 4 {
		t.Fatalf("expected 4 mana-producing creatures, got %d", result.ManaProducerCount)
	}
	if result.CurrentTotalSources != 24 {
		t.Fatalf("expected current total sources 24, got %d", result.CurrentTotalSources)
	}
}

func TestAnalyzeManaCurve_ComputesTypeDistribution(t *testing.T) {
	cards := []*domain.Card{
		{ID: "land", Name: "Mountain", TypeLine: "Basic Land - Mountain"},
		{ID: "creature", Name: "Goblin Guide", CMC: 1, TypeLine: "Creature - Goblin"},
		{ID: "instant", Name: "Lightning Bolt", CMC: 1, TypeLine: "Instant"},
		{ID: "sorcery", Name: "Lava Spike", CMC: 1, TypeLine: "Sorcery"},
		{ID: "artifact", Name: "Skullclamp", CMC: 1, TypeLine: "Artifact"},
		{ID: "planeswalker", Name: "Chandra", CMC: 4, TypeLine: "Planeswalker - Chandra"},
	}

	q := map[string]int{
		"land":        2,
		"creature":    4,
		"instant":     4,
		"sorcery":     2,
		"artifact":    1,
		"planeswalker": 1,
	}

	result := usecase.AnalyzeManaCurve(cards, q, "modern")

	if result.TypeDistribution.Creature != 4 {
		t.Fatalf("expected creature count 4, got %d", result.TypeDistribution.Creature)
	}
	if result.TypeDistribution.Spell != 6 {
		t.Fatalf("expected spell count 6 (instant+sorcery), got %d", result.TypeDistribution.Spell)
	}
	if result.TypeDistribution.EnchantArtifact != 1 {
		t.Fatalf("expected enchant/artifact count 1, got %d", result.TypeDistribution.EnchantArtifact)
	}
	if result.TypeDistribution.Planeswalker != 1 {
		t.Fatalf("expected planeswalker count 1, got %d", result.TypeDistribution.Planeswalker)
	}
}

func TestAnalyzeManaCurve_LandConsistencyPercentagesBounded(t *testing.T) {
	cards := []*domain.Card{
		{ID: "forest", Name: "Forest", TypeLine: "Basic Land - Forest"},
		{ID: "spell", Name: "Spell", TypeLine: "Instant", CMC: 2, ManaCost: "{1}{G}"},
	}

	q := map[string]int{
		"forest": 24,
		"spell":  36,
	}

	result := usecase.AnalyzeManaCurve(cards, q, "modern")

	if result.LandSampleDraws != 12 {
		t.Fatalf("expected sample draws 12, got %d", result.LandSampleDraws)
	}
	if result.SweetSpotMinLands <= 0 || result.SweetSpotMaxLands <= 0 {
		t.Fatalf("expected positive sweet-spot bounds, got min=%d max=%d", result.SweetSpotMinLands, result.SweetSpotMaxLands)
	}
	if result.SweetSpotMinLands > result.SweetSpotMaxLands {
		t.Fatalf("invalid sweet-spot bounds: min=%d max=%d", result.SweetSpotMinLands, result.SweetSpotMaxLands)
	}

	total := result.ManaScrewChance + result.ManaFloodChance + result.SweetSpotChance
	if total < 99.0 || total > 101.0 {
		t.Fatalf("expected consistency percentages around 100, got %.2f", total)
	}
}

func TestAnalyzeManaCurve_LowLandDeckShowsHigherScrewThanFlood(t *testing.T) {
	cards := []*domain.Card{
		{ID: "land", Name: "Island", TypeLine: "Basic Land - Island"},
		{ID: "spell", Name: "Spell", TypeLine: "Instant", CMC: 2, ManaCost: "{1}{U}"},
	}

	q := map[string]int{
		"land": 16,
		"spell": 44,
	}

	result := usecase.AnalyzeManaCurve(cards, q, "modern")
	if result.ManaScrewChance <= result.ManaFloodChance {
		t.Fatalf("expected screw risk > flood risk for low-land deck, got screw=%.1f flood=%.1f", result.ManaScrewChance, result.ManaFloodChance)
	}
}
