package notify

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSMSSender_Name(t *testing.T) {
	t.Parallel()
	s := NewSMSSender(SMSConfig{})
	if s.Name() != "sms" {
		t.Fatalf("expected name %q, got %q", "sms", s.Name())
	}
}

func TestSMSSender_Send_Success(t *testing.T) {
	t.Parallel()

	var capturedBody string
	var capturedAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Capture basic auth.
		user, pass, ok := r.BasicAuth()
		if ok {
			capturedAuth = user + ":" + pass
		}

		body, _ := io.ReadAll(r.Body)
		capturedBody = string(body)
		w.WriteHeader(http.StatusCreated)
		_, _ = w.Write([]byte(`{"sid":"SM123"}`))
	}))
	defer server.Close()

	sender := newTestSMSSender(SMSConfig{
		AccountSID: "AC_test_sid",
		AuthToken:  "test_token",
		FromNumber: "+15550001111",
	}, server.URL)

	phone := "+15559998888"
	err := sender.Send(context.Background(), testApproval(), Recipient{
		UserID:   "user-001",
		Username: "alice",
		Phone:    &phone,
	})

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify basic auth credentials.
	if capturedAuth != "AC_test_sid:test_token" {
		t.Errorf("expected auth %q, got %q", "AC_test_sid:test_token", capturedAuth)
	}

	// Verify form fields.
	if !strings.Contains(capturedBody, "To=%2B15559998888") {
		t.Errorf("expected To phone in body, got: %s", capturedBody)
	}
	if !strings.Contains(capturedBody, "From=%2B15550001111") {
		t.Errorf("expected From phone in body, got: %s", capturedBody)
	}
	if !strings.Contains(capturedBody, "Body=") {
		t.Errorf("expected Body in form data, got: %s", capturedBody)
	}
}

func TestSMSSender_Send_NoPhone(t *testing.T) {
	t.Parallel()

	sender := NewSMSSender(SMSConfig{
		AccountSID: "AC_test",
		AuthToken:  "token",
		FromNumber: "+15550001111",
	})

	// nil phone — should skip without error.
	err := sender.Send(context.Background(), testApproval(), Recipient{
		UserID:   "user-001",
		Username: "alice",
		Phone:    nil,
	})
	if err != nil {
		t.Fatalf("expected nil error for nil phone, got: %v", err)
	}

	// Empty phone — should skip without error.
	empty := ""
	err = sender.Send(context.Background(), testApproval(), Recipient{
		UserID:   "user-001",
		Username: "alice",
		Phone:    &empty,
	})
	if err != nil {
		t.Fatalf("expected nil error for empty phone, got: %v", err)
	}
}

func TestSMSSender_Send_TwilioError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte(`{"code":21211,"message":"Invalid 'To' Phone Number"}`))
	}))
	defer server.Close()

	sender := newTestSMSSender(SMSConfig{
		AccountSID: "AC_test",
		AuthToken:  "token",
		FromNumber: "+15550001111",
	}, server.URL)

	phone := "+15559998888"
	err := sender.Send(context.Background(), testApproval(), Recipient{
		UserID:   "user-001",
		Username: "alice",
		Phone:    &phone,
	})

	if err == nil {
		t.Fatal("expected error for Twilio 400 response")
	}
	if !strings.Contains(err.Error(), "twilio returned 400") {
		t.Errorf("expected error to mention status 400, got: %v", err)
	}
}

func TestSMSSender_Send_ServerError(t *testing.T) {
	t.Parallel()

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte(`Internal Server Error`))
	}))
	defer server.Close()

	sender := newTestSMSSender(SMSConfig{
		AccountSID: "AC_test",
		AuthToken:  "token",
		FromNumber: "+15550001111",
	}, server.URL)

	phone := "+15559998888"
	err := sender.Send(context.Background(), testApproval(), Recipient{
		UserID: "user-001",
		Phone:  &phone,
	})

	if err == nil {
		t.Fatal("expected error for 500 response")
	}
	if !strings.Contains(err.Error(), "twilio returned 500") {
		t.Errorf("expected error to mention status 500, got: %v", err)
	}
}

func TestSMSSender_ContextCancellation(t *testing.T) {
	t.Parallel()

	// Use a channel-based block instead of time.Sleep so the server handler
	// cleans up immediately when the test ends (no lingering goroutines).
	blocker := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		<-blocker
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()
	defer close(blocker) // Unblock handler so server.Close() can finish.

	sender := newTestSMSSender(SMSConfig{
		AccountSID: "AC_test",
		AuthToken:  "token",
		FromNumber: "+15550001111",
	}, server.URL)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately.

	phone := "+15559998888"
	err := sender.Send(ctx, testApproval(), Recipient{
		UserID: "user-001",
		Phone:  &phone,
	})

	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

// ── FormatSMSBody tests ─────────────────────────────────────────────────────

func TestFormatSMSBody_WithAgentName(t *testing.T) {
	t.Parallel()
	a := Approval{
		AgentID:     42,
		AgentName:   "My AI Assistant",
		Action:      json.RawMessage(`{"type":"email.send"}`),
		ApprovalURL: "https://app.example.com/approve/appr_xyz",
	}
	body := formatSMSBody(a)

	if !strings.Contains(body, "My AI Assistant") {
		t.Errorf("expected agent name in body, got: %s", body)
	}
	if !strings.Contains(body, "email.send") {
		t.Errorf("expected action type in body, got: %s", body)
	}
	if !strings.Contains(body, "https://app.example.com/approve/appr_xyz") {
		t.Errorf("expected approval URL in body, got: %s", body)
	}
}

func TestFormatSMSBody_NoAgentName(t *testing.T) {
	t.Parallel()
	a := Approval{
		AgentID:     99,
		AgentName:   "",
		Action:      json.RawMessage(`{"type":"slack.post"}`),
		ApprovalURL: "https://example.com/approve/appr_123",
	}
	body := formatSMSBody(a)

	if !strings.Contains(body, "Agent #99") {
		t.Errorf("expected fallback agent name, got: %s", body)
	}
}

func TestFormatSMSBody_NoApprovalURL(t *testing.T) {
	t.Parallel()
	a := Approval{
		AgentID:   42,
		AgentName: "Bot",
		Action:    json.RawMessage(`{"type":"test"}`),
	}
	body := formatSMSBody(a)

	if strings.Contains(body, "Approve:") {
		t.Errorf("expected no Approve URL in body, got: %s", body)
	}
}

// ── BuildSenders config tests ───────────────────────────────────────────────

func TestBuildSenders_SMSEnabled(t *testing.T) {
	t.Parallel()
	cfg := Config{
		TwilioAccountSID: "AC_test",
		TwilioAuthToken:  "token",
		TwilioFromNumber: "+15550001111",
	}
	senders := cfg.BuildSenders()
	if len(senders) != 1 {
		t.Fatalf("expected 1 sender, got %d", len(senders))
	}
	if senders[0].Name() != "sms" {
		t.Errorf("expected sender name %q, got %q", "sms", senders[0].Name())
	}
}

func TestBuildSenders_SMSPartialConfig(t *testing.T) {
	t.Parallel()
	cfg := Config{
		TwilioAccountSID: "AC_test",
		// Missing AuthToken and FromNumber.
	}
	senders := cfg.BuildSenders()
	if len(senders) != 0 {
		t.Fatalf("expected 0 senders for partial config, got %d", len(senders))
	}
}

// ── Helpers ─────────────────────────────────────────────────────────────────

// newTestSMSSender creates an SMSSender that talks to a local test server
// instead of the real Twilio API by overriding the sender's baseURL to
// point at the provided testServerURL.
func newTestSMSSender(cfg SMSConfig, testServerURL string) *SMSSender {
	s := NewSMSSender(cfg)
	s.baseURL = testServerURL
	return s
}
