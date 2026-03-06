package shopify

import (
	"net/http"
	"testing"

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
	h.Set("Retry-After", "5")
	err := checkResponse(429, h, []byte(`{"errors":"Exceeded 2 calls per second for api client."}`))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsRateLimitError(err) {
		t.Errorf("expected RateLimitError, got %T", err)
	}
	var rle *connectors.RateLimitError
	if connectors.AsRateLimitError(err, &rle) {
		if rle.RetryAfter != 5*1e9 {
			t.Errorf("RetryAfter = %v, want 5s", rle.RetryAfter)
		}
	}
}

func TestCheckResponse_RateLimitDefaultRetryAfter(t *testing.T) {
	t.Parallel()
	err := checkResponse(429, http.Header{}, []byte(`{"errors":"rate limited"}`))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var rle *connectors.RateLimitError
	if connectors.AsRateLimitError(err, &rle) {
		if rle.RetryAfter != defaultRetryAfter {
			t.Errorf("RetryAfter = %v, want %v", rle.RetryAfter, defaultRetryAfter)
		}
	}
}

func TestCheckResponse_Auth(t *testing.T) {
	t.Parallel()
	for _, code := range []int{401, 403} {
		err := checkResponse(code, http.Header{}, []byte(`{"errors":"[API] Invalid API key"}`))
		if err == nil {
			t.Fatalf("checkResponse(%d) expected error, got nil", code)
		}
		if !connectors.IsAuthError(err) {
			t.Errorf("checkResponse(%d) expected AuthError, got %T", code, err)
		}
	}
}

func TestCheckResponse_ValidationError(t *testing.T) {
	t.Parallel()
	err := checkResponse(422, http.Header{}, []byte(`{"errors":{"title":["can't be blank"]}}`))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T", err)
	}
}

func TestCheckResponse_NotFound(t *testing.T) {
	t.Parallel()
	err := checkResponse(404, http.Header{}, []byte(`{"errors":"Not Found"}`))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError for 404, got %T", err)
	}
}

func TestCheckResponse_ServerError(t *testing.T) {
	t.Parallel()
	err := checkResponse(500, http.Header{}, []byte(`{"errors":"Internal Server Error"}`))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError, got %T", err)
	}
}

func TestExtractErrorMessage_StringErrors(t *testing.T) {
	t.Parallel()
	msg := extractErrorMessage([]byte(`{"errors":"Not Found"}`))
	if msg != "Not Found" {
		t.Errorf("got %q, want %q", msg, "Not Found")
	}
}

func TestExtractErrorMessage_FieldErrors(t *testing.T) {
	t.Parallel()
	msg := extractErrorMessage([]byte(`{"errors":{"title":["can't be blank"]}}`))
	if msg == "" || msg == "unknown error" {
		t.Errorf("expected field error message, got %q", msg)
	}
}

func TestExtractErrorMessage_SingularError(t *testing.T) {
	t.Parallel()
	msg := extractErrorMessage([]byte(`{"error":"Not Found"}`))
	if msg != "Not Found" {
		t.Errorf("got %q, want %q", msg, "Not Found")
	}
}

func TestExtractErrorMessage_EmptyBody(t *testing.T) {
	t.Parallel()
	msg := extractErrorMessage([]byte{})
	if msg != "unknown error" {
		t.Errorf("got %q, want %q", msg, "unknown error")
	}
}
