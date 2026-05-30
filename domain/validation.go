package domain

// ValidationError represents a single validation issue.
type ValidationError struct {
	Field   string `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Validator is a function that validates input and returns errors.
type Validator func() []ValidationError

// IsNonEmpty validates a non-empty string.
func IsNonEmpty(field, value string) []ValidationError {
	if value == "" {
		return []ValidationError{{
			Field:   field,
			Code:    "ERR_REQUIRED",
			Message: field + " is required",
		}}
	}
	return nil
}

// ValidateFormat validates a format string against known formats.
func ValidateFormat(field, value string) []ValidationError {
	validFormats := map[string]bool{
		"standard":   true,
		"modern":     true,
		"pioneer":    true,
		"legacy":     true,
		"vintage":    true,
		"commander":  true,
		"pauper":     true,
	}
	if !validFormats[value] {
		return []ValidationError{{
			Field:   field,
			Code:    "ERR_UNKNOWN_FORMAT",
			Message: "format '" + value + "' is not supported",
		}}
	}
	return nil
}

// IsValidPlan validates a user plan.
func IsValidPlan(field, value string) []ValidationError {
	validPlans := map[string]bool{
		"free": true,
		"pro":  true,
	}
	if !validPlans[value] {
		return []ValidationError{{
			Field:   field,
			Code:    "ERR_INVALID_PLAN",
			Message: "plan must be 'free' or 'pro'",
		}}
	}
	return nil
}

// Combine aggregates multiple validators.
func Combine(validators ...func() []ValidationError) []ValidationError {
	var errs []ValidationError
	for _, v := range validators {
		errs = append(errs, v()...)
	}
	return errs
}
