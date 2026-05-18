package api

import (
	"encoding/json"
	"log"
	"net/http"
)

// Error represents the inner error object matching the spec's Error schema.
type Error struct {
	Code       ErrorCode      `json:"code"`
	Message    string         `json:"message"`
	Retryable  bool           `json:"retryable"`
	Details    map[string]any `json:"details,omitempty"`
	TraceID    string         `json:"trace_id,omitempty"`
	RetryAfter int            `json:"retry_after,omitempty"`
}

// ErrorResponse wraps an Error, matching the spec's ErrorResponse schema.
type ErrorResponse struct {
	Error Error `json:"error"`
}

// RespondError writes a JSON error response with the given HTTP status code.
// If the request context contains a trace ID (from TraceIDMiddleware), it is
// automatically set on the error response.
func RespondError(w http.ResponseWriter, r *http.Request, status int, e ErrorResponse) {
	if traceID := TraceID(r.Context()); traceID != "" && e.Error.TraceID == "" {
		e.Error.TraceID = traceID
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(e); err != nil {
		log.Printf("RespondError: failed to encode error response: %v", err)
	}
}

func newErrorResponse(code ErrorCode, message string, retryable bool) ErrorResponse {
	return ErrorResponse{Error: Error{Code: code, Message: message, Retryable: retryable}}
}

// BadRequest returns a 400 ErrorResponse.
func BadRequest(code ErrorCode, message string) ErrorResponse { return newErrorResponse(code, message, false) }

// Unauthorized returns a 401 ErrorResponse.
func Unauthorized(code ErrorCode, message string) ErrorResponse { return newErrorResponse(code, message, false) }

// Forbidden returns a 403 ErrorResponse.
func Forbidden(code ErrorCode, message string) ErrorResponse { return newErrorResponse(code, message, false) }

// NotFound returns a 404 ErrorResponse.
func NotFound(code ErrorCode, message string) ErrorResponse { return newErrorResponse(code, message, false) }

// Conflict returns a 409 ErrorResponse.
func Conflict(code ErrorCode, message string) ErrorResponse { return newErrorResponse(code, message, false) }

// Gone returns a 410 ErrorResponse.
func Gone(code ErrorCode, message string) ErrorResponse { return newErrorResponse(code, message, false) }

// TooManyRequests returns a 429 ErrorResponse with a retry_after value.
func TooManyRequests(message string, retryAfter int) ErrorResponse {
	resp := newErrorResponse(ErrRateLimited, message, true)
	resp.Error.RetryAfter = retryAfter
	return resp
}

// InternalError returns a 500 ErrorResponse.
func InternalError(message string) ErrorResponse { return newErrorResponse(ErrInternalError, message, true) }

// ServiceUnavailable returns a 503 ErrorResponse.
func ServiceUnavailable(message string) ErrorResponse { return newErrorResponse(ErrServiceUnavailable, message, true) }
