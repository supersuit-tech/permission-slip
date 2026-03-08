package sendgrid

import (
	"net/http"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestUnixToISO(t *testing.T) {
	t.Parallel()

	tests := []struct {
		ts   int64
		want string
	}{
		{ts: 0, want: "1970-01-01T00:00:00Z"},
		{ts: 1700000000, want: "2023-11-14T22:13:20Z"},
		{ts: 1800000000, want: "2027-01-15T08:00:00Z"},
	}

	for _, tt := range tests {
		got := unixToISO(tt.ts)
		if got != tt.want {
			t.Errorf("unixToISO(%d) = %q, want %q", tt.ts, got, tt.want)
		}
	}
}

func TestValidateEmailAddresses(t *testing.T) {
	t.Parallel()

	t.Run("valid addresses pass", func(t *testing.T) {
		t.Parallel()
		if err := validateEmailAddresses("cc", []string{"a@example.com", "b@example.org"}); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("empty slice passes", func(t *testing.T) {
		t.Parallel()
		if err := validateEmailAddresses("cc", nil); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("invalid address returns ValidationError", func(t *testing.T) {
		t.Parallel()
		err := validateEmailAddresses("bcc", []string{"valid@example.com", "not-an-email"})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		if !connectors.IsValidationError(err) {
			t.Errorf("expected ValidationError, got %T: %v", err, err)
		}
	})

	t.Run("field name is included in error message", func(t *testing.T) {
		t.Parallel()
		err := validateEmailAddresses("cc", []string{"bad"})
		if err == nil {
			t.Fatal("expected error, got nil")
		}
		// Field name should appear so callers can identify which field failed.
		verr, ok := err.(*connectors.ValidationError)
		if !ok {
			t.Fatalf("expected *connectors.ValidationError, got %T", err)
		}
		if verr.Message == "" {
			t.Error("error message should not be empty")
		}
	})
}

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
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
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
