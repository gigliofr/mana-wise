package domain_test

import (
	"testing"
	"time"

	"github.com/manawise/api/domain"
)

func TestBusinessDayForTime_EuropeRome(t *testing.T) {
	now := time.Date(2026, 3, 15, 23, 15, 0, 0, time.UTC)
	got := domain.BusinessDayForTime(now, "Europe/Rome")
	if got != "2026-03-16" {
		t.Fatalf("expected Europe/Rome business day 2026-03-16, got %s", got)
	}
}

func TestBusinessDayForTime_InvalidTimezoneFallsBack(t *testing.T) {
	now := time.Date(2026, 3, 15, 12, 0, 0, 0, time.UTC)
	got := domain.BusinessDayForTime(now, "Not/AZone")
	if got == "" {
		t.Fatal("expected non-empty business day fallback")
	}
}

func TestUser_CanAnalyze_FreePlan(t *testing.T) {
	today := "2026-03-06"
	u := &domain.User{
		Plan:            domain.PlanFree,
		LastAnalysisDay: today,
		DailyAnalyses:   0,
	}

	if !u.CanAnalyze(today) {
		t.Error("user with 0 analyses today should be able to analyze")
	}

	u.DailyAnalyses = domain.FreeDailyLimit
	if u.CanAnalyze(today) {
		t.Error("user who reached limit should not be able to analyze")
	}
}

func TestUser_CanAnalyze_NewDay(t *testing.T) {
	u := &domain.User{
		Plan:            domain.PlanFree,
		LastAnalysisDay: "2026-03-05",
		DailyAnalyses:   domain.FreeDailyLimit,
	}
	if !u.CanAnalyze("2026-03-06") {
		t.Error("user on a new day should have reset quota")
	}
}

func TestUser_CanAnalyze_ProPlan(t *testing.T) {
	today := "2026-03-06"
	u := &domain.User{
		Plan:            domain.PlanPro,
		LastAnalysisDay: today,
		DailyAnalyses:   9999,
	}
	if !u.CanAnalyze(today) {
		t.Error("pro user should always be able to analyze")
	}
}

func TestUser_RemainingAnalyses(t *testing.T) {
	today := "2026-03-06"
	u := &domain.User{
		Plan:            domain.PlanFree,
		LastAnalysisDay: today,
		DailyAnalyses:   1,
	}
	if u.RemainingAnalyses(today) != domain.FreeDailyLimit-1 {
		t.Errorf("expected %d remaining, got %d", domain.FreeDailyLimit-1, u.RemainingAnalyses(today))
	}

	u.DailyAnalyses = domain.FreeDailyLimit + 10
	if u.RemainingAnalyses(today) != 0 {
		t.Error("remaining should not go below 0")
	}
}

func TestUser_RemainingAnalyses_ProUnlimited(t *testing.T) {
	u := &domain.User{Plan: domain.PlanPro}
	if u.RemainingAnalyses("2026-03-06") != -1 {
		t.Error("pro user should have -1 (unlimited) remaining")
	}
}

func TestCard_IsLegal(t *testing.T) {
	card := &domain.Card{
		Legalities: map[string]string{
			"modern":  "legal",
			"vintage": "restricted",
			"legacy":  "banned",
		},
	}
	if !card.IsLegal("modern") {
		t.Error("card should be legal in modern")
	}
	if card.IsLegal("legacy") {
		t.Error("banned card should not be legal")
	}
	if card.IsLegal("commander") {
		t.Error("card without entry should not be legal")
	}
}

func TestIsValidFormat(t *testing.T) {
	valids := []string{"modern", "commander", "standard", "legacy", "pioneer", "vintage", "pauper"}
	for _, f := range valids {
		if !domain.IsValidFormat(f) {
			t.Errorf("%q should be valid", f)
		}
	}
	if domain.IsValidFormat("freeform") {
		t.Error("freeform should not be valid")
	}
}

func TestNormalizeFormat(t *testing.T) {
	tests := map[string]string{
		"EDH":        "commander",
		"std":        "standard",
		"Pio":        "pioneer",
		"modern":     "modern",
		" legacy ":   "legacy",
		"unknownfmt": "unknownfmt",
	}
	for in, expected := range tests {
		got := domain.NormalizeFormat(in)
		if got != expected {
			t.Fatalf("NormalizeFormat(%q) = %q, expected %q", in, got, expected)
		}
	}
}
