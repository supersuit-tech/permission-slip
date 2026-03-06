package amadeus

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// amadeusAPIError represents a single error in the Amadeus error response
// envelope: {"errors": [{"status": N, "code": N, "title": "...", "detail": "..."}]}.
type amadeusAPIError struct {
	Status int    `json:"status"`
	Code   int    `json:"code"`
	Title  string `json:"title"`
	Detail string `json:"detail"`
}

// checkResponse inspects the HTTP status code and returns an appropriate
// typed error for non-success responses. Amadeus returns errors in an
// {"errors": [...]} envelope.
func checkResponse(statusCode int, header http.Header, body []byte) error {
	if statusCode >= 200 && statusCode < 300 {
		return nil
	}

	// Extract Amadeus error message from the errors array.
	msg := extractErrorMessage(body)

	switch {
	case statusCode == http.StatusTooManyRequests:
		retryAfter := connectors.ParseRetryAfter(header.Get("Retry-After"), 0)
		return &connectors.RateLimitError{
			Message:    fmt.Sprintf("Amadeus API rate limit exceeded: %s", msg),
			RetryAfter: retryAfter,
		}
	case statusCode == http.StatusUnauthorized:
		return &connectors.AuthError{Message: fmt.Sprintf("Amadeus API auth error: %s", msg)}
	case statusCode == http.StatusForbidden:
		return &connectors.AuthError{Message: fmt.Sprintf("Amadeus API forbidden: %s", msg)}
	case statusCode == http.StatusBadRequest:
		return &connectors.ValidationError{Message: fmt.Sprintf("Amadeus API validation error: %s", msg)}
	default:
		return &connectors.ExternalError{
			StatusCode: statusCode,
			Message:    fmt.Sprintf("Amadeus API error: %s", msg),
		}
	}
}

// extractErrorMessage tries to pull a human-readable message from the Amadeus
// error response envelope. Falls back to the raw body string.
func extractErrorMessage(body []byte) string {
	var envelope struct {
		Errors []amadeusAPIError `json:"errors"`
	}
	if json.Unmarshal(body, &envelope) == nil && len(envelope.Errors) > 0 {
		first := envelope.Errors[0]
		if first.Detail != "" {
			return first.Detail
		}
		if first.Title != "" {
			return first.Title
		}
	}
	return string(body)
}
