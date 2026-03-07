package walmart

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// checkResponse inspects the HTTP status code and returns an appropriate
// typed error for non-success responses.
func checkResponse(statusCode int, header http.Header, body []byte) error {
	if statusCode >= 200 && statusCode < 300 {
		return nil
	}

	msg := extractErrorMessage(body)

	switch {
	case statusCode == http.StatusTooManyRequests:
		retryAfter := connectors.ParseRetryAfter(header.Get("Retry-After"), 0)
		return &connectors.RateLimitError{
			Message:    fmt.Sprintf("Walmart API rate limit exceeded: %s", msg),
			RetryAfter: retryAfter,
		}
	case statusCode == http.StatusUnauthorized:
		return &connectors.AuthError{Message: fmt.Sprintf("Walmart API auth error (401): %s", msg)}
	case statusCode == http.StatusForbidden:
		return &connectors.AuthError{Message: fmt.Sprintf("Walmart API forbidden (403): %s", msg)}
	case statusCode == http.StatusBadRequest:
		return &connectors.ValidationError{Message: fmt.Sprintf("Walmart API validation error: %s", msg)}
	case statusCode == http.StatusNotFound:
		return &connectors.ValidationError{Message: fmt.Sprintf("Walmart API resource not found: %s", msg)}
	default:
		return &connectors.ExternalError{
			StatusCode: statusCode,
			Message:    fmt.Sprintf("Walmart API error: %s", msg),
		}
	}
}

// extractErrorMessage tries to pull a human-readable message from the
// Walmart error response. Walmart returns errors in various formats:
//   - {"errors": [{"code": N, "message": "..."}]}
//   - {"message": "..."}
//
// Falls back to the raw body string.
func extractErrorMessage(body []byte) string {
	// Try {"errors": [{"message": "..."}]} format.
	var errorsEnvelope struct {
		Errors []struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"errors"`
	}
	if json.Unmarshal(body, &errorsEnvelope) == nil && len(errorsEnvelope.Errors) > 0 {
		first := errorsEnvelope.Errors[0]
		if first.Message != "" {
			return first.Message
		}
	}

	// Try {"message": "..."} format.
	var singleMsg struct {
		Message string `json:"message"`
	}
	if json.Unmarshal(body, &singleMsg) == nil && singleMsg.Message != "" {
		return singleMsg.Message
	}

	s := string(body)
	if len(s) > 500 {
		s = s[:500] + "... (truncated)"
	}
	if len(s) > 0 {
		return s
	}
	return "unknown error"
}
