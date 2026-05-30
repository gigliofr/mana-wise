package domain

import (
	"testing"
)

func TestNewAPIError(t *testing.T) {
	err := NewAPIError("test error", ErrInvalidFormat)
	if err.Error != "test error" {
		t.Fatalf("expected 'test error', got %s", err.Error)
	}
	if err.Code != ErrInvalidFormat {
		t.Fatalf("expected ErrInvalidFormat, got %s", err.Code)
	}
}

func TestAPIError_WithDetails(t *testing.T) {
	err := NewAPIError("test error", ErrInvalidFormat).
		WithDetails("field", "format").
		WithDetails("provided", "invalid_value")

	if len(err.Details) != 2 {
		t.Fatalf("expected 2 details, got %d", len(err.Details))
	}
	if err.Details["field"] != "format" {
		t.Fatalf("expected 'format' for field, got %s", err.Details["field"])
	}
}

func TestAPIError_ErrorCodes(t *testing.T) {
	codes := []ErrorCode{
		ErrInvalidFormat,
		ErrCardNotFound,
		ErrDeckNotFound,
		ErrUnauthorized,
		ErrRateLimited,
		ErrTimeout,
		ErrInternalError,
		ErrInvalidRequest,
	}

	for _, code := range codes {
		if code == "" {
			t.Fatalf("error code should not be empty")
		}
	}
}
