package connectors

import (
	"errors"
	"fmt"
	"testing"
	"time"
)

func TestExternalError(t *testing.T) {
	t.Parallel()
	err := &ExternalError{StatusCode: 500, Message: "internal server error"}

	if err.Error() != "external service error (status 500): internal server error" {
		t.Errorf("unexpected error message: %s", err.Error())
	}
}

func TestAuthError(t *testing.T) {
	t.Parallel()
	err := &AuthError{Message: "invalid API key"}

	if err.Error() != "external service auth error: invalid API key" {
		t.Errorf("unexpected error message: %s", err.Error())
	}
}

func TestRateLimitError(t *testing.T) {
	t.Parallel()
	err := &RateLimitError{Message: "too many requests", RetryAfter: 30 * time.Second}

	if err.Error() != "external service rate limit: too many requests (retry after 30s)" {
		t.Errorf("unexpected error message: %s", err.Error())
	}
}

func TestTimeoutError(t *testing.T) {
	t.Parallel()
	err := &TimeoutError{Message: "request timed out"}

	if err.Error() != "external service timeout: request timed out" {
		t.Errorf("unexpected error message: %s", err.Error())
	}
}

func TestValidationError(t *testing.T) {
	t.Parallel()
	err := &ValidationError{Message: "missing required field: owner"}

	if err.Error() != "validation error: missing required field: owner" {
		t.Errorf("unexpected error message: %s", err.Error())
	}
}

func TestIsExternalError(t *testing.T) {
	t.Parallel()
	err := &ExternalError{StatusCode: 502, Message: "bad gateway"}

	if !IsExternalError(err) {
		t.Error("expected IsExternalError to return true for *ExternalError")
	}
	if IsExternalError(errors.New("other")) {
		t.Error("expected IsExternalError to return false for non-ExternalError")
	}
}

func TestIsAuthError(t *testing.T) {
	t.Parallel()
	err := &AuthError{Message: "unauthorized"}

	if !IsAuthError(err) {
		t.Error("expected IsAuthError to return true for *AuthError")
	}
	if IsAuthError(errors.New("other")) {
		t.Error("expected IsAuthError to return false for non-AuthError")
	}
}

func TestIsRateLimitError(t *testing.T) {
	t.Parallel()
	err := &RateLimitError{Message: "rate limited", RetryAfter: 60 * time.Second}

	if !IsRateLimitError(err) {
		t.Error("expected IsRateLimitError to return true for *RateLimitError")
	}
	if IsRateLimitError(errors.New("other")) {
		t.Error("expected IsRateLimitError to return false for non-RateLimitError")
	}
}

func TestIsTimeoutError(t *testing.T) {
	t.Parallel()
	err := &TimeoutError{Message: "timed out"}

	if !IsTimeoutError(err) {
		t.Error("expected IsTimeoutError to return true for *TimeoutError")
	}
	if IsTimeoutError(errors.New("other")) {
		t.Error("expected IsTimeoutError to return false for non-TimeoutError")
	}
}

func TestIsValidationError(t *testing.T) {
	t.Parallel()
	err := &ValidationError{Message: "bad input"}

	if !IsValidationError(err) {
		t.Error("expected IsValidationError to return true for *ValidationError")
	}
	if IsValidationError(errors.New("other")) {
		t.Error("expected IsValidationError to return false for non-ValidationError")
	}
}

func TestIsChecks_WrappedErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		err     error
		checker func(error) bool
	}{
		{"wrapped ExternalError", fmt.Errorf("wrap: %w", &ExternalError{StatusCode: 500, Message: "fail"}), IsExternalError},
		{"wrapped AuthError", fmt.Errorf("wrap: %w", &AuthError{Message: "denied"}), IsAuthError},
		{"wrapped RateLimitError", fmt.Errorf("wrap: %w", &RateLimitError{Message: "slow down", RetryAfter: time.Second}), IsRateLimitError},
		{"wrapped TimeoutError", fmt.Errorf("wrap: %w", &TimeoutError{Message: "timeout"}), IsTimeoutError},
		{"wrapped ValidationError", fmt.Errorf("wrap: %w", &ValidationError{Message: "invalid"}), IsValidationError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !tt.checker(tt.err) {
				t.Errorf("expected Is* check to return true for wrapped error")
			}
		})
	}
}

func TestIsChecks_Nil(t *testing.T) {
	t.Parallel()

	if IsExternalError(nil) {
		t.Error("expected IsExternalError(nil) = false")
	}
	if IsAuthError(nil) {
		t.Error("expected IsAuthError(nil) = false")
	}
	if IsRateLimitError(nil) {
		t.Error("expected IsRateLimitError(nil) = false")
	}
	if IsTimeoutError(nil) {
		t.Error("expected IsTimeoutError(nil) = false")
	}
	if IsValidationError(nil) {
		t.Error("expected IsValidationError(nil) = false")
	}
}
