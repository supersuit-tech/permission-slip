package kroger

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// krogerErrorResponse is the Kroger API error envelope.
type krogerErrorResponse struct {
	Errors []struct {
		Code    string `json:"code"`
		Reason  string `json:"reason"`
		Message string `json:"message"`
	} `json:"errors"`
}

// checkResponse inspects the HTTP status code and returns an appropriate
// typed error for non-success responses.
func checkResponse(statusCode int, header http.Header, body []byte) error {
	if statusCode >= 200 && statusCode < 300 {
		return nil
	}

	// Truncate raw body to prevent leaking large or sensitive payloads in
	// error messages (the body could be up to maxResponseBytes).
	const maxErrBody = 512
	msg := connectors.TruncateUTF8(string(body), maxErrBody)

	// Try to extract a structured Kroger API error message.
	var kErr krogerErrorResponse
	if json.Unmarshal(body, &kErr) == nil && len(kErr.Errors) > 0 {
		if kErr.Errors[0].Message != "" {
			msg = kErr.Errors[0].Message
		} else if kErr.Errors[0].Reason != "" {
			msg = kErr.Errors[0].Reason
		}
	}

	switch {
	case statusCode == http.StatusTooManyRequests:
		retryAfter := connectors.ParseRetryAfter(header.Get("Retry-After"), defaultRetryAfter)
		return &connectors.RateLimitError{
			Message:    fmt.Sprintf("Kroger API rate limit exceeded: %s", msg),
			RetryAfter: retryAfter,
		}
	case statusCode == http.StatusUnauthorized:
		return &connectors.AuthError{Message: fmt.Sprintf("Kroger API auth error: %s", msg)}
	case statusCode == http.StatusForbidden:
		return &connectors.AuthError{Message: fmt.Sprintf("Kroger API permission error: %s", msg)}
	default:
		return &connectors.ExternalError{StatusCode: statusCode, Message: fmt.Sprintf("Kroger API error: %s", msg)}
	}
}
