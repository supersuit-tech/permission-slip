package sendgrid

import (
	"net/http"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestCheckResponse_Success(t *testing.T) {
	t.Parallel()

	for _, code := range []int{200, 201, 202, 204} {
		if err := checkResponse(code, http.Header{}, nil); err != nil {
			t.Errorf("checkResponse(%d) = %v, want nil", code, err)
		}
	}
}

func TestCheckResponse_Unauthorized(t *testing.T) {
	t.Parallel()

	body := []byte(`{"errors":[{"message":"authorization required"}]}`)
	err := checkResponse(401, http.Header{}, body)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError, got %T: %v", err, err)
	}
}

func TestCheckResponse_Forbidden(t *testing.T) {
	t.Parallel()

	err := checkResponse(403, http.Header{}, []byte(`{"errors":[{"message":"forbidden"}]}`))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError, got %T: %v", err, err)
	}
}

func TestCheckResponse_BadRequest(t *testing.T) {
	t.Parallel()

	body := []byte(`{"errors":[{"message":"invalid field","field":"email"}]}`)
	err := checkResponse(400, http.Header{}, body)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestCheckResponse_RateLimit(t *testing.T) {
	t.Parallel()

	h := http.Header{}
	h.Set("Retry-After", "60")
	err := checkResponse(429, h, []byte(`{"errors":[{"message":"rate limited"}]}`))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsRateLimitError(err) {
		t.Errorf("expected RateLimitError, got %T: %v", err, err)
	}
}

func TestCheckResponse_NotFound(t *testing.T) {
	t.Parallel()

	err := checkResponse(404, http.Header{}, []byte(`{"errors":[{"message":"not found"}]}`))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError, got %T: %v", err, err)
	}
}

func TestCheckResponse_ServerError(t *testing.T) {
	t.Parallel()

	err := checkResponse(500, http.Header{}, []byte(`{"errors":[{"message":"internal error"}]}`))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError, got %T: %v", err, err)
	}
}
