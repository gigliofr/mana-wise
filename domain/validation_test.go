package domain

import (
	"testing"
)

func TestIsNonEmpty_WithValue(t *testing.T) {
	errs := IsNonEmpty("name", "John")
	if len(errs) > 0 {
		t.Fatalf("expected no errors, got %d", len(errs))
	}
}

func TestIsNonEmpty_EmptyValue(t *testing.T) {
	errs := IsNonEmpty("name", "")
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errs))
	}
	if errs[0].Code != "ERR_REQUIRED" {
		t.Fatalf("expected ERR_REQUIRED, got %s", errs[0].Code)
	}
}

func TestValidateFormat_ValidFormat(t *testing.T) {
	errs := ValidateFormat("format", "modern")
	if len(errs) > 0 {
		t.Fatalf("expected no errors, got %d", len(errs))
	}
}

func TestValidateFormat_InvalidFormat(t *testing.T) {
	errs := ValidateFormat("format", "invalid_format")
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d", len(errs))
	}
	if errs[0].Code != "ERR_UNKNOWN_FORMAT" {
		t.Fatalf("expected ERR_UNKNOWN_FORMAT, got %s", errs[0].Code)
	}
}

func TestIsValidPlan_ValidPlan(t *testing.T) {
	for _, plan := range []string{"free", "pro"} {
		errs := IsValidPlan("plan", plan)
		if len(errs) > 0 {
			t.Fatalf("expected no errors for plan %s, got %d", plan, len(errs))
		}
	}
}

func TestCombine_MultipleValidators(t *testing.T) {
	errs := Combine(
		func() []ValidationError { return IsNonEmpty("field1", "") },
		func() []ValidationError { return ValidateFormat("format", "invalid") },
	)

	if len(errs) != 2 {
		t.Fatalf("expected 2 errors, got %d", len(errs))
	}
}
