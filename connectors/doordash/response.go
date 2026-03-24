package doordash

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// doordashErrorResponse represents the DoorDash API error envelope.
// DoorDash returns errors as: {"field_errors": [...], "message": "..."}
type doordashErrorResponse struct {
	FieldErrors []doordashFieldError `json:"field_errors"`
	Message     string               `json:"message"`
	Code        string               `json:"code"`
}

type doordashFieldError struct {
	Field   string `json:"field"`
	Error   string `json:"error"`
}

// checkResponse inspects the HTTP status code and returns an appropriate
// typed error for non-success responses.
func checkResponse(statusCode int, body []byte) error {
	if statusCode >= 200 && statusCode < 300 {
		return nil
	}

	msg := formatErrorMessage(body)

	switch {
	case statusCode == http.StatusTooManyRequests:
		return &connectors.RateLimitError{
			Message: fmt.Sprintf("DoorDash API rate limit exceeded: %s", msg),
		}
	case statusCode == http.StatusUnauthorized:
		return &connectors.AuthError{Message: fmt.Sprintf("DoorDash API auth error: %s — verify your developer_id, key_id, and signing_secret at https://developer.doordash.com/portal/integration/drive", msg)}
	case statusCode == http.StatusForbidden:
		return &connectors.AuthError{Message: fmt.Sprintf("DoorDash API forbidden: %s — your credentials may lack Drive API access. Check your app permissions at https://developer.doordash.com/portal/integration/drive", msg)}
	case statusCode == http.StatusBadRequest:
		return &connectors.ValidationError{Message: fmt.Sprintf("DoorDash API validation error: %s", msg)}
	case statusCode == http.StatusNotFound:
		return &connectors.ValidationError{Message: fmt.Sprintf("DoorDash API not found: %s", msg)}
	default:
		return &connectors.ExternalError{StatusCode: statusCode, Message: fmt.Sprintf("DoorDash API error: %s", msg)}
	}
}

const maxErrorBodyLen = 512

// formatErrorMessage builds a human-readable message from the DoorDash error response.
func formatErrorMessage(body []byte) string {
	var resp doordashErrorResponse
	if json.Unmarshal(body, &resp) != nil {
		return truncateBody(body)
	}

	var parts []string

	if resp.Message != "" {
		parts = append(parts, resp.Message)
	}

	for _, fe := range resp.FieldErrors {
		part := fe.Error
		if fe.Field != "" {
			part = fe.Field + ": " + part
		}
		parts = append(parts, part)
	}

	if len(parts) == 0 {
		return truncateBody(body)
	}
	return connectors.TruncateUTF8(strings.Join(parts, "; "), maxErrorBodyLen)
}

func truncateBody(body []byte) string {
	return connectors.TruncateUTF8(string(body), maxErrorBodyLen)
}
