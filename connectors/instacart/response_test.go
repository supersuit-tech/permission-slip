package instacart

import (
	"net/http"
	"strings"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestCheckResponse_Success(t *testing.T) {
	t.Parallel()
	if err := checkResponse(http.StatusOK, nil, nil); err != nil {
		t.Errorf("checkResponse(200) = %v, want nil", err)
	}
}

func TestCheckResponse_Unauthorized(t *testing.T) {
	t.Parallel()
	body := []byte(`{"error":{"message":"bad key","code":401}}`)
	err := checkResponse(http.StatusUnauthorized, nil, body)
	if !connectors.IsAuthError(err) {
		t.Fatalf("want AuthError, got %T: %v", err, err)
	}
}

func TestCheckResponse_MultiErrorFirstMessage(t *testing.T) {
	t.Parallel()
	body := []byte(`{"errors":[{"message":"Invalid health filters"}]}`)
	err := checkResponse(http.StatusBadRequest, nil, body)
	ext, ok := err.(*connectors.ExternalError)
	if !ok {
		t.Fatalf("want ExternalError, got %T", err)
	}
	if !strings.Contains(ext.Message, "Invalid health") {
		t.Errorf("message = %q", ext.Message)
	}
}

func TestCheckResponse_TruncateLongMessage(t *testing.T) {
	t.Parallel()
	longMsg := strings.Repeat("x", 600)
	body := []byte(`{"error":{"message":"` + longMsg + `"}}`)
	err := checkResponse(http.StatusBadRequest, nil, body)
	ext, ok := err.(*connectors.ExternalError)
	if !ok {
		t.Fatalf("want ExternalError, got %T", err)
	}
	if !strings.Contains(ext.Message, "...(truncated)") {
		t.Errorf("expected truncated API message in %q", ext.Message)
	}
}

func TestCheckResponse_UnprocessableEntity(t *testing.T) {
	t.Parallel()
	body := []byte(`{"error":{"message":"line_items[0].name is required"}}`)
	err := checkResponse(http.StatusUnprocessableEntity, nil, body)
	if !connectors.IsValidationError(err) {
		t.Fatalf("want ValidationError, got %T: %v", err, err)
	}
}

func TestCheckResponse_RateLimit(t *testing.T) {
	t.Parallel()
	h := http.Header{}
	h.Set("Retry-After", "120")
	err := checkResponse(http.StatusTooManyRequests, h, []byte(`{}`))
	rl, ok := err.(*connectors.RateLimitError)
	if !ok {
		t.Fatalf("want RateLimitError, got %T", err)
	}
	if rl.RetryAfter == 0 {
		t.Error("expected RetryAfter to be set")
	}
}
