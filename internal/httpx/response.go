package httpx

import (
	"encoding/json"
	"net/http"
)

type errorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
}

// errorResponseWithDetails extends errorResponse with an optional details field.
// Used when the caller needs to surface structured sub-errors (e.g. provider failures).
type errorResponseWithDetails struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

func JSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func Error(w http.ResponseWriter, status int, code, message string) {
	JSON(w, status, errorResponse{
		Error:   code,
		Message: message,
	})
}

// ErrorWithDetails writes an error response that includes a structured details
// payload (e.g. a slice of provider errors). details is omitted when nil.
func ErrorWithDetails(w http.ResponseWriter, status int, code, message string, details any) {
	JSON(w, status, errorResponseWithDetails{
		Error:   code,
		Message: message,
		Details: details,
	})
}

func BadRequest(w http.ResponseWriter, message string) {
	Error(w, http.StatusBadRequest, "invalid_request", message)
}

func Unauthorized(w http.ResponseWriter, message string) {
	Error(w, http.StatusUnauthorized, "unauthorized", message)
}

func NotFound(w http.ResponseWriter, message string) {
	Error(w, http.StatusNotFound, "not_found", message)
}

func ServiceUnavailable(w http.ResponseWriter, message string) {
	Error(w, http.StatusServiceUnavailable, "service_unavailable", message)
}

func GatewayTimeout(w http.ResponseWriter, message string) {
	Error(w, http.StatusGatewayTimeout, "provider_timeout", message)
}

func InternalError(w http.ResponseWriter) {
	Error(w, http.StatusInternalServerError, "internal_error", "an unexpected error occurred")
}
