package cloudflare

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

const maxErrorMessageLen = 512

// cfEnvelope is the standard Cloudflare API v4 response wrapper.
// All responses have: { success, errors, messages, result }.
type cfEnvelope struct {
	Success  bool            `json:"success"`
	Errors   []cfError       `json:"errors"`
	Messages []cfMessage     `json:"messages"`
	Result   json.RawMessage `json:"result"`
}

type cfError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

type cfMessage struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// checkResponse inspects the HTTP status code and returns an appropriate
// typed error for non-success responses.
func checkResponse(statusCode int, header http.Header, body []byte) error {
	if statusCode >= 200 && statusCode < 300 {
		return nil
	}

	var envelope cfEnvelope
	var msg string
	if json.Unmarshal(body, &envelope) == nil && len(envelope.Errors) > 0 {
		msg = truncateErrorMessage(envelope.Errors[0].Message)
		for _, e := range envelope.Errors[1:] {
			msg += "; " + truncateErrorMessage(e.Message)
		}
	}
	if msg == "" {
		msg = http.StatusText(statusCode)
	}

	switch {
	case statusCode == http.StatusTooManyRequests:
		retryAfter := connectors.ParseRetryAfter(header.Get("Retry-After"), 0)
		return &connectors.RateLimitError{
			Message:    fmt.Sprintf("Cloudflare API rate limit exceeded: %s", msg),
			RetryAfter: retryAfter,
		}
	case statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden:
		return &connectors.AuthError{Message: fmt.Sprintf("Cloudflare API auth error (%d): %s", statusCode, msg)}
	case statusCode == http.StatusBadRequest:
		return &connectors.ValidationError{Message: fmt.Sprintf("Cloudflare API validation error: %s", msg)}
	case statusCode == http.StatusNotFound:
		return &connectors.ValidationError{Message: fmt.Sprintf("Cloudflare resource not found: %s", msg)}
	default:
		return &connectors.ExternalError{StatusCode: statusCode, Message: fmt.Sprintf("Cloudflare API error: %s", msg)}
	}
}

func truncateErrorMessage(msg string) string {
	return connectors.TruncateUTF8(msg, maxErrorMessageLen)
}
