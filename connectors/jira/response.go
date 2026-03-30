package jira

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// checkResponse inspects the HTTP status code and returns an appropriate
// typed error for non-success responses. Jira Cloud returns structured
// error bodies with an "errorMessages" array and/or "errors" map.
func checkResponse(statusCode int, header http.Header, body []byte) error {
	if statusCode >= 200 && statusCode < 300 {
		return nil
	}

	msg := extractErrorMessage(body)

	switch {
	case statusCode == http.StatusTooManyRequests:
		retryAfter := connectors.ParseRetryAfter(header.Get("Retry-After"), 0)
		return &connectors.RateLimitError{
			Message:    fmt.Sprintf("Jira API rate limit exceeded: %s", msg),
			RetryAfter: retryAfter,
		}
	case statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden:
		return &connectors.AuthError{Message: fmt.Sprintf("Jira API auth error (%d): %s", statusCode, msg)}
	case statusCode == http.StatusBadRequest || statusCode == http.StatusUnprocessableEntity:
		return &connectors.ValidationError{Message: fmt.Sprintf("Jira API validation error: %s", msg)}
	case statusCode == http.StatusNotFound:
		return &connectors.ValidationError{Message: fmt.Sprintf("Jira API resource not found: %s", msg)}
	default:
		return &connectors.ExternalError{StatusCode: statusCode, Message: fmt.Sprintf("Jira API error: %s", msg)}
	}
}

// extractErrorMessage tries to parse a Jira error response body.
// Jira returns errors in two formats:
//
//	{"errorMessages": ["msg1", "msg2"], "errors": {"field": "msg"}}
//	{"message": "msg"}
func extractErrorMessage(body []byte) string {
	if len(body) == 0 {
		return "(no error details returned)"
	}

	var jiraErr struct {
		ErrorMessages []string          `json:"errorMessages"`
		Errors        map[string]string `json:"errors"`
		Message       string            `json:"message"`
	}
	if json.Unmarshal(body, &jiraErr) != nil {
		return truncate(string(body), 200)
	}

	var parts []string
	parts = append(parts, jiraErr.ErrorMessages...)
	for field, msg := range jiraErr.Errors {
		parts = append(parts, fmt.Sprintf("%s: %s", field, msg))
	}
	if jiraErr.Message != "" {
		parts = append(parts, jiraErr.Message)
	}

	if len(parts) == 0 {
		return truncate(string(body), 200)
	}
	return strings.Join(parts, "; ")
}

// truncate shortens s to maxLen characters, appending an ellipsis if truncated.
func truncate(s string, maxLen int) string {
	return connectors.TruncateUTF8(s, maxLen)
}
