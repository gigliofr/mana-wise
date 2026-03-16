package main

import "testing"

func TestParseAnalyzeCommand(t *testing.T) {
	format, decklist := parseAnalyzeCommand("!analizza modern\n4 Lightning Bolt\n4 Goblin Guide", "commander")
	if format != "modern" {
		t.Fatalf("expected modern, got %q", format)
	}
	if decklist == "" {
		t.Fatal("expected decklist")
	}
}

func TestParseAnalyzeCommand_DefaultFormat(t *testing.T) {
	format, _ := parseAnalyzeCommand("!analizza\n1 Sol Ring", "commander")
	if format != "commander" {
		t.Fatalf("expected commander, got %q", format)
	}
}

func TestCommandRemainder(t *testing.T) {
	if got := commandRemainder("!prezzo Black Lotus", "!prezzo"); got != "Black Lotus" {
		t.Fatalf("unexpected remainder: %q", got)
	}
	if got := commandRemainder("!Sinergie Rhystic Study", "!sinergie"); got != "Rhystic Study" {
		t.Fatalf("unexpected remainder with case-insensitive prefix: %q", got)
	}
}

func TestFormatPct(t *testing.T) {
	if got := formatPct(nil); got != "n/a" {
		t.Fatalf("expected n/a, got %q", got)
	}
	v := 12.3456
	if got := formatPct(&v); got != "12.35%" {
		t.Fatalf("unexpected formatted pct: %q", got)
	}
}
