package aws

import (
	"net/http"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestCheckResponse_Success(t *testing.T) {
	t.Parallel()
	if err := checkResponse(http.StatusOK, nil); err != nil {
		t.Errorf("checkResponse(200) = %v, want nil", err)
	}
}

func TestCheckResponse_RateLimit(t *testing.T) {
	t.Parallel()
	err := checkResponse(http.StatusTooManyRequests, []byte("throttled"))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsRateLimitError(err) {
		t.Errorf("expected RateLimitError, got %T", err)
	}
}

func TestCheckResponse_AuthErrors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		status int
		body   string
	}{
		{http.StatusUnauthorized, "bad creds"},
		{http.StatusForbidden, "forbidden"},
	}

	for _, tt := range tests {
		err := checkResponse(tt.status, []byte(tt.body))
		if err == nil {
			t.Fatalf("checkResponse(%d) expected error, got nil", tt.status)
		}
		if !connectors.IsAuthError(err) {
			t.Errorf("checkResponse(%d) = %T, want AuthError", tt.status, err)
		}
	}
}

func TestCheckResponse_ValidationError(t *testing.T) {
	t.Parallel()
	err := checkResponse(http.StatusBadRequest, []byte("invalid param"))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T", err)
	}
}

func TestCheckResponse_ExternalError(t *testing.T) {
	t.Parallel()
	err := checkResponse(http.StatusInternalServerError, []byte("internal error"))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError, got %T", err)
	}
}

func TestExtractErrorMessage_XMLErrorResponse(t *testing.T) {
	t.Parallel()
	body := `<ErrorResponse><Error><Code>InvalidParameterValue</Code><Message>Bad value</Message></Error></ErrorResponse>`
	msg := extractErrorMessage([]byte(body))
	if msg != "InvalidParameterValue: Bad value" {
		t.Errorf("extractErrorMessage() = %q, want %q", msg, "InvalidParameterValue: Bad value")
	}
}

func TestExtractErrorMessage_SimpleXMLError(t *testing.T) {
	t.Parallel()
	body := `<Error><Code>NoSuchBucket</Code><Message>The specified bucket does not exist</Message></Error>`
	msg := extractErrorMessage([]byte(body))
	if msg != "NoSuchBucket: The specified bucket does not exist" {
		t.Errorf("extractErrorMessage() = %q, want %q", msg, "NoSuchBucket: The specified bucket does not exist")
	}
}

func TestExtractErrorMessage_PlainText(t *testing.T) {
	t.Parallel()
	body := `plain text error`
	msg := extractErrorMessage([]byte(body))
	if msg != "plain text error" {
		t.Errorf("extractErrorMessage() = %q, want %q", msg, "plain text error")
	}
}
