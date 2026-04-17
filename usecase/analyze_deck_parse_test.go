package usecase

import "testing"

func TestParseDecklist_MTGAArenaItalianFormat(t *testing.T) {
	raw := `Mazzo
4 Elfi di Llanowar (FDN) 227
12 Foresta (EOE) 276
4 Chocobo di Sazh (FIN) 200
2 Sazh Katzroy (FIN) 199`

	entries, err := parseDecklist(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 4 {
		t.Fatalf("expected 4 entries, got %d", len(entries))
	}

	if entries[0].qty != 4 || entries[0].name != "Elfi di Llanowar" {
		t.Fatalf("unexpected first entry: %+v", entries[0])
	}
	if entries[0].setCode != "fdn" || entries[0].collectorNumber != "227" {
		t.Fatalf("unexpected first entry set/collector: %+v", entries[0])
	}
	if entries[1].qty != 12 || entries[1].name != "Foresta" {
		t.Fatalf("unexpected second entry: %+v", entries[1])
	}
	if entries[1].setCode != "eoe" || entries[1].collectorNumber != "276" {
		t.Fatalf("unexpected second entry set/collector: %+v", entries[1])
	}
}

func TestParseDecklist_EnglishDeckHeaderAndClassicLines(t *testing.T) {
	raw := `Deck
4 Lightning Bolt
2 Monastery Swiftspear`

	entries, err := parseDecklist(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
	if entries[0].name != "Lightning Bolt" || entries[1].name != "Monastery Swiftspear" {
		t.Fatalf("unexpected names: %+v", entries)
	}
}

func TestSanitizeCardName(t *testing.T) {
	tests := map[string]string{
		"Elfi di Llanowar (FDN) 227": "Elfi di Llanowar",
		"Bedevil (fic) [Removal]":     "Bedevil",
		"Terra, Herald of Hope (fic) [Commander{top}]": "Terra, Herald of Hope",
		"Lightning Bolt":             "Lightning Bolt",
		"":                           "",
	}

	for in, expected := range tests {
		got := sanitizeCardName(in)
		if got != expected {
			t.Fatalf("sanitizeCardName(%q) = %q, expected %q", in, got, expected)
		}
	}
}

func TestParseDecklist_ArchidektAnnotatedLines(t *testing.T) {
	raw := `1 Bedevil (fic) [Removal]
3 Mountain (sos) 278
1 Terra, Herald of Hope (fic) [Commander{top}]`

	entries, err := parseDecklist(raw)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 3 {
		t.Fatalf("expected 3 entries, got %d", len(entries))
	}

	if entries[0].name != "Bedevil" {
		t.Fatalf("expected normalized name 'Bedevil', got %q", entries[0].name)
	}
	if entries[1].name != "Mountain" || entries[1].setCode != "sos" || entries[1].collectorNumber != "278" {
		t.Fatalf("unexpected second entry: %+v", entries[1])
	}
	if entries[2].name != "Terra, Herald of Hope" {
		t.Fatalf("expected normalized commander name, got %q", entries[2].name)
	}
}
