package context

import (
	"errors"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// HandleRateLimit returns a metadata-only Slack context when err is or wraps a
// Slack rate limit (HTTP 429 / RateLimitError / ratelimited). For other errors
// it returns nil, false.
func HandleRateLimit(err error) (*SlackContext, bool) {
	if err == nil {
		return nil, false
	}
	if !connectors.IsRateLimitError(err) {
		return nil, false
	}
	return &SlackContext{
		ContextScope: ScopeMetadataOnly,
	}, true
}

// IsRateLimit reports whether err should trigger metadata-only degradation.
func IsRateLimit(err error) bool {
	if err == nil {
		return false
	}
	if connectors.IsRateLimitError(err) {
		return true
	}
	var ext *connectors.ExternalError
	if errors.As(err, &ext) && ext.StatusCode == 429 {
		return true
	}
	return false
}
