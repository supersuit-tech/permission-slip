package intercom

import (
	"net/http"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestCheckResponse_Success(t *testing.T) {
	t.Parallel()
	err := checkResponse(200, http.Header{}, []byte(`{"ok":true}`))
	if err != nil {
		t.Errorf("expected no error for 200, got: %v", err)
	}
}

func TestCheckResponse_RateLimit(t *testing.T) {
	t.Parallel()
	header := http.Header{}
	header.Set("Retry-After", "30")
	err := checkResponse(429, header, []byte(`{"message":"rate limit"}`))
	if err == nil {
		t.Fatal("expected error for 429")
	}
	if !connectors.IsRateLimitError(err) {
		t.Errorf("expected RateLimitError, got: %T", err)
	}
}

func TestCheckResponse_Unauthorized(t *testing.T) {
	t.Parallel()
	err := checkResponse(401, http.Header{}, []byte(`{"type":"error.list","errors":[{"code":"unauthorized","message":"bad token"}]}`))
	if err == nil {
		t.Fatal("expected error for 401")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError, got: %T", err)
	}
}

func TestCheckResponse_NotFound(t *testing.T) {
	t.Parallel()
	err := checkResponse(404, http.Header{}, []byte(`{"type":"error.list","errors":[{"code":"not_found","message":"not found"}]}`))
	if err == nil {
		t.Fatal("expected error for 404")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCheckResponse_ServerError(t *testing.T) {
	t.Parallel()
	err := checkResponse(500, http.Header{}, []byte(`{"message":"internal error"}`))
	if err == nil {
		t.Fatal("expected error for 500")
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError, got: %T", err)
	}
}

func TestCheckResponse_TokenUnauthorized(t *testing.T) {
	t.Parallel()
	body := `{"type":"error.list","request_id":"req-123","errors":[{"code":"token_unauthorized","message":"token is invalid"}]}`
	err := checkResponse(401, http.Header{}, []byte(body))
	if err == nil {
		t.Fatal("expected error for token_unauthorized")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError, got: %T", err)
	}
	errMsg := err.Error()
	if !containsSubstr(errMsg, "generate a new token") {
		t.Errorf("expected actionable guidance in error, got: %s", errMsg)
	}
	if !containsSubstr(errMsg, "req-123") {
		t.Errorf("expected request_id in error, got: %s", errMsg)
	}
}

func TestCheckResponse_ParameterNotFound(t *testing.T) {
	t.Parallel()
	body := `{"type":"error.list","errors":[{"code":"parameter_not_found","message":"contact not found"}]}`
	err := checkResponse(404, http.Header{}, []byte(body))
	if err == nil {
		t.Fatal("expected error for parameter_not_found")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCheckResponse_UnknownErrorCode(t *testing.T) {
	t.Parallel()
	body := `{"type":"error.list","errors":[{"code":"some_future_code","message":"something"}]}`
	err := checkResponse(400, http.Header{}, []byte(body))
	if err == nil {
		t.Fatal("expected error for unknown code")
	}
	// Should fall through to mapStatusCodeError.
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError for 400, got: %T", err)
	}
}

func containsSubstr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestTruncateBody(t *testing.T) {
	t.Parallel()

	if got := truncateBody(nil); got != "(empty response)" {
		t.Errorf("truncateBody(nil) = %q, want %q", got, "(empty response)")
	}

	short := "short body"
	if got := truncateBody([]byte(short)); got != short {
		t.Errorf("truncateBody(%q) = %q, want %q", short, got, short)
	}
}
