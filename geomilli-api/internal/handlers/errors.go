package handlers

import "net/http"

// ErrorCode represents standard machine-readable error codes per RFC-001 §11.2.
type ErrorCode string

const (
	ErrMissingParameter    ErrorCode = "missing_parameter"
	ErrInvalidParameter    ErrorCode = "invalid_parameter"
	ErrInvalidBody         ErrorCode = "invalid_body"
	ErrNotFound            ErrorCode = "not_found"
	ErrUpstreamUnavailable ErrorCode = "upstream_unavailable"
	ErrInternalError       ErrorCode = "internal_error"
	ErrUnauthorized        ErrorCode = "unauthorized" // Reserved for future auth
)

// ErrorEnvelope wraps the standard API error response body.
type ErrorEnvelope struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail describes the error reason, code, and HTTP status code.
type ErrorDetail struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
	Status  int       `json:"status"`
}

// writeError serializes an error response into the standard RFC-001 error envelope.
func writeError(w http.ResponseWriter, status int, code ErrorCode, message string) {
	writeJSON(w, status, ErrorEnvelope{
		Error: ErrorDetail{
			Code:    code,
			Message: message,
			Status:  status,
		},
	})
}
