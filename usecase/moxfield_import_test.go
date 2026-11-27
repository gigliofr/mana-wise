package usecase

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/gigliofr/mana-wise/domain"
	"github.com/gigliofr/mana-wise/infrastructure/scryfall"
)

func TestParseDecklist_MoxfieldExportLines(t *testing.T) {
	raw := strings.Join([]string{
		"1 The First Sliver (J21) 691",
		"1 Cloudshredder Sliver (SLD) 660 *F*",
		"1 Constricting Sliver (PLST) M15-7",
		"1 Thrumming Hivepool (PEOE) 247p *F*",
		"1 Toxin Sliver (PSAL) D25",
		"1 Ready / Willing (DGM) 132",
		"1 Fire // Ice",
		"14 Forest (M20) 279",
	}, "\n")
	entries, err := parseDecklist(raw)
	if err != nil {
		t.Fatal(err)
	}
	want := []struct{ name, set, num string }{
		{"The First Sliver", "j21", "691"},
		{"Cloudshredder Sliver", "sld", "660"},
		{"Constricting Sliver", "plst", "m15-7"},
		{"Thrumming Hivepool", "peoe", "247p"},
		{"Toxin Sliver", "psal", "d25"},
		{"Ready // Willing", "dgm", "132"},
		{"Fire // Ice", "", ""},
		{"Forest", "m20", "279"},
	}
	if len(entries) != len(want) {
		t.Fatalf("expected %d entries, got %d: %+v", len(want), len(entries), entries)
	}
	for i, w := range want {
		e := entries[i]
		if e.name != w.name || e.setCode != w.set || e.collectorNumber != w.num {
			t.Errorf("line %d: got %q (%s/%s), want %q (%s/%s)", i+1, e.name, e.setCode, e.collectorNumber, w.name, w.set, w.num)
		}
	}
	if entries[7].qty != 14 {
		t.Errorf("Forest qty = %d, want 14", entries[7].qty)
	}
}

func TestCanBeCommander(t *testing.T) {
	cases := []struct {
		card *domain.Card
		want bool
	}{
		{&domain.Card{TypeLine: "Legendary Creature — Sliver"}, true},
		{&domain.Card{TypeLine: "Legendary Planeswalker — Teferi", OracleText: "Teferi can be your commander."}, true},
		{&domain.Card{TypeLine: "Creature — Sliver"}, false},
		{&domain.Card{TypeLine: "Legendary Artifact"}, false},
		{nil, false},
	}
	for i, c := range cases {
		if got := CanBeCommander(c.card); got != c.want {
			t.Errorf("case %d: CanBeCommander=%v, want %v", i, got, c.want)
		}
	}
}

// A Moxfield Commander export has no "Commander" header: the first line is the
// commander and must be recognised from the card type.
func TestAnalyzeDeckExecute_InfersCommanderFromFirstLine(t *testing.T) {
	byName := map[string]*domain.Card{
		"The First Sliver": {ID: "first", Name: "The First Sliver", CMC: 5, TypeLine: "Legendary Creature — Sliver",
			Colors: []string{"W", "U", "B", "R", "G"}, Legalities: map[string]string{"commander": "legal"}},
		"Forest": {ID: "forest", Name: "Forest", TypeLine: "Basic Land — Forest", Legalities: map[string]string{"commander": "legal"}},
	}
	lines := []string{"1 The First Sliver (J21) 691"}
	for i := 0; i < 61; i++ {
		name := fmt.Sprintf("Test Sliver %02d", i)
		byName[name] = &domain.Card{ID: fmt.Sprintf("s%02d", i), Name: name, CMC: 3, TypeLine: "Creature — Sliver",
			Legalities: map[string]string{"commander": "legal"}}
		lines = append(lines, fmt.Sprintf("1 %s (M14) %d *F*", name, i+1))
	}
	lines = append(lines, "38 Forest (M20) 279")

	repo := &fakeCardRepo{byName: byName}
	uc := NewAnalyzeDeckUseCase(&fakeCardFetcher{exact: map[string]*scryfall.ScryfallCard{}}, repo, 2)
	resp, err := uc.Execute(context.Background(), AnalyzeDeckRequest{Decklist: strings.Join(lines, "\n"), Format: "commander"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Commander == nil || len(resp.Commander.Cards) != 1 || resp.Commander.Cards[0].Name != "The First Sliver" {
		t.Fatalf("expected The First Sliver as commander, got %+v", resp.Commander)
	}
	if len(resp.Warnings) != 0 {
		t.Fatalf("all cards should resolve, warnings: %v", resp.Warnings)
	}
}

func TestAnalyzeDeckExecute_DoesNotInferNonLegendaryCommander(t *testing.T) {
	byName := map[string]*domain.Card{
		"Lightning Bolt": {ID: "bolt", Name: "Lightning Bolt", CMC: 1, TypeLine: "Instant", Legalities: map[string]string{"commander": "legal"}},
		"Mountain":       {ID: "mountain", Name: "Mountain", TypeLine: "Basic Land — Mountain", Legalities: map[string]string{"commander": "legal"}},
	}
	repo := &fakeCardRepo{byName: byName}
	uc := NewAnalyzeDeckUseCase(&fakeCardFetcher{exact: map[string]*scryfall.ScryfallCard{}}, repo, 2)
	resp, err := uc.Execute(context.Background(), AnalyzeDeckRequest{Decklist: "1 Lightning Bolt\n99 Mountain", Format: "commander"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.Commander != nil && len(resp.Commander.Cards) > 0 {
		t.Fatalf("an instant must not become the commander, got %+v", resp.Commander.Cards)
	}
}
