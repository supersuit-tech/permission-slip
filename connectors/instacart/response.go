package instacart

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// instacartSingleError is the Instacart Connect single-error shape.
type instacartSingleError struct {
	Error struct {
		Message string `json:"message"`
		Code    any    `json:"code"`
	} `json:"error"`
}

// checkResponse maps non-success HTTP responses to typed connector errors.
func checkResponse(statusCode int, header http.Header, body []byte) error {
	if statusCode >= 200 && statusCode < 300 {
		return nil
	}

	const maxErrBody = 512
	msg := string(body)
	if len(msg) > maxErrBody {
		msg = msg[:maxErrBody] + "...(truncated)"
	}

	var single instacartSingleError
	if json.Unmarshal(body, &single) == nil && single.Error.Message != "" {
		msg = strings.TrimSpace(single.Error.Message)
	}

	switch {
	case statusCode == http.StatusTooManyRequests:
		retryAfter := connectors.ParseRetryAfter(header.Get("Retry-After"), defaultRetryAfter)
		return &connectors.RateLimitError{
			Message:    fmt.Sprintf("Instacart API rate limit exceeded: %s", msg),
			RetryAfter: retryAfter,
		}
	case statusCode == http.StatusUnauthorized:
		return &connectors.AuthError{Message: fmt.Sprintf("Instacart API auth error: %s", msg)}
	case statusCode == http.StatusForbidden:
		return &connectors.AuthError{Message: fmt.Sprintf("Instacart API permission error: %s", msg)}
	default:
		return &connectors.ExternalError{StatusCode: statusCode, Message: fmt.Sprintf("Instacart API error: %s", msg)}
	}
}
