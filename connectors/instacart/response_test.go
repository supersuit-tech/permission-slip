package instacart

import (
	"net/http"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
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
