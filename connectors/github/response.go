package github

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// checkResponse inspects the HTTP status code and returns an appropriate
// typed error for non-success responses. The headers parameter is used to
// extract metadata like Retry-After for rate-limit responses.
func checkResponse(statusCode int, header http.Header, body []byte) error {
	if statusCode >= 200 && statusCode < 300 {
		return nil
	}

	// Try to extract GitHub's error message and error details.
	var ghErr struct {
		Message string `json:"message"`
		Errors  []struct {
			Message string `json:"message"`
			Field   string `json:"field"`
			Code    string `json:"code"`
		} `json:"errors"`
	}
	msg := string(body)
	if json.Unmarshal(body, &ghErr) == nil {
		if ghErr.Message != "" {
			msg = ghErr.Message
		}
		// Append error details if available
		if len(ghErr.Errors) > 0 {
			for i, err := range ghErr.Errors {
				if err.Message != "" {
					if i == 0 {
						msg += ". Details: " + err.Message
					} else {
						msg += "; " + err.Message
					}
				}
			}
		}
	}

	switch {
	case statusCode == http.StatusTooManyRequests:
		retryAfter := connectors.ParseRetryAfter(header.Get("Retry-After"), 0)
		return &connectors.RateLimitError{
			Message:    fmt.Sprintf("GitHub API rate limit exceeded: %s", msg),
			RetryAfter: retryAfter,
		}
	case statusCode == http.StatusForbidden && isRateLimited(header):
		retryAfter := connectors.ParseRetryAfter(header.Get("Retry-After"), 0)
		return &connectors.RateLimitError{
			Message:    fmt.Sprintf("GitHub API rate limit exceeded (403): %s", msg),
			RetryAfter: retryAfter,
		}
	case statusCode == http.StatusNotFound:
		return &connectors.ValidationError{Message: fmt.Sprintf("GitHub API resource not found: %s — check that the resource exists and your token has access", msg)}
	case statusCode == http.StatusUnauthorized || statusCode == http.StatusForbidden:
		return &connectors.AuthError{Message: fmt.Sprintf("GitHub API auth error (%d): %s", statusCode, msg)}
	case statusCode == http.StatusUnprocessableEntity:
		return &connectors.ValidationError{Message: fmt.Sprintf("GitHub API validation error: %s", msg)}
	default:
		return &connectors.ExternalError{StatusCode: statusCode, Message: fmt.Sprintf("GitHub API error: %s", msg)}
	}
}

// isRateLimited detects GitHub rate-limit signals on a 403 response.
// GitHub returns 403 (not 429) for primary rate limits when
// X-RateLimit-Remaining is "0", and for secondary/abuse rate limits
// when a Retry-After header is present.
func isRateLimited(header http.Header) bool {
	if header.Get("Retry-After") != "" {
		return true
	}
	return header.Get("X-RateLimit-Remaining") == "0"
}

