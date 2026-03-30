package hubspot

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// maxErrorBodyPreview limits how much of a raw (non-JSON) response body
// is included in error messages to prevent information leakage and log bloat.
const maxErrorBodyPreview = 200

// hubspotError represents the standard HubSpot API error response format.
// HubSpot returns {"status": "error", "category": "...", "message": "...",
// "correlationId": "..."}. The correlationId is included in error messages
// to help users troubleshoot issues with HubSpot support.
type hubspotError struct {
	Status        string `json:"status"`
	Message       string `json:"message"`
	Category      string `json:"category"`
	CorrelationID string `json:"correlationId"`
}

// truncateBody returns a safe preview of the response body for error messages.
// It prevents leaking large or sensitive raw responses into logs/error strings.
func truncateBody(body []byte) string {
	if len(body) == 0 {
		return "(empty response)"
	}
	return connectors.TruncateUTF8(string(body), maxErrorBodyPreview)
}

// checkResponse inspects the HTTP status code and response body, returning
// an appropriate typed error for non-success responses. HubSpot signals
// error categories via the JSON body's "category" field.
func checkResponse(statusCode int, header http.Header, body []byte) error {
	if statusCode >= 200 && statusCode < 300 {
		return nil
	}

	// Try to extract HubSpot's structured error message. If the body
	// isn't valid JSON, fall back to a generic message with a truncated
	// body preview — never pass the full raw body into error messages
	// to avoid leaking internal server details or bloating logs.
	var hsErr hubspotError
	msg := truncateBody(body)
	if json.Unmarshal(body, &hsErr) == nil && hsErr.Message != "" {
		msg = hsErr.Message
		if hsErr.CorrelationID != "" {
			msg = fmt.Sprintf("%s (correlationId: %s)", msg, hsErr.CorrelationID)
		}
	}

	// HTTP 429 always means rate limit, regardless of body category.
	if statusCode == http.StatusTooManyRequests {
		retryAfter := connectors.ParseRetryAfter(header.Get("Retry-After"), 10*time.Second)
		return &connectors.RateLimitError{
			Message:    fmt.Sprintf("HubSpot API rate limit exceeded: %s", msg),
			RetryAfter: retryAfter,
		}
	}

	// Map by category when available, fall back to status code.
	if hsErr.Category != "" {
		return mapCategoryError(hsErr.Category, msg, statusCode)
	}

	return mapStatusCodeError(statusCode, msg)
}

// mapCategoryError maps HubSpot's error category strings to typed errors.
func mapCategoryError(category, msg string, statusCode int) error {
	switch category {
	case "UNAUTHORIZED", "INVALID_AUTHENTICATION", "REVOKED_AUTHENTICATION":
		return &connectors.AuthError{Message: fmt.Sprintf("HubSpot auth error: %s", msg)}
	case "RATE_LIMITS":
		return &connectors.RateLimitError{
			Message:    fmt.Sprintf("HubSpot API rate limit exceeded: %s", msg),
			RetryAfter: 10 * time.Second,
		}
	case "VALIDATION_ERROR", "INVALID_PARAMS", "PROPERTY_DOESNT_EXIST",
		"INVALID_EMAIL", "CONTACT_EXISTS":
		return &connectors.ValidationError{Message: fmt.Sprintf("HubSpot validation error: %s", msg)}
	case "OBJECT_NOT_FOUND", "RESOURCE_NOT_FOUND":
		return &connectors.ValidationError{Message: fmt.Sprintf("HubSpot resource not found: %s", msg)}
	default:
		return &connectors.ExternalError{
			StatusCode: statusCode,
			Message:    fmt.Sprintf("HubSpot API error (%s): %s", category, msg),
		}
	}
}

// mapStatusCodeError is the fallback when the response body lacks a category.
func mapStatusCodeError(statusCode int, msg string) error {
	switch {
	case statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden:
		return &connectors.AuthError{Message: fmt.Sprintf("HubSpot auth error (%d): %s", statusCode, msg)}
	case statusCode == http.StatusUnprocessableEntity || statusCode == http.StatusBadRequest || statusCode == http.StatusConflict:
		return &connectors.ValidationError{Message: fmt.Sprintf("HubSpot validation error: %s", msg)}
	case statusCode == http.StatusNotFound:
		return &connectors.ValidationError{Message: fmt.Sprintf("HubSpot resource not found: %s", msg)}
	default:
		return &connectors.ExternalError{
			StatusCode: statusCode,
			Message:    fmt.Sprintf("HubSpot API error: %s", msg),
		}
	}
}
