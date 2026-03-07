package kroger

import (
	"net/http"
	"strings"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestCheckResponse_Success(t *testing.T) {
	t.Parallel()

	err := checkResponse(http.StatusOK, nil, []byte(`{"data":[]}`))
	if err != nil {
		t.Errorf("checkResponse(200) = %v, want nil", err)
	}
}

func TestCheckResponse_StructuredError(t *testing.T) {
	t.Parallel()

	body := []byte(`{"errors":[{"code":"NOT_FOUND","message":"Product not found"}]}`)
	err := checkResponse(http.StatusNotFound, http.Header{}, body)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "Product not found") {
		t.Errorf("error should contain structured message, got: %v", err)
	}
}

func TestCheckResponse_StructuredErrorWithReasonFallback(t *testing.T) {
	t.Parallel()

	body := []byte(`{"errors":[{"code":"INVALID","reason":"Bad request format"}]}`)
	err := checkResponse(http.StatusBadRequest, http.Header{}, body)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "Bad request format") {
		t.Errorf("error should contain reason fallback, got: %v", err)
	}
}

func TestCheckResponse_TruncatesLargeBody(t *testing.T) {
	t.Parallel()

	// Create a body larger than 512 chars that doesn't parse as JSON.
	// Use "z" instead of "x" because "x" appears in the error wrapper
	// ("external service error..."), which inflates the count.
	largeBody := []byte(strings.Repeat("z", 1000))
	err := checkResponse(http.StatusInternalServerError, http.Header{}, largeBody)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "...(truncated)") {
		t.Errorf("large error body should be truncated, got length %d", len(msg))
	}
	// The raw body portion should be at most 512 chars + truncation suffix.
	if strings.Count(msg, "z") > 512 {
		t.Error("truncated body should contain at most 512 chars of raw content")
	}
}

func TestCheckResponse_RateLimitWithRetryAfter(t *testing.T) {
	t.Parallel()

	header := http.Header{}
	header.Set("Retry-After", "120")
	body := []byte(`{"errors":[{"message":"Too many requests"}]}`)

	err := checkResponse(http.StatusTooManyRequests, header, body)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsRateLimitError(err) {
		t.Errorf("expected RateLimitError, got %T: %v", err, err)
	}
}

func TestCheckResponse_Unauthorized(t *testing.T) {
	t.Parallel()

	err := checkResponse(http.StatusUnauthorized, http.Header{}, []byte(`{}`))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError, got %T: %v", err, err)
	}
}

func TestCheckResponse_Forbidden(t *testing.T) {
	t.Parallel()

	err := checkResponse(http.StatusForbidden, http.Header{}, []byte(`{}`))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError, got %T: %v", err, err)
	}
}

func TestCheckResponse_ServerError(t *testing.T) {
	t.Parallel()

	err := checkResponse(http.StatusInternalServerError, http.Header{}, []byte(`Internal Server Error`))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError, got %T: %v", err, err)
	}
}
