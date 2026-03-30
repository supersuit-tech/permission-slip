package confluence

import (
	"net/http"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestCheckResponse_Success(t *testing.T) {
	t.Parallel()
	if err := checkResponse(200, nil, nil); err != nil {
		t.Errorf("checkResponse(200) = %v, want nil", err)
	}
}

func TestCheckResponse_RateLimit(t *testing.T) {
	t.Parallel()
	err := checkResponse(429, http.Header{"Retry-After": []string{"30"}}, []byte(`{"message":"rate limited"}`))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsRateLimitError(err) {
		t.Errorf("expected RateLimitError, got %T: %v", err, err)
	}
}

func TestCheckResponse_Auth(t *testing.T) {
	t.Parallel()
	for _, code := range []int{401, 403} {
		err := checkResponse(code, nil, []byte(`{"message":"not authorized"}`))
		if err == nil {
			t.Fatalf("expected error for %d, got nil", code)
		}
		if !connectors.IsAuthError(err) {
			t.Errorf("expected AuthError for %d, got %T: %v", code, err, err)
		}
	}
}

func TestCheckResponse_Validation(t *testing.T) {
	t.Parallel()
	for _, code := range []int{400, 422, 404} {
		err := checkResponse(code, nil, []byte(`{"message":"bad request"}`))
		if err == nil {
			t.Fatalf("expected error for %d, got nil", code)
		}
		if !connectors.IsValidationError(err) {
			t.Errorf("expected ValidationError for %d, got %T: %v", code, err, err)
		}
	}
}

func TestCheckResponse_External(t *testing.T) {
	t.Parallel()
	err := checkResponse(500, nil, []byte(`{"message":"internal error"}`))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError, got %T: %v", err, err)
	}
}

func TestExtractErrorMessage_StructuredError(t *testing.T) {
	t.Parallel()
	msg := extractErrorMessage([]byte(`{"statusCode":400,"message":"Title is required"}`))
	if msg != "Title is required" {
		t.Errorf("extractErrorMessage() = %q, want %q", msg, "Title is required")
	}
}

func TestExtractErrorMessage_ErrorsArray(t *testing.T) {
	t.Parallel()
	msg := extractErrorMessage([]byte(`{"errors":[{"status":404,"title":"Page not found"}]}`))
	if msg != "Page not found" {
		t.Errorf("extractErrorMessage() = %q, want %q", msg, "Page not found")
	}
}

func TestExtractErrorMessage_EmptyBody(t *testing.T) {
	t.Parallel()
	msg := extractErrorMessage(nil)
	if msg != "(no error details returned)" {
		t.Errorf("extractErrorMessage() = %q, want %q", msg, "(no error details returned)")
	}
}
