package intercom

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

const maxErrorBodyPreview = 200

// intercomError represents the Intercom API error response format.
type intercomError struct {
	Type      string            `json:"type"`
	Message   string            `json:"message"`
	RequestID string            `json:"request_id"`
	Errors    []intercomErrItem `json:"errors"`
}

type intercomErrItem struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// truncateBody returns a safe preview of the response body for error messages.
func truncateBody(body []byte) string {
	if len(body) == 0 {
		return "(empty response)"
	}
	return connectors.TruncateUTF8(string(body), maxErrorBodyPreview)
}

// checkResponse inspects the HTTP status code and response body, returning
// an appropriate typed error for non-success responses.
func checkResponse(statusCode int, header http.Header, body []byte) error {
	if statusCode >= 200 && statusCode < 300 {
		return nil
	}

	var icErr intercomError
	msg := truncateBody(body)
	if json.Unmarshal(body, &icErr) == nil {
		if icErr.Message != "" {
			msg = icErr.Message
		} else if len(icErr.Errors) > 0 && icErr.Errors[0].Message != "" {
			msg = icErr.Errors[0].Message
		}
		if icErr.RequestID != "" {
			msg = fmt.Sprintf("%s (request_id: %s)", msg, icErr.RequestID)
		}

		// Map well-known Intercom error codes to actionable messages.
		if len(icErr.Errors) > 0 {
			if mapped := mapIntercomAPIError(icErr.Errors[0].Code, icErr.RequestID); mapped != nil {
				return mapped
			}
		}
	}

	if statusCode == http.StatusTooManyRequests {
		retryAfter := connectors.ParseRetryAfter(header.Get("Retry-After"), 60*time.Second)
		return &connectors.RateLimitError{
			Message:    fmt.Sprintf("Intercom API rate limit exceeded: %s", msg),
			RetryAfter: retryAfter,
		}
	}

	return mapStatusCodeError(statusCode, msg)
}

// mapIntercomAPIError maps well-known Intercom error codes to actionable
// messages with guidance on how to fix common issues.
func mapIntercomAPIError(errCode, requestID string) error {
	suffix := ""
	if requestID != "" {
		suffix = fmt.Sprintf(" (request_id: %s)", requestID)
	}

	switch errCode {
	// Auth errors
	case "token_unauthorized":
		return &connectors.AuthError{Message: "Intercom access token is invalid or expired — generate a new token in Developer Hub > Your App > Authentication" + suffix}
	case "token_not_found":
		return &connectors.AuthError{Message: "Intercom access token not found — verify the token exists and has not been revoked" + suffix}
	case "unauthorized":
		return &connectors.AuthError{Message: "Intercom API access denied — ensure the token has the required scopes for this action" + suffix}

	// Resource errors
	case "not_found":
		return &connectors.ValidationError{Message: "Intercom resource not found — verify the ticket, contact, or admin ID exists" + suffix}
	case "parameter_not_found":
		return &connectors.ValidationError{Message: "Intercom parameter not found — a referenced ID (contact, ticket type, admin) does not exist" + suffix}
	case "parameter_invalid":
		return &connectors.ValidationError{Message: "Intercom parameter is invalid — check field value types and constraints" + suffix}

	// State errors
	case "ticket_state_invalid":
		return &connectors.ValidationError{Message: "Intercom ticket state transition is invalid — tickets can only move to certain states from their current state" + suffix}

	// Contact errors
	case "unique_user_constraint_violation":
		return &connectors.ValidationError{Message: "Intercom contact already exists with this email or external ID" + suffix}

	default:
		return nil
	}
}

func mapStatusCodeError(statusCode int, msg string) error {
	switch {
	case statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden:
		return &connectors.AuthError{Message: fmt.Sprintf("Intercom auth error (%d): %s — verify your access token is valid", statusCode, msg)}
	case statusCode == http.StatusUnprocessableEntity || statusCode == http.StatusBadRequest || statusCode == http.StatusConflict:
		return &connectors.ValidationError{Message: fmt.Sprintf("Intercom validation error: %s", msg)}
	case statusCode == http.StatusNotFound:
		return &connectors.ValidationError{Message: fmt.Sprintf("Intercom resource not found: %s — verify the ID exists and is accessible", msg)}
	default:
		return &connectors.ExternalError{
			StatusCode: statusCode,
			Message:    fmt.Sprintf("Intercom API error: %s", msg),
		}
	}
}
