package plaid

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestCheckResponse_Success(t *testing.T) {
	t.Parallel()
	if err := checkResponse(http.StatusOK, nil, nil); err != nil {
		t.Errorf("checkResponse(200) = %v, want nil", err)
	}
}

func TestCheckResponse_PlaidError(t *testing.T) {
	t.Parallel()

	body, _ := json.Marshal(map[string]any{
		"error_type":    "INVALID_REQUEST",
		"error_code":    "MISSING_FIELDS",
		"error_message": "access_token is required",
	})

	err := checkResponse(http.StatusBadRequest, nil, body)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestCheckResponse_PlaidErrorFormat(t *testing.T) {
	t.Parallel()

	body, _ := json.Marshal(map[string]any{
		"error_type":    "INVALID_REQUEST",
		"error_code":    "MISSING_FIELDS",
		"error_message": "access_token is required",
	})

	err := checkResponse(http.StatusBadRequest, nil, body)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	msg := err.Error()
	// Error message should include code and type for debugging.
	if !strings.Contains(msg, "MISSING_FIELDS") {
		t.Errorf("error message should include error code, got: %s", msg)
	}
	if !strings.Contains(msg, "INVALID_REQUEST") {
		t.Errorf("error message should include error type, got: %s", msg)
	}
}

func TestCheckResponse_AuthError(t *testing.T) {
	t.Parallel()

	body, _ := json.Marshal(map[string]any{
		"error_type":    "INVALID_INPUT",
		"error_code":    "INVALID_API_KEYS",
		"error_message": "invalid API keys",
	})

	err := checkResponse(http.StatusUnauthorized, nil, body)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError, got %T: %v", err, err)
	}
}

func TestCheckResponse_RateLimit(t *testing.T) {
	t.Parallel()

	err := checkResponse(http.StatusTooManyRequests, nil, []byte(`{"error_message":"rate limited"}`))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsRateLimitError(err) {
		t.Errorf("expected RateLimitError, got %T: %v", err, err)
	}
}

func TestCheckResponse_RateLimitRetryAfter(t *testing.T) {
	t.Parallel()

	header := http.Header{}
	header.Set("Retry-After", "10")

	err := checkResponse(http.StatusTooManyRequests, header, []byte(`{"error_message":"rate limited"}`))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	rle, ok := err.(*connectors.RateLimitError)
	if !ok {
		t.Fatalf("expected *RateLimitError, got %T", err)
	}
	if rle.RetryAfter != 10*time.Second {
		t.Errorf("RetryAfter = %v, want 10s", rle.RetryAfter)
	}
}

func TestCheckResponse_RateLimitDefaultRetryAfter(t *testing.T) {
	t.Parallel()

	err := checkResponse(http.StatusTooManyRequests, nil, []byte(`{"error_message":"rate limited"}`))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	rle, ok := err.(*connectors.RateLimitError)
	if !ok {
		t.Fatalf("expected *RateLimitError, got %T", err)
	}
	if rle.RetryAfter != defaultRetryAfter {
		t.Errorf("RetryAfter = %v, want %v", rle.RetryAfter, defaultRetryAfter)
	}
}

func TestCheckResponse_NotFound(t *testing.T) {
	t.Parallel()

	err := checkResponse(http.StatusNotFound, nil, []byte(`{"error_message":"not found"}`))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError, got %T: %v", err, err)
	}
}

func TestCheckResponse_Forbidden(t *testing.T) {
	t.Parallel()

	err := checkResponse(http.StatusForbidden, nil, []byte(`{"error_message":"forbidden"}`))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError, got %T: %v", err, err)
	}
}

func TestCheckResponse_ServerError(t *testing.T) {
	t.Parallel()

	err := checkResponse(http.StatusInternalServerError, nil, []byte(`{"error_message":"internal error"}`))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError, got %T: %v", err, err)
	}
}

func TestCheckResponse_TruncatesLongBody(t *testing.T) {
	t.Parallel()

	longBody := make([]byte, 1024)
	for i := range longBody {
		longBody[i] = 'x'
	}

	err := checkResponse(http.StatusInternalServerError, nil, longBody)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	// Should contain truncated marker.
	errMsg := err.Error()
	if len(errMsg) > 700 {
		t.Errorf("error message too long (%d chars), expected truncation", len(errMsg))
	}
}

