package protonmail

import (
	"encoding/json"
	"fmt"
	"net/smtp"
	"strings"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestSendEmail_Success(t *testing.T) {
	t.Parallel()

	var capturedFrom string
	var capturedTo []string
	var capturedMsg []byte

	conn := New()
	action := &sendEmailAction{
		conn: conn,
		sendFunc: func(_ string, _ smtp.Auth, from string, to []string, msg []byte) error {
			capturedFrom = from
			capturedTo = to
			capturedMsg = msg
			return nil
		},
	}

	params, _ := json.Marshal(sendEmailParams{
		To:      []string{"alice@example.com"},
		Subject: "Test Subject",
		Body:    "Hello, Alice!",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "protonmail.send_email",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if capturedFrom != "user@proton.me" {
		t.Errorf("expected from 'user@proton.me', got %q", capturedFrom)
	}
	if len(capturedTo) != 1 || capturedTo[0] != "alice@example.com" {
		t.Errorf("expected to ['alice@example.com'], got %v", capturedTo)
	}
	if !strings.Contains(string(capturedMsg), "Subject: Test Subject") {
		t.Errorf("expected message to contain subject, got %q", string(capturedMsg))
	}
	if !strings.Contains(string(capturedMsg), "Hello, Alice!") {
		t.Errorf("expected message to contain body, got %q", string(capturedMsg))
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data["status"] != "sent" {
		t.Errorf("expected status 'sent', got %q", data["status"])
	}
}

func TestSendEmail_WithCcBcc(t *testing.T) {
	t.Parallel()

	var capturedTo []string
	var capturedMsg []byte

	conn := New()
	action := &sendEmailAction{
		conn: conn,
		sendFunc: func(_ string, _ smtp.Auth, _ string, to []string, msg []byte) error {
			capturedTo = to
			capturedMsg = msg
			return nil
		},
	}

	params, _ := json.Marshal(sendEmailParams{
		To:      []string{"alice@example.com"},
		Cc:      []string{"bob@example.com"},
		Bcc:     []string{"charlie@example.com"},
		Subject: "Test",
		Body:    "Hello",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "protonmail.send_email",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// All recipients should be in the SMTP envelope.
	if len(capturedTo) != 3 {
		t.Fatalf("expected 3 recipients, got %d: %v", len(capturedTo), capturedTo)
	}

	// Cc header should be present, Bcc should not.
	msgStr := string(capturedMsg)
	if !strings.Contains(msgStr, "Cc: bob@example.com") {
		t.Error("expected Cc header in message")
	}
	// BCC should NOT appear in headers.
	if strings.Contains(msgStr, "Bcc:") {
		t.Error("Bcc should not appear in message headers")
	}
}

func TestSendEmail_WithReplyTo(t *testing.T) {
	t.Parallel()

	var capturedMsg []byte

	conn := New()
	action := &sendEmailAction{
		conn: conn,
		sendFunc: func(_ string, _ smtp.Auth, _ string, _ []string, msg []byte) error {
			capturedMsg = msg
			return nil
		},
	}

	params, _ := json.Marshal(sendEmailParams{
		To:      []string{"alice@example.com"},
		Subject: "Test",
		Body:    "Hello",
		ReplyTo: "noreply@example.com",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "protonmail.send_email",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(string(capturedMsg), "Reply-To: noreply@example.com") {
		t.Error("expected Reply-To header in message")
	}
}

func TestSendEmail_HTMLContentType(t *testing.T) {
	t.Parallel()

	var capturedMsg []byte

	conn := New()
	action := &sendEmailAction{
		conn: conn,
		sendFunc: func(_ string, _ smtp.Auth, _ string, _ []string, msg []byte) error {
			capturedMsg = msg
			return nil
		},
	}

	params, _ := json.Marshal(sendEmailParams{
		To:          []string{"alice@example.com"},
		Subject:     "Test",
		Body:        "<h1>Hello</h1>",
		ContentType: "text/html",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "protonmail.send_email",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(string(capturedMsg), "Content-Type: text/html") {
		t.Error("expected text/html content type in message")
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
		ActionType:  "protonmail.send_email",
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

	params, _ := json.Marshal(map[string]any{
		"to":   []string{"alice@example.com"},
		"body": "Hello",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "protonmail.send_email",
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

	params, _ := json.Marshal(map[string]any{
		"to":      []string{"alice@example.com"},
		"subject": "Test",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "protonmail.send_email",
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

func TestSendEmail_InvalidContentType(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &sendEmailAction{conn: conn}

	params, _ := json.Marshal(map[string]any{
		"to":           []string{"alice@example.com"},
		"subject":      "Test",
		"body":         "Hello",
		"content_type": "application/json",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "protonmail.send_email",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid content_type")
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
		ActionType:  "protonmail.send_email",
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

func TestSendEmail_SMTPError(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &sendEmailAction{
		conn: conn,
		sendFunc: func(_ string, _ smtp.Auth, _ string, _ []string, _ []byte) error {
			return fmt.Errorf("connection refused")
		},
	}

	params, _ := json.Marshal(sendEmailParams{
		To:      []string{"alice@example.com"},
		Subject: "Test",
		Body:    "Hello",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "protonmail.send_email",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for SMTP failure")
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError, got: %T (%v)", err, err)
	}
}

func TestBuildMessage(t *testing.T) {
	t.Parallel()

	msg := buildMessage("sender@proton.me", sendEmailParams{
		To:          []string{"alice@example.com", "bob@example.com"},
		Cc:          []string{"charlie@example.com"},
		Subject:     "Hello World",
		Body:        "This is the body.",
		ContentType: "text/plain",
		ReplyTo:     "noreply@proton.me",
	})

	msgStr := string(msg)

	checks := []struct {
		label    string
		contains string
	}{
		{"From header", "From: sender@proton.me"},
		{"To header", "To: alice@example.com, bob@example.com"},
		{"Cc header", "Cc: charlie@example.com"},
		{"Reply-To header", "Reply-To: noreply@proton.me"},
		{"Subject header", "Subject: Hello World"},
		{"Content-Type", "Content-Type: text/plain; charset=UTF-8"},
		{"MIME-Version", "MIME-Version: 1.0"},
		{"Body", "This is the body."},
	}

	for _, c := range checks {
		if !strings.Contains(msgStr, c.contains) {
			t.Errorf("expected %s in message: %q", c.label, c.contains)
		}
	}
}

func TestSendEmail_InvalidEmailAddress(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &sendEmailAction{conn: conn}

	params, _ := json.Marshal(map[string]any{
		"to":      []string{"not-an-email"},
		"subject": "Test",
		"body":    "Hello",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "protonmail.send_email",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid email address")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestSendEmail_HeaderInjectionPrevented(t *testing.T) {
	t.Parallel()

	var capturedMsg []byte

	conn := New()
	action := &sendEmailAction{
		conn: conn,
		sendFunc: func(_ string, _ smtp.Auth, _ string, _ []string, msg []byte) error {
			capturedMsg = msg
			return nil
		},
	}

	params, _ := json.Marshal(sendEmailParams{
		To:      []string{"alice@example.com"},
		Subject: "Test\r\nBcc: attacker@evil.com",
		Body:    "Hello",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "protonmail.send_email",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	msgStr := string(capturedMsg)
	// The injected \r\n should be stripped, so "Bcc:" should not appear
	// as a standalone header line (it may appear within the sanitized subject).
	lines := strings.Split(msgStr, "\r\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "Bcc:") {
			t.Error("header injection: injected Bcc header found as separate line")
		}
	}
	// Verify the subject was sanitized (newlines removed).
	for _, line := range lines {
		if strings.HasPrefix(line, "Subject:") {
			if strings.Contains(line, "\r") || strings.Contains(line, "\n") {
				t.Error("header injection: subject contains newlines")
			}
			break
		}
	}
}

func TestSanitizeHeaderValue(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected string
	}{
		{"normal subject", "normal subject"},
		{"inject\r\nBcc: evil@test.com", "injectBcc: evil@test.com"},
		{"inject\nX-Header: bad", "injectX-Header: bad"},
		{"clean", "clean"},
	}

	for _, tt := range tests {
		got := sanitizeHeaderValue(tt.input)
		if got != tt.expected {
			t.Errorf("sanitizeHeaderValue(%q) = %q, want %q", tt.input, got, tt.expected)
		}
	}
}

func TestSendEmail_DefaultContentType(t *testing.T) {
	t.Parallel()

	var capturedMsg []byte

	conn := New()
	action := &sendEmailAction{
		conn: conn,
		sendFunc: func(_ string, _ smtp.Auth, _ string, _ []string, msg []byte) error {
			capturedMsg = msg
			return nil
		},
	}

	// Don't specify content_type; it should default to text/plain.
	params, _ := json.Marshal(map[string]any{
		"to":      []string{"alice@example.com"},
		"subject": "Test",
		"body":    "Hello",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "protonmail.send_email",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(string(capturedMsg), "Content-Type: text/plain") {
		t.Error("expected default content type text/plain")
	}
}
