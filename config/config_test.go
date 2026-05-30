package config

import (
	"reflect"
	"testing"
	"time"
)

func TestLoadAISettings_DefaultsAndOverrides(t *testing.T) {
	t.Setenv("MONGODB_URI", "mongodb://localhost:27017/manawise")
	t.Setenv("JWT_SECRET", "test-secret")
	t.Setenv("AI_MODE", "hybrid_prefer_external")
	t.Setenv("AI_INTERNAL_RULES_ENABLED", "true")
	t.Setenv("AI_FALLBACK_ON_STATUS", "429,503,504")
	t.Setenv("AI_FALLBACK_ON_TIMEOUT_MS", "7000")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if cfg.LLM.AIMode != "hybrid_prefer_external" {
		t.Fatalf("expected AI mode %q, got %q", "hybrid_prefer_external", cfg.LLM.AIMode)
	}
	if !cfg.LLM.InternalRulesEnabled {
		t.Fatal("expected internal rules to be enabled")
	}
	if !cfg.LLM.FallbackOnTimeout {
		t.Fatal("expected timeout fallback to be enabled")
	}
	if cfg.LLM.Timeout != 7*time.Second {
		t.Fatalf("expected 7s timeout, got %s", cfg.LLM.Timeout)
	}
	wantStatuses := []int{429, 503, 504}
	if !reflect.DeepEqual(cfg.LLM.FallbackOnStatus, wantStatuses) {
		t.Fatalf("expected fallback statuses %v, got %v", wantStatuses, cfg.LLM.FallbackOnStatus)
	}
}

func TestLoadAISettings_DisablesTimeoutFallbackWhenConfigured(t *testing.T) {
	t.Setenv("MONGODB_URI", "mongodb://localhost:27017/manawise")
	t.Setenv("JWT_SECRET", "test-secret")
	t.Setenv("AI_FALLBACK_ON_TIMEOUT_MS", "0")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load() returned error: %v", err)
	}

	if cfg.LLM.FallbackOnTimeout {
		t.Fatal("expected timeout fallback to be disabled")
	}
	if cfg.LLM.Timeout != 15*time.Second {
		t.Fatalf("expected default timeout to stay at 15s, got %s", cfg.LLM.Timeout)
	}
}
