package utils

import (
    "encoding/json"
    "log"
    "net/http"
    "student_portal/backend/config"
)

// APIResponse is the standard shape of every response
type APIResponse struct {
    Success bool        `json:"success"`
    Message string      `json:"message"`
    Data    any         `json:"data,omitempty"`
    Error   *APIError   `json:"error,omitempty"`
}

// APIError holds machine-readable error codes
type APIError struct {
    Code   string `json:"code"`
    Detail string `json:"detail,omitempty"`
}

// SendSuccess writes a success response
func SendSuccess(w http.ResponseWriter, statusCode int, message string, data any) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(statusCode)

    resp := APIResponse{
        Success: true,
        Message: message,
        Data:    data,
    }
    _ = json.NewEncoder(w).Encode(resp)
}

// SendError writes an error response
func SendError(w http.ResponseWriter, statusCode int, message string, code string) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(statusCode)

    resp := APIResponse{
        Success: false,
        Message: message,
        Error: &APIError{
            Code:   code,
            Detail: "", // always empty in production
        },
    }
    _ = json.NewEncoder(w).Encode(resp)
}

// SendUnauthorized always returns 401 Unauthorized
func SendUnauthorized(w http.ResponseWriter) {
    SendError(w, http.StatusUnauthorized, "Authentication required", "UNAUTHORIZED")
}

// SendForbidden always returns 403 Forbidden
func SendForbidden(w http.ResponseWriter) {
    SendError(w, http.StatusForbidden, "Access denied", "FORBIDDEN")
}

// SendRateLimited returns 429 Too Many Requests with Retry-After header
func SendRateLimited(w http.ResponseWriter, retryAfterSeconds int) {
    w.Header().Set("Retry-After", string(rune(retryAfterSeconds)))
    SendError(w, http.StatusTooManyRequests, "Too many requests", "RATE_LIMIT_HIT")
}

// SendValidationError returns 400 Bad Request
func SendValidationError(w http.ResponseWriter, detail string, cfg *config.AppConfig) {
    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusBadRequest)

    apiErr := &APIError{Code: "VALIDATION_ERROR"}
    if cfg != nil && cfg.Server.APP_ENV == "development" {
        apiErr.Detail = detail
    }

    resp := APIResponse{
        Success: false,
        Message: "Invalid request",
        Error:   apiErr,
    }
    _ = json.NewEncoder(w).Encode(resp)
}

// SendInternalError returns 500 Internal Server Error
func SendInternalError(w http.ResponseWriter, err error, cfg *config.AppConfig) {
    log.Printf("Internal error: %v", err)

    w.Header().Set("Content-Type", "application/json")
    w.WriteHeader(http.StatusInternalServerError)

    apiErr := &APIError{Code: "INTERNAL_ERROR"}
    if cfg != nil && cfg.Server.APP_ENV == "development" {
        apiErr.Detail = err.Error()
    }

    resp := APIResponse{
        Success: false,
        Message: "Internal server error",
        Error:   apiErr,
    }
    _ = json.NewEncoder(w).Encode(resp)
}
