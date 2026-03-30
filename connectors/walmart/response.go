package walmart

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// checkResponse inspects the HTTP status code and returns an appropriate
// typed error for non-success responses. The error type determines the
// HTTP response code returned to the caller (see connectors/errors.go).
// 404 maps to ValidationError (not ExternalError) because a missing
// product is a client-side input issue, not a server failure.
func checkResponse(statusCode int, header http.Header, body []byte) error {
	if statusCode >= 200 && statusCode < 300 {
		return nil
	}

	msg := extractErrorMessage(body)

	switch {
	case statusCode == http.StatusTooManyRequests:
		retryAfter := connectors.ParseRetryAfter(header.Get("Retry-After"), 0)
		return &connectors.RateLimitError{
			Message:    fmt.Sprintf("Walmart rate limit exceeded: %s", msg),
			RetryAfter: retryAfter,
		}
	case statusCode == http.StatusUnauthorized:
		return &connectors.AuthError{Message: fmt.Sprintf("Walmart auth error: %s", msg)}
	case statusCode == http.StatusForbidden:
		return &connectors.AuthError{Message: fmt.Sprintf("Walmart auth error: %s", msg)}
	case statusCode == http.StatusBadRequest:
		return &connectors.ValidationError{Message: fmt.Sprintf("Walmart validation error: %s", msg)}
	case statusCode == http.StatusNotFound:
		return &connectors.ValidationError{Message: fmt.Sprintf("Walmart resource not found: %s", msg)}
	default:
		return &connectors.ExternalError{
			StatusCode: statusCode,
			Message:    fmt.Sprintf("Walmart error: %s", msg),
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
			if first.Code != 0 {
				return fmt.Sprintf("%s (code: %d)", first.Message, first.Code)
			}
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
	if len(s) > 0 {
		return connectors.TruncateUTF8(s, 500)
	}
	return "unknown error"
}
