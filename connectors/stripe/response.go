package stripe

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// stripeError represents the error envelope returned by the Stripe API.
// See https://docs.stripe.com/api/errors
type stripeError struct {
	Error struct {
		Type    string `json:"type"`
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

// checkResponse inspects the HTTP status code and Stripe error body, returning
// an appropriate typed error for non-success responses.
func checkResponse(statusCode int, header http.Header, body []byte) error {
	if statusCode >= 200 && statusCode < 300 {
		return nil
	}

	// Try to extract Stripe's structured error.
	var se stripeError
	msg := truncate(string(body), maxErrorMessageBytes)
	if json.Unmarshal(body, &se) == nil && se.Error.Message != "" {
		msg = se.Error.Message
		// Include the error code when available (e.g., "card_declined",
		// "expired_card") — useful for debugging and action-level handling.
		if se.Error.Code != "" {
			msg = fmt.Sprintf("%s (code: %s)", msg, se.Error.Code)
		}
	}

	// Rate limit is checked first regardless of error type.
	if statusCode == http.StatusTooManyRequests {
		retryAfter := connectors.ParseRetryAfter(header.Get("Retry-After"), defaultRetryAfter)
		return &connectors.RateLimitError{
			Message:    fmt.Sprintf("Stripe API rate limit exceeded: %s", msg),
			RetryAfter: retryAfter,
		}
	}

	// Map Stripe error types to connector error types.
	if se.Error.Type != "" {
		switch se.Error.Type {
		case "authentication_error":
			return &connectors.AuthError{Message: fmt.Sprintf("Stripe auth error: %s", msg)}
		case "invalid_request_error":
			return &connectors.ValidationError{Message: fmt.Sprintf("Stripe validation error: %s", msg)}
		case "rate_limit_error":
			retryAfter := connectors.ParseRetryAfter(header.Get("Retry-After"), defaultRetryAfter)
			return &connectors.RateLimitError{
				Message:    fmt.Sprintf("Stripe rate limit error: %s", msg),
				RetryAfter: retryAfter,
			}
		case "card_error":
			return &connectors.ExternalError{StatusCode: statusCode, Message: fmt.Sprintf("Stripe card error: %s", msg)}
		case "api_error":
			return &connectors.ExternalError{StatusCode: statusCode, Message: fmt.Sprintf("Stripe API error: %s", msg)}
		}
	}

	// Fallback: map by HTTP status code.
	switch {
	case statusCode == http.StatusUnauthorized:
		return &connectors.AuthError{Message: fmt.Sprintf("Stripe auth error (%d): %s", statusCode, msg)}
	default:
		return &connectors.ExternalError{StatusCode: statusCode, Message: fmt.Sprintf("Stripe API error: %s", msg)}
	}
}
