package zendesk

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

const maxErrorBodyPreview = 200

// zendeskError represents the Zendesk API error response format.
type zendeskError struct {
	Error       string                `json:"error"`
	Description string                `json:"description"`
	Details     map[string][]zdDetail `json:"details"`
	Message     string                `json:"message"`
}

// zdDetail represents a field-level error detail from the Zendesk API.
type zdDetail struct {
	Description string `json:"description"`
	Error       string `json:"error"`
}

// truncateBody returns a safe preview of the response body for error messages.
func truncateBody(body []byte) string {
	if len(body) == 0 {
		return "(empty response)"
	}
	if len(body) <= maxErrorBodyPreview {
		return string(body)
	}
	return string(body[:maxErrorBodyPreview]) + "... (truncated)"
}

// checkResponse inspects the HTTP status code and response body, returning
// an appropriate typed error for non-success responses.
func checkResponse(statusCode int, header http.Header, body []byte) error {
	if statusCode >= 200 && statusCode < 300 {
		return nil
	}

	var zdErr zendeskError
	msg := truncateBody(body)
	if json.Unmarshal(body, &zdErr) == nil {
		// Build a rich error message including field-level details when available.
		msg = buildErrorMessage(zdErr, msg)
	}

	if statusCode == http.StatusTooManyRequests {
		retryAfter := connectors.ParseRetryAfter(header.Get("Retry-After"), 60*time.Second)
		return &connectors.RateLimitError{
			Message:    fmt.Sprintf("Zendesk API rate limit exceeded: %s", msg),
			RetryAfter: retryAfter,
		}
	}

	// Map well-known Zendesk API error codes to actionable messages.
	if mapped := mapZendeskAPIError(zdErr.Error); mapped != nil {
		return mapped
	}

	return mapStatusCodeError(statusCode, msg)
}

// buildErrorMessage constructs a human-readable error from the Zendesk error
// response, including field-level details when present.
func buildErrorMessage(zdErr zendeskError, fallback string) string {
	base := fallback
	if zdErr.Description != "" {
		base = zdErr.Description
	} else if zdErr.Error != "" {
		base = zdErr.Error
	} else if zdErr.Message != "" {
		base = zdErr.Message
	}

	if len(zdErr.Details) == 0 {
		return base
	}

	// Append field-level errors for debugging: "base — field: description"
	var parts []string
	for field, details := range zdErr.Details {
		for _, d := range details {
			desc := d.Description
			if desc == "" {
				desc = d.Error
			}
			if desc != "" {
				parts = append(parts, fmt.Sprintf("%s: %s", field, desc))
			}
		}
	}
	if len(parts) > 0 {
		return fmt.Sprintf("%s — %s", base, strings.Join(parts, "; "))
	}
	return base
}

// mapZendeskAPIError maps well-known Zendesk API error codes to actionable
// error messages, similar to Slack's mapSlackError pattern.
func mapZendeskAPIError(errCode string) error {
	switch errCode {
	// Auth errors
	case "InvalidCredentials":
		return &connectors.AuthError{Message: "Zendesk credentials are invalid — verify your email and API token at Admin > Channels > API"}
	case "Forbidden":
		return &connectors.AuthError{Message: "Zendesk API access denied — ensure the agent has permission for this action"}

	// Record errors
	case "RecordNotFound":
		return &connectors.ValidationError{Message: "Zendesk record not found — verify the ticket or user ID exists"}
	case "RecordInvalid":
		return &connectors.ValidationError{Message: "Zendesk record is invalid — check required fields and field value constraints"}

	// Merge-specific errors
	case "MergeNotPossible":
		return &connectors.ValidationError{Message: "Zendesk merge failed — tickets may already be merged or closed"}

	// Suspended ticket
	case "SuspendedTicket":
		return &connectors.ValidationError{Message: "ticket was suspended by Zendesk — check the suspended tickets queue"}

	default:
		return nil
	}
}

func mapStatusCodeError(statusCode int, msg string) error {
	switch {
	case statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden:
		return &connectors.AuthError{Message: fmt.Sprintf("Zendesk auth error (%d): %s — verify your API token and email are correct", statusCode, msg)}
	case statusCode == http.StatusUnprocessableEntity || statusCode == http.StatusBadRequest || statusCode == http.StatusConflict:
		return &connectors.ValidationError{Message: fmt.Sprintf("Zendesk validation error: %s", msg)}
	case statusCode == http.StatusNotFound:
		return &connectors.ValidationError{Message: fmt.Sprintf("Zendesk resource not found: %s — verify the ticket ID exists and is accessible", msg)}
	default:
		return &connectors.ExternalError{
			StatusCode: statusCode,
			Message:    fmt.Sprintf("Zendesk API error: %s", msg),
		}
	}
}
