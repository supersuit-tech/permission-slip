package trello

import (
	"net/http"
	"strings"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestTruncateErrorMessage_Short(t *testing.T) {
	t.Parallel()
	msg := "short error"
	result := truncateErrorMessage(msg)
	if result != msg {
		t.Errorf("expected unchanged message, got %q", result)
	}
}

func TestTruncateErrorMessage_ExactLimit(t *testing.T) {
	t.Parallel()
	msg := strings.Repeat("x", maxErrorMessageLen)
	result := truncateErrorMessage(msg)
	if result != msg {
		t.Error("expected unchanged message at exact limit")
	}
}

func TestTruncateErrorMessage_Long(t *testing.T) {
	t.Parallel()
	msg := strings.Repeat("x", maxErrorMessageLen+100)
	result := truncateErrorMessage(msg)
	if !strings.HasSuffix(result, "...(truncated)") {
		t.Errorf("expected truncated suffix, got %q", result[len(result)-20:])
	}
	if len(result) != maxErrorMessageLen+len("...(truncated)") {
		t.Errorf("expected length %d, got %d", maxErrorMessageLen+len("...(truncated)"), len(result))
	}
}

func TestTruncateErrorMessage_Empty(t *testing.T) {
	t.Parallel()
	result := truncateErrorMessage("")
	if result != "" {
		t.Errorf("expected empty string, got %q", result)
	}
}

func TestCheckResponse_EmptyBody(t *testing.T) {
	t.Parallel()
	err := checkResponse(500, http.Header{}, []byte(""))
	if err == nil {
		t.Fatal("expected error for 500")
	}
	// Should use http.StatusText as fallback.
	if !strings.Contains(err.Error(), "Internal Server Error") {
		t.Errorf("expected StatusText fallback, got %q", err.Error())
	}
}

func TestCheckResponse_LongBody(t *testing.T) {
	t.Parallel()
	longBody := []byte(strings.Repeat("a", 2000))
	err := checkResponse(400, http.Header{}, longBody)
	if err == nil {
		t.Fatal("expected error")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T", err)
	}
	if !strings.Contains(err.Error(), "truncated") {
		t.Errorf("expected truncated message, got %q", err.Error())
	}
}
