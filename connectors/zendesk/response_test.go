package zendesk

import (
	"net/http"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
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
	err := checkResponse(429, header, []byte(`{"error":"rate limit"}`))
	if err == nil {
		t.Fatal("expected error for 429")
	}
	if !connectors.IsRateLimitError(err) {
		t.Errorf("expected RateLimitError, got: %T", err)
	}
}

func TestCheckResponse_Unauthorized(t *testing.T) {
	t.Parallel()
	err := checkResponse(401, http.Header{}, []byte(`{"error":"invalid credentials"}`))
	if err == nil {
		t.Fatal("expected error for 401")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError, got: %T", err)
	}
}

func TestCheckResponse_Forbidden(t *testing.T) {
	t.Parallel()
	err := checkResponse(403, http.Header{}, []byte(`{"error":"forbidden"}`))
	if err == nil {
		t.Fatal("expected error for 403")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError, got: %T", err)
	}
}

func TestCheckResponse_NotFound(t *testing.T) {
	t.Parallel()
	err := checkResponse(404, http.Header{}, []byte(`{"error":"not found"}`))
	if err == nil {
		t.Fatal("expected error for 404")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCheckResponse_ValidationError(t *testing.T) {
	t.Parallel()
	err := checkResponse(422, http.Header{}, []byte(`{"description":"invalid field"}`))
	if err == nil {
		t.Fatal("expected error for 422")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCheckResponse_ServerError(t *testing.T) {
	t.Parallel()
	err := checkResponse(500, http.Header{}, []byte(`{"error":"internal"}`))
	if err == nil {
		t.Fatal("expected error for 500")
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError, got: %T", err)
	}
}

func TestCheckResponse_FieldLevelDetails(t *testing.T) {
	t.Parallel()
	body := `{"error":"RecordInvalid","description":"Record validation errors","details":{"base":[{"description":"Subject cannot be blank","error":"BlankError"}]}}`
	err := checkResponse(422, http.Header{}, []byte(body))
	if err == nil {
		t.Fatal("expected error for 422 with details")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
	// Verify that the field-level details are included in the error message.
	errMsg := err.Error()
	if !contains(errMsg, "base: Subject cannot be blank") {
		t.Errorf("expected field-level detail in error, got: %s", errMsg)
	}
}

func TestMapZendeskAPIError_InvalidCredentials(t *testing.T) {
	t.Parallel()
	body := `{"error":"InvalidCredentials","description":"invalid credentials"}`
	err := checkResponse(401, http.Header{}, []byte(body))
	if err == nil {
		t.Fatal("expected error for InvalidCredentials")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError, got: %T", err)
	}
	errMsg := err.Error()
	if !contains(errMsg, "verify your email and API token") {
		t.Errorf("expected actionable guidance in error, got: %s", errMsg)
	}
}

func TestMapZendeskAPIError_RecordNotFound(t *testing.T) {
	t.Parallel()
	body := `{"error":"RecordNotFound","description":"not found"}`
	err := checkResponse(404, http.Header{}, []byte(body))
	if err == nil {
		t.Fatal("expected error for RecordNotFound")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestMapZendeskAPIError_UnknownCode(t *testing.T) {
	t.Parallel()
	body := `{"error":"SomeFutureError","description":"something new"}`
	err := checkResponse(400, http.Header{}, []byte(body))
	if err == nil {
		t.Fatal("expected error for unknown error code")
	}
	// Should fall through to mapStatusCodeError, not mapZendeskAPIError.
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError for 400, got: %T", err)
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
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

	long := make([]byte, 300)
	for i := range long {
		long[i] = 'x'
	}
	got := truncateBody(long)
	if len(got) > maxErrorBodyPreview+20 { // 20 for "... (truncated)"
		t.Errorf("truncateBody(long) too long: %d chars", len(got))
	}
}
