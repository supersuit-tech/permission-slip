package twilio

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

	// Try to extract Twilio's error message and code.
	var twilioErr struct {
		Code    int    `json:"code"`
		Message string `json:"message"`
	}
	msg := string(body)
	if json.Unmarshal(body, &twilioErr) == nil && twilioErr.Message != "" {
		msg = twilioErr.Message
	}

	switch {
	case statusCode == http.StatusTooManyRequests:
		retryAfter := connectors.ParseRetryAfter(header.Get("Retry-After"), 0)
		return &connectors.RateLimitError{
			Message:    fmt.Sprintf("Twilio API rate limit exceeded: %s", msg),
			RetryAfter: retryAfter,
		}
	case statusCode == http.StatusUnauthorized:
		return &connectors.AuthError{Message: fmt.Sprintf("Twilio API auth error: %s", msg)}
	case statusCode == http.StatusForbidden:
		return &connectors.AuthError{Message: fmt.Sprintf("Twilio API forbidden: %s", msg)}
	case statusCode == http.StatusBadRequest:
		return &connectors.ValidationError{Message: fmt.Sprintf("Twilio API validation error: %s", msg)}
	case statusCode == http.StatusNotFound:
		return &connectors.ExternalError{StatusCode: statusCode, Message: fmt.Sprintf("Twilio API resource not found: %s", msg)}
	default:
		return &connectors.ExternalError{StatusCode: statusCode, Message: fmt.Sprintf("Twilio API error: %s", msg)}
	}
}
