package hubspot

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestCheckResponse_Success(t *testing.T) {
	t.Parallel()
	for _, code := range []int{200, 201, 204} {
		if err := checkResponse(code, http.Header{}, nil); err != nil {
			t.Errorf("checkResponse(%d) = %v, want nil", code, err)
		}
	}
}

func TestCheckResponse_RateLimit429(t *testing.T) {
	t.Parallel()
	header := http.Header{}
	header.Set("Retry-After", "15")
	body := []byte(`{"status":"error","category":"RATE_LIMITS","message":"too many requests"}`)

	err := checkResponse(http.StatusTooManyRequests, header, body)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsRateLimitError(err) {
		t.Errorf("expected RateLimitError, got %T", err)
	}
	var rle *connectors.RateLimitError
	if connectors.AsRateLimitError(err, &rle) {
		if rle.RetryAfter.Seconds() != 15 {
			t.Errorf("RetryAfter = %v, want 15s", rle.RetryAfter)
		}
	}
}

func TestCheckResponse_RateLimitDefaultRetry(t *testing.T) {
	t.Parallel()
	body := []byte(`{"status":"error","message":"rate limited"}`)

	err := checkResponse(http.StatusTooManyRequests, http.Header{}, body)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var rle *connectors.RateLimitError
	if connectors.AsRateLimitError(err, &rle) {
		if rle.RetryAfter.Seconds() != 10 {
			t.Errorf("RetryAfter = %v, want 10s (default)", rle.RetryAfter)
		}
	}
}

func TestCheckResponse_CategoryMapping(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		category string
		status   int
		checker  func(error) bool
	}{
		{"unauthorized", "UNAUTHORIZED", 401, connectors.IsAuthError},
		{"invalid_auth", "INVALID_AUTHENTICATION", 401, connectors.IsAuthError},
		{"revoked_auth", "REVOKED_AUTHENTICATION", 401, connectors.IsAuthError},
		{"rate_limits", "RATE_LIMITS", 400, connectors.IsRateLimitError},
		{"validation_error", "VALIDATION_ERROR", 400, connectors.IsValidationError},
		{"invalid_params", "INVALID_PARAMS", 400, connectors.IsValidationError},
		{"property_doesnt_exist", "PROPERTY_DOESNT_EXIST", 400, connectors.IsValidationError},
		{"invalid_email", "INVALID_EMAIL", 400, connectors.IsValidationError},
		{"contact_exists", "CONTACT_EXISTS", 409, connectors.IsValidationError},
		{"object_not_found", "OBJECT_NOT_FOUND", 404, connectors.IsValidationError},
		{"resource_not_found", "RESOURCE_NOT_FOUND", 404, connectors.IsValidationError},
		{"unknown_category", "SOME_OTHER_ERROR", 500, connectors.IsExternalError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(hubspotError{
				Status:   "error",
				Message:  "test error message",
				Category: tt.category,
			})
			err := checkResponse(tt.status, http.Header{}, body)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.checker(err) {
				t.Errorf("category %q → %T, expected different error type", tt.category, err)
			}
		})
	}
}

func TestCheckResponse_StatusCodeFallback(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		status  int
		checker func(error) bool
	}{
		{"unauthorized", 401, connectors.IsAuthError},
		{"forbidden", 403, connectors.IsAuthError},
		{"bad_request", 400, connectors.IsValidationError},
		{"unprocessable", 422, connectors.IsValidationError},
		{"not_found", 404, connectors.IsValidationError},
		{"server_error", 500, connectors.IsExternalError},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use a body without a category to trigger status code fallback.
			body := []byte(`{"message":"something went wrong"}`)
			err := checkResponse(tt.status, http.Header{}, body)
			if err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.checker(err) {
				t.Errorf("status %d → %T, expected different error type", tt.status, err)
			}
		})
	}
}

func TestCheckResponse_MalformedBody(t *testing.T) {
	t.Parallel()
	err := checkResponse(500, http.Header{}, []byte("not json"))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError for malformed body, got %T", err)
	}
}
