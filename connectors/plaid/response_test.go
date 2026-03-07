package plaid

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestCheckResponse_Success(t *testing.T) {
	t.Parallel()
	if err := checkResponse(http.StatusOK, nil); err != nil {
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

	err := checkResponse(http.StatusBadRequest, body)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestCheckResponse_AuthError(t *testing.T) {
	t.Parallel()

	body, _ := json.Marshal(map[string]any{
		"error_type":    "INVALID_INPUT",
		"error_code":    "INVALID_API_KEYS",
		"error_message": "invalid API keys",
	})

	err := checkResponse(http.StatusUnauthorized, body)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError, got %T: %v", err, err)
	}
}

func TestCheckResponse_RateLimit(t *testing.T) {
	t.Parallel()

	err := checkResponse(http.StatusTooManyRequests, []byte(`{"error_message":"rate limited"}`))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsRateLimitError(err) {
		t.Errorf("expected RateLimitError, got %T: %v", err, err)
	}
}

func TestCheckResponse_NotFound(t *testing.T) {
	t.Parallel()

	err := checkResponse(http.StatusNotFound, []byte(`{"error_message":"not found"}`))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError, got %T: %v", err, err)
	}
}

func TestCheckResponse_Forbidden(t *testing.T) {
	t.Parallel()

	err := checkResponse(http.StatusForbidden, []byte(`{"error_message":"forbidden"}`))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError, got %T: %v", err, err)
	}
}

func TestCheckResponse_ServerError(t *testing.T) {
	t.Parallel()

	err := checkResponse(http.StatusInternalServerError, []byte(`{"error_message":"internal error"}`))
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

	err := checkResponse(http.StatusInternalServerError, longBody)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	// Should contain truncated marker.
	errMsg := err.Error()
	if len(errMsg) > 700 {
		t.Errorf("error message too long (%d chars), expected truncation", len(errMsg))
	}
}
