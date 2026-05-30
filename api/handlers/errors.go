package handlers

import (
    "encoding/json"
    "net/http"
    "strings"

    "github.com/gigliofr/mana-wise/domain"
)

// WriteAPIErrorFromMsg writes a standardized API error response, inferring
// a domain.ErrorCode from HTTP status and message for backward compatibility.
func WriteAPIErrorFromMsg(w http.ResponseWriter, msg string, status int) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)

    // Infer domain ErrorCode from status and message similar to legacy behavior.
    var errCode domain.ErrorCode
    switch status {
    case http.StatusBadRequest:
        errCode = domain.ErrInvalidRequest
    case http.StatusUnauthorized:
        errCode = domain.ErrUnauthorized
    case http.StatusForbidden:
        errCode = domain.ErrUnauthorized
    case http.StatusNotFound:
        if strings.Contains(strings.ToLower(msg), "card") {
            errCode = domain.ErrCardNotFound
        } else if strings.Contains(strings.ToLower(msg), "deck") {
            errCode = domain.ErrDeckNotFound
        } else if strings.Contains(strings.ToLower(msg), "user") {
            errCode = domain.ErrUserNotFound
        } else {
            errCode = domain.ErrInvalidRequest
        }
    case http.StatusUnprocessableEntity:
        errCode = domain.ErrValidation
    case http.StatusTooManyRequests:
        errCode = domain.ErrRateLimited
    case http.StatusRequestTimeout:
        errCode = domain.ErrTimeout
    case http.StatusServiceUnavailable:
        errCode = domain.ErrProviderUnavailable
    default:
        errCode = domain.ErrInternalError
    }

    payload := map[string]interface{}{
        "error":      msg,
        "code":       statusCodeSlug(status), // legacy slug for compatibility
        "status":     status,
        "error_code": errCode,
    }

    _ = json.NewEncoder(w).Encode(payload)
}

// WriteAPIError writes an APIError object using the legacy response shape.
func WriteAPIError(w http.ResponseWriter, apiErr *domain.APIError, status int) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(status)

    payload := map[string]interface{}{
        "error":      apiErr.Error,
        "code":       statusCodeSlug(status),
        "status":     status,
        "error_code": apiErr.Code,
    }
    if apiErr.Details != nil && len(apiErr.Details) > 0 {
        payload["details"] = apiErr.Details
    }
    _ = json.NewEncoder(w).Encode(payload)
}

func statusCodeSlug(code int) string {
    status := strings.ToLower(strings.TrimSpace(http.StatusText(code)))
    if status == "" {
        return "unknown_error"
    }
    status = strings.ReplaceAll(status, "-", " ")
    status = strings.ReplaceAll(status, "  ", " ")
    status = strings.ReplaceAll(status, " ", "_")
    return status
}
