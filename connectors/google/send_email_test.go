package google

import (
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestSendEmail_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/gmail/v1/users/me/messages/send" {
			t.Errorf("expected path /gmail/v1/users/me/messages/send, got %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer ya29.test-access-token-123" {
			t.Errorf("expected Bearer token in Authorization header, got %q", got)
		}

		var body gmailSendRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body.Raw == "" {
			t.Error("expected non-empty raw message")
		}

		// Verify the raw message is valid unpadded base64url and decodes
		// to a well-formed RFC 2822 message.
		if strings.ContainsRune(body.Raw, '=') {
			t.Error("raw message contains padding; Gmail API expects unpadded base64url")
		}
		decoded, err := base64.RawURLEncoding.DecodeString(body.Raw)
		if err != nil {
			t.Fatalf("raw message is not valid base64url: %v", err)
		}
		msg := string(decoded)
		if !strings.Contains(msg, "To: user@example.com") {
			t.Error("decoded message missing To header")
		}
		if !strings.Contains(msg, "Subject: Test Subject") {
			t.Error("decoded message missing Subject header")
		}
		if !strings.Contains(msg, "Hello, world!") {
			t.Error("decoded message missing body text")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(gmailSendResponse{
			ID:       "msg-123",
			ThreadID: "thread-456",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL, "")
	action := &sendEmailAction{conn: conn}

	params, _ := json.Marshal(sendEmailParams{
		To:      "user@example.com",
		Subject: "Test Subject",
		Body:    "Hello, world!",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.send_email",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]string
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data["id"] != "msg-123" {
		t.Errorf("expected id 'msg-123', got %q", data["id"])
	}
	if data["thread_id"] != "thread-456" {
		t.Errorf("expected thread_id 'thread-456', got %q", data["thread_id"])
	}
}

func TestSendEmail_MissingTo(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &sendEmailAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"subject": "Test",
		"body":    "Hello",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.send_email",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing to")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestSendEmail_MissingSubject(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &sendEmailAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"to":   "user@example.com",
		"body": "Hello",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.send_email",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing subject")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestSendEmail_MissingBody(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &sendEmailAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"to":      "user@example.com",
		"subject": "Test",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.send_email",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing body")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestSendEmail_AuthFailure(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"code":    401,
				"message": "Invalid Credentials",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL, "")
	action := &sendEmailAction{conn: conn}

	params, _ := json.Marshal(sendEmailParams{
		To:      "user@example.com",
		Subject: "Test",
		Body:    "Hello",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.send_email",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for auth failure")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError, got: %T (%v)", err, err)
	}
}

func TestSendEmail_RateLimit(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "60")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL, "")
	action := &sendEmailAction{conn: conn}

	params, _ := json.Marshal(sendEmailParams{
		To:      "user@example.com",
		Subject: "Test",
		Body:    "Hello",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.send_email",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for rate limit")
	}
	if !connectors.IsRateLimitError(err) {
		t.Errorf("expected RateLimitError, got: %T", err)
	}
}

func TestSendEmail_HeaderInjectionTo(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &sendEmailAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"to":      "user@example.com\r\nBcc: attacker@evil.com",
		"subject": "Test",
		"body":    "Hello",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.send_email",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for newline in to")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestSendEmail_HeaderInjectionSubject(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &sendEmailAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"to":      "user@example.com",
		"subject": "Test\r\nBcc: attacker@evil.com",
		"body":    "Hello",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.send_email",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for newline in subject")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestSendEmail_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &sendEmailAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.send_email",
		Parameters:  []byte(`{invalid`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}
