package domain

// ErrorCode represents standardized error codes.
type ErrorCode string

const (
	ErrInvalidFormat      ErrorCode = "ERR_INVALID_FORMAT"
	ErrCardNotFound       ErrorCode = "ERR_CARD_NOT_FOUND"
	ErrDeckNotFound       ErrorCode = "ERR_DECK_NOT_FOUND"
	ErrUnauthorized       ErrorCode = "ERR_UNAUTHORIZED"
	ErrRateLimited        ErrorCode = "ERR_RATE_LIMITED"
	ErrInternalError      ErrorCode = "ERR_INTERNAL_ERROR"
	ErrTimeout            ErrorCode = "ERR_TIMEOUT"
	ErrValidation         ErrorCode = "ERR_VALIDATION"
	ErrUserNotFound       ErrorCode = "ERR_USER_NOT_FOUND"
	ErrInvalidRequest     ErrorCode = "ERR_INVALID_REQUEST"
	ErrCircuitBreakerOpen ErrorCode = "ERR_CIRCUIT_BREAKER_OPEN"
	ErrProviderUnavailable ErrorCode = "ERR_PROVIDER_UNAVAILABLE"
)

// APIError is a standard API error response.
type APIError struct {
	Error   string                 `json:"error"`
	Code    ErrorCode              `json:"code"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// NewAPIError creates a new API error.
func NewAPIError(message string, code ErrorCode) *APIError {
	return &APIError{
		Error:   message,
		Code:    code,
		Details: nil,
	}
}

// WithDetails adds a detail key-value pair to an API error.
func (ae *APIError) WithDetails(key string, value interface{}) *APIError {
	if ae.Details == nil {
		ae.Details = make(map[string]interface{})
	}
	ae.Details[key] = value
	return ae
}
