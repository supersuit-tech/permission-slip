package twilio

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestCheckResponse_Success(t *testing.T) {
	t.Parallel()
	for _, code := range []int{200, 201, 204} {
		if err := checkResponse(code, http.Header{}, nil); err != nil {
			t.Errorf("checkResponse(%d) = %v, want nil", code, err)
		}
	}
}

func TestCheckResponse_RateLimit(t *testing.T) {
	t.Parallel()
	h := http.Header{}
	h.Set("Retry-After", "60")
	err := checkResponse(429, h, []byte(`{"code":20429,"message":"Too Many Requests"}`))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsRateLimitError(err) {
		t.Errorf("expected RateLimitError, got %T", err)
	}
	var rle *connectors.RateLimitError
	if connectors.AsRateLimitError(err, &rle) {
		if rle.RetryAfter != 60*time.Second {
			t.Errorf("RetryAfter = %v, want 60s", rle.RetryAfter)
		}
	}
}

func TestCheckResponse_RateLimitNoRetryAfter(t *testing.T) {
	t.Parallel()
	err := checkResponse(429, http.Header{}, []byte(`{"code":20429,"message":"Too Many Requests"}`))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var rle *connectors.RateLimitError
	if connectors.AsRateLimitError(err, &rle) {
		if rle.RetryAfter != 0 {
			t.Errorf("RetryAfter = %v, want 0 (no header)", rle.RetryAfter)
		}
	}
}

func TestCheckResponse_Auth(t *testing.T) {
	t.Parallel()
	for _, code := range []int{401, 403} {
		err := checkResponse(code, http.Header{}, []byte(`{"code":20003,"message":"Authentication Error"}`))
		if err == nil {
			t.Fatalf("checkResponse(%d) expected error, got nil", code)
		}
		if !connectors.IsAuthError(err) {
			t.Errorf("checkResponse(%d) expected AuthError, got %T", code, err)
		}
	}
}

func TestCheckResponse_BadRequest(t *testing.T) {
	t.Parallel()
	err := checkResponse(400, http.Header{}, []byte(`{"code":21211,"message":"Invalid 'To' Phone Number"}`))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T", err)
	}
}

func TestCheckResponse_NotFound(t *testing.T) {
	t.Parallel()
	err := checkResponse(404, http.Header{}, []byte(`{"code":20404,"message":"The requested resource was not found"}`))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError, got %T", err)
	}
}

func TestCheckResponse_ServerError(t *testing.T) {
	t.Parallel()
	err := checkResponse(500, http.Header{}, []byte(`{"code":20500,"message":"Internal Server Error"}`))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError, got %T", err)
	}
}

func TestCheckResponse_TwilioErrorCodeInMessage(t *testing.T) {
	t.Parallel()
	err := checkResponse(400, http.Header{}, []byte(`{"code":21211,"message":"Invalid Phone","more_info":"https://www.twilio.com/docs/errors/21211"}`))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "[21211]") {
		t.Errorf("error message should contain Twilio error code, got %q", msg)
	}
	if !strings.Contains(msg, "Invalid Phone") {
		t.Errorf("error message should contain Twilio message, got %q", msg)
	}
	if !strings.Contains(msg, "https://www.twilio.com/docs/errors/21211") {
		t.Errorf("error message should contain more_info URL, got %q", msg)
	}
}

func TestCheckResponse_NonJSONBody(t *testing.T) {
	t.Parallel()
	err := checkResponse(500, http.Header{}, []byte("Internal Server Error"))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError, got %T", err)
	}
}

func TestCheckResponse_BodyTruncation(t *testing.T) {
	t.Parallel()
	longBody := strings.Repeat("x", 1000)
	err := checkResponse(500, http.Header{}, []byte(longBody))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	msg := err.Error()
	if !strings.Contains(msg, "...(truncated)") {
		t.Errorf("long error body should be truncated, got length %d", len(msg))
	}
	// The raw body portion should be at most 512 chars + truncation suffix
	if strings.Contains(msg, strings.Repeat("x", 513)) {
		t.Error("error message contains more than 512 chars of raw body")
	}
}
