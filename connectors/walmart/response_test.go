package walmart

import (
	"net/http"
	"strings"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestCheckResponse_Success(t *testing.T) {
	t.Parallel()
	err := checkResponse(200, http.Header{}, []byte(`{}`))
	if err != nil {
		t.Fatalf("checkResponse(200) unexpected error: %v", err)
	}
}

func TestCheckResponse_RateLimit(t *testing.T) {
	t.Parallel()
	h := http.Header{}
	h.Set("Retry-After", "5")
	err := checkResponse(429, h, walmartErrorResponse(429, "rate limit"))
	if err == nil {
		t.Fatal("checkResponse(429) expected error, got nil")
	}
	if !connectors.IsRateLimitError(err) {
		t.Errorf("expected RateLimitError, got %T: %v", err, err)
	}
}

func TestCheckResponse_Unauthorized(t *testing.T) {
	t.Parallel()
	err := checkResponse(401, http.Header{}, walmartErrorResponse(401, "invalid key"))
	if err == nil {
		t.Fatal("expected error")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError, got %T: %v", err, err)
	}
}

func TestCheckResponse_Forbidden(t *testing.T) {
	t.Parallel()
	err := checkResponse(403, http.Header{}, walmartErrorResponse(403, "forbidden"))
	if err == nil {
		t.Fatal("expected error")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError, got %T: %v", err, err)
	}
}

func TestCheckResponse_BadRequest(t *testing.T) {
	t.Parallel()
	err := checkResponse(400, http.Header{}, walmartErrorResponse(400, "bad request"))
	if err == nil {
		t.Fatal("expected error")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestCheckResponse_NotFound(t *testing.T) {
	t.Parallel()
	err := checkResponse(404, http.Header{}, walmartErrorResponse(404, "not found"))
	if err == nil {
		t.Fatal("expected error")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestCheckResponse_ServerError(t *testing.T) {
	t.Parallel()
	err := checkResponse(500, http.Header{}, []byte(`{"message":"Internal Server Error"}`))
	if err == nil {
		t.Fatal("expected error")
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError, got %T: %v", err, err)
	}
}

func TestExtractErrorMessage_ErrorsArrayWithCode(t *testing.T) {
	t.Parallel()
	body := walmartErrorResponse(400, "Invalid query parameter")
	msg := extractErrorMessage(body)
	want := "Invalid query parameter (code: 400)"
	if msg != want {
		t.Errorf("extractErrorMessage = %q, want %q", msg, want)
	}
}

func TestExtractErrorMessage_ErrorsArrayNoCode(t *testing.T) {
	t.Parallel()
	body := []byte(`{"errors":[{"code":0,"message":"Something failed"}]}`)
	msg := extractErrorMessage(body)
	if msg != "Something failed" {
		t.Errorf("extractErrorMessage = %q, want %q", msg, "Something failed")
	}
}

func TestExtractErrorMessage_SingleMessage(t *testing.T) {
	t.Parallel()
	msg := extractErrorMessage([]byte(`{"message":"Something went wrong"}`))
	if msg != "Something went wrong" {
		t.Errorf("extractErrorMessage = %q, want %q", msg, "Something went wrong")
	}
}

func TestExtractErrorMessage_Fallback(t *testing.T) {
	t.Parallel()
	msg := extractErrorMessage([]byte(`plain text error`))
	if msg != "plain text error" {
		t.Errorf("extractErrorMessage = %q, want %q", msg, "plain text error")
	}
}

func TestExtractErrorMessage_Empty(t *testing.T) {
	t.Parallel()
	msg := extractErrorMessage([]byte{})
	if msg != "unknown error" {
		t.Errorf("extractErrorMessage = %q, want %q", msg, "unknown error")
	}
}

func TestExtractErrorMessage_Truncation(t *testing.T) {
	t.Parallel()
	long := strings.Repeat("x", 600)
	msg := extractErrorMessage([]byte(long))
	if len(msg) > 520 {
		t.Errorf("extractErrorMessage should truncate long bodies, got len %d", len(msg))
	}
	if !strings.HasSuffix(msg, "...(truncated)") {
		t.Errorf("expected truncation suffix, got %q", msg[len(msg)-20:])
	}
}
