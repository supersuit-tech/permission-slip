package paypal

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// paypalError is a common error shape from PayPal REST APIs.
type paypalError struct {
	Name    string `json:"name"`
	Message string `json:"message"`
	DebugID string `json:"debug_id"`
	Details []struct {
		Issue       string `json:"issue"`
		Description string `json:"description"`
	} `json:"details"`
}

func checkResponse(statusCode int, header http.Header, body []byte) error {
	if statusCode >= 200 && statusCode < 300 {
		return nil
	}

	msg := truncate(string(body), maxErrorMessageBytes)
	var pe paypalError
	if json.Unmarshal(body, &pe) == nil {
		if pe.Message != "" {
			msg = pe.Message
			if pe.Name != "" {
				msg = fmt.Sprintf("%s: %s", pe.Name, pe.Message)
			}
			if pe.DebugID != "" {
				msg = fmt.Sprintf("%s (debug_id: %s)", msg, pe.DebugID)
			}
			if len(pe.Details) > 0 && pe.Details[0].Issue != "" {
				msg = fmt.Sprintf("%s — %s", msg, pe.Details[0].Issue)
			}
		}
	}

	if statusCode == http.StatusTooManyRequests {
		retryAfter := connectors.ParseRetryAfter(header.Get("Retry-After"), defaultRetryAfter)
		return &connectors.RateLimitError{
			Message:    fmt.Sprintf("PayPal API rate limit exceeded: %s", msg),
			RetryAfter: retryAfter,
		}
	}

	lowerName := strings.ToLower(pe.Name)
	switch {
	case statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden:
		return &connectors.AuthError{Message: fmt.Sprintf("PayPal auth error (%d): %s", statusCode, msg)}
	case statusCode == http.StatusBadRequest || statusCode == http.StatusUnprocessableEntity:
		if strings.Contains(lowerName, "validation") || strings.Contains(lowerName, "invalid") {
			return &connectors.ValidationError{Message: fmt.Sprintf("PayPal validation error: %s", msg)}
		}
		return &connectors.ValidationError{Message: fmt.Sprintf("PayPal request error: %s", msg)}
	default:
		return &connectors.ExternalError{StatusCode: statusCode, Message: fmt.Sprintf("PayPal API error: %s", msg)}
	}
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
