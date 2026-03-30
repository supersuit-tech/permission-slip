package expedia

import (
	"net/http"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestCheckResponse_Success(t *testing.T) {
	t.Parallel()

	for _, code := range []int{200, 201, 204} {
		if err := checkResponse(code, http.Header{}, nil); err != nil {
			t.Errorf("checkResponse(%d) = %v, want nil", code, err)
		}
	}
}

func TestCheckResponse_NonJSONBody(t *testing.T) {
	t.Parallel()

	// When the body isn't valid JSON, the raw body should be used as the message.
	err := checkResponse(http.StatusInternalServerError, http.Header{}, []byte("gateway timeout"))
	if err == nil {
		t.Fatal("checkResponse(500) expected error, got nil")
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("checkResponse(500) = %T, want *connectors.ExternalError", err)
	}
	if got := err.Error(); got == "" {
		t.Error("error message is empty")
	}
}

func TestCheckResponse_EmptyBody(t *testing.T) {
	t.Parallel()

	err := checkResponse(http.StatusBadRequest, http.Header{}, []byte{})
	if err == nil {
		t.Fatal("checkResponse(400) expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("checkResponse(400) = %T, want *connectors.ValidationError", err)
	}
}

func TestCheckResponse_ForbiddenIsAuthError(t *testing.T) {
	t.Parallel()

	body := []byte(`{"type":"request_forbidden","message":"Access denied"}`)
	err := checkResponse(http.StatusForbidden, http.Header{}, body)
	if err == nil {
		t.Fatal("checkResponse(403) expected error, got nil")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("checkResponse(403) = %T, want *connectors.AuthError", err)
	}
}

func TestCheckResponse_404IsValidationError(t *testing.T) {
	t.Parallel()

	body := []byte(`{"type":"resource_not_found","message":"Property not found"}`)
	err := checkResponse(http.StatusNotFound, http.Header{}, body)
	if err == nil {
		t.Fatal("checkResponse(404) expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("checkResponse(404) = %T, want *connectors.ValidationError", err)
	}
}

func TestCheckResponse_RateLimitWithoutRetryAfter(t *testing.T) {
	t.Parallel()

	body := []byte(`{"type":"too_many_requests","message":"Rate limited"}`)
	err := checkResponse(http.StatusTooManyRequests, http.Header{}, body)
	if err == nil {
		t.Fatal("checkResponse(429) expected error, got nil")
	}
	var rlErr *connectors.RateLimitError
	if !connectors.AsRateLimitError(err, &rlErr) {
		t.Fatalf("checkResponse(429) = %T, want *connectors.RateLimitError", err)
	}
	if rlErr.RetryAfter != 0 {
		t.Errorf("RetryAfter = %v, want 0 (no Retry-After header)", rlErr.RetryAfter)
	}
}
