package connectors

import (
	"errors"
	"fmt"
	"time"
)

// ExternalError indicates the external service returned an error.
// Maps to HTTP 502 Bad Gateway.
type ExternalError struct {
	StatusCode int    // HTTP status code from the external service
	Message    string // human-readable description
}

func (e *ExternalError) Error() string {
	return fmt.Sprintf("external service error (status %d): %s", e.StatusCode, e.Message)
}

// AuthError indicates the external service rejected the provided credentials.
// Maps to HTTP 502 Bad Gateway.
type AuthError struct {
	Message string // human-readable description
}

func (e *AuthError) Error() string {
	return fmt.Sprintf("external service auth error: %s", e.Message)
}

// RateLimitError indicates the external service rate-limited the request.
// Maps to HTTP 429 Too Many Requests.
type RateLimitError struct {
	Message    string        // human-readable description
	RetryAfter time.Duration // how long to wait before retrying
}

func (e *RateLimitError) Error() string {
	return fmt.Sprintf("external service rate limit: %s (retry after %s)", e.Message, e.RetryAfter.String())
}

// TimeoutError indicates the external service did not respond in time.
// Maps to HTTP 504 Gateway Timeout.
type TimeoutError struct {
	Message string // human-readable description
}

func (e *TimeoutError) Error() string {
	return fmt.Sprintf("external service timeout: %s", e.Message)
}

// ValidationError indicates a parameter or credential validation failure.
// Maps to HTTP 400 Bad Request.
type ValidationError struct {
	Message string // human-readable description
}

func (e *ValidationError) Error() string {
	return fmt.Sprintf("validation error: %s", e.Message)
}

// IsExternalError reports whether err is or wraps an *ExternalError.
func IsExternalError(err error) bool {
	var target *ExternalError
	return errors.As(err, &target)
}

// IsAuthError reports whether err is or wraps an *AuthError.
func IsAuthError(err error) bool {
	var target *AuthError
	return errors.As(err, &target)
}

// IsRateLimitError reports whether err is or wraps a *RateLimitError.
func IsRateLimitError(err error) bool {
	var target *RateLimitError
	return errors.As(err, &target)
}

// IsTimeoutError reports whether err is or wraps a *TimeoutError.
func IsTimeoutError(err error) bool {
	var target *TimeoutError
	return errors.As(err, &target)
}

// IsValidationError reports whether err is or wraps a *ValidationError.
func IsValidationError(err error) bool {
	var target *ValidationError
	return errors.As(err, &target)
}

// AsRateLimitError extracts a *RateLimitError from err if present.
// Convenience wrapper around errors.As for callers that need the RetryAfter field.
func AsRateLimitError(err error, target **RateLimitError) bool {
	return errors.As(err, target)
}

// OAuthRefreshError indicates that the OAuth token refresh failed and the user
// needs to re-authorize. Maps to HTTP 401 with a message telling the agent
// the user needs to reconnect the OAuth provider.
type OAuthRefreshError struct {
	Provider string // OAuth provider ID (e.g. "google")
	Message  string // human-readable description
}

func (e *OAuthRefreshError) Error() string {
	return fmt.Sprintf("OAuth token refresh failed for provider %q: %s", e.Provider, e.Message)
}

// IsOAuthRefreshError reports whether err is or wraps an *OAuthRefreshError.
func IsOAuthRefreshError(err error) bool {
	var target *OAuthRefreshError
	return errors.As(err, &target)
}

// AsOAuthRefreshError extracts an *OAuthRefreshError from err if present.
// Convenience wrapper around errors.As for callers that need the Provider field.
func AsOAuthRefreshError(err error, target **OAuthRefreshError) bool {
	return errors.As(err, target)
}
