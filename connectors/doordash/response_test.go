package doordash

import (
	"net/http"
	"strings"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestCheckResponse_Success(t *testing.T) {
	t.Parallel()
	if err := checkResponse(http.StatusOK, []byte(`{"ok":true}`)); err != nil {
		t.Errorf("checkResponse(200) = %v, want nil", err)
	}
}

func TestCheckResponse_Unauthorized(t *testing.T) {
	t.Parallel()
	err := checkResponse(http.StatusUnauthorized, []byte(`{"message":"Invalid JWT"}`))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError, got %T: %v", err, err)
	}
	if !strings.Contains(err.Error(), "developer.doordash.com") {
		t.Errorf("auth error should include developer portal link: %v", err)
	}
}

func TestCheckResponse_Forbidden(t *testing.T) {
	t.Parallel()
	err := checkResponse(http.StatusForbidden, []byte(`{"message":"Access denied"}`))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError, got %T: %v", err, err)
	}
	if !strings.Contains(err.Error(), "developer.doordash.com") {
		t.Errorf("auth error should include developer portal link: %v", err)
	}
}

func TestCheckResponse_BadRequest(t *testing.T) {
	t.Parallel()
	err := checkResponse(http.StatusBadRequest, []byte(`{"message":"Invalid request","field_errors":[{"field":"pickup_address","error":"is required"}]}`))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
	if !strings.Contains(err.Error(), "pickup_address") {
		t.Errorf("error should mention field name: %v", err)
	}
}

func TestCheckResponse_NotFound(t *testing.T) {
	t.Parallel()
	err := checkResponse(http.StatusNotFound, []byte(`{"message":"Delivery not found"}`))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestCheckResponse_RateLimit(t *testing.T) {
	t.Parallel()
	err := checkResponse(http.StatusTooManyRequests, []byte(`{"message":"Rate limited"}`))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsRateLimitError(err) {
		t.Errorf("expected RateLimitError, got %T: %v", err, err)
	}
}

func TestCheckResponse_ServerError(t *testing.T) {
	t.Parallel()
	err := checkResponse(http.StatusInternalServerError, []byte(`{"message":"Something went wrong"}`))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError, got %T: %v", err, err)
	}
}

func TestCheckResponse_UnparseableBody(t *testing.T) {
	t.Parallel()
	err := checkResponse(http.StatusBadRequest, []byte(`not json`))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !strings.Contains(err.Error(), "not json") {
		t.Errorf("error should contain raw body: %v", err)
	}
}

func TestFormatErrorMessage_FieldErrors(t *testing.T) {
	t.Parallel()
	body := []byte(`{"message":"Validation failed","field_errors":[{"field":"pickup_address","error":"is required"},{"field":"dropoff_phone","error":"invalid format"}]}`)
	msg := formatErrorMessage(body)
	if !strings.Contains(msg, "Validation failed") {
		t.Errorf("message should contain top-level message: %q", msg)
	}
	if !strings.Contains(msg, "pickup_address: is required") {
		t.Errorf("message should contain field error: %q", msg)
	}
	if !strings.Contains(msg, "dropoff_phone: invalid format") {
		t.Errorf("message should contain field error: %q", msg)
	}
}

func TestTruncateBody(t *testing.T) {
	t.Parallel()
	short := "short body"
	if got := truncateBody([]byte(short)); got != short {
		t.Errorf("truncateBody(%q) = %q", short, got)
	}

	long := strings.Repeat("x", maxErrorBodyLen+100)
	got := truncateBody([]byte(long))
	if len(got) > maxErrorBodyLen+20 {
		t.Errorf("truncateBody(long) length = %d, want <= %d", len(got), maxErrorBodyLen+20)
	}
	if !strings.HasSuffix(got, "... (truncated)") {
		t.Errorf("truncateBody(long) should end with truncation marker")
	}
}
