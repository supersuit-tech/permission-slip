package notify

import (
	"context"
	"strings"
	"testing"
)

func TestSMTP_Name(t *testing.T) {
	t.Parallel()
	sender := NewSMTPSender("smtp.example.com", "587", "user", "pass", "from@example.com")
	if sender.Name() != "email" {
		t.Errorf("expected name=email, got: %s", sender.Name())
	}
}

func TestSMTP_NoEmail(t *testing.T) {
	t.Parallel()
	sender := NewSMTPSender("smtp.example.com", "587", "", "", "from@example.com")

	recipient := Recipient{UserID: "user-001", Username: "alice"}
	err := sender.Send(context.Background(), sendGridTestApproval(), recipient)
	if err == nil {
		t.Fatal("expected error for nil email")
	}
}

func TestSMTP_EmptyEmail(t *testing.T) {
	t.Parallel()
	sender := NewSMTPSender("smtp.example.com", "587", "", "", "from@example.com")

	empty := ""
	recipient := Recipient{UserID: "user-001", Username: "alice", Email: &empty}
	err := sender.Send(context.Background(), sendGridTestApproval(), recipient)
	if err == nil {
		t.Fatal("expected error for empty email")
	}
}

func TestSMTP_HeaderInjection(t *testing.T) {
	t.Parallel()
	sender := NewSMTPSender("smtp.example.com", "587", "", "", "from@example.com")

	tests := []struct {
		name  string
		email string
	}{
		{"newline injection", "victim@example.com\r\nBcc: attacker@evil.com"},
		{"bare LF", "victim@example.com\nBcc: attacker@evil.com"},
		{"bare CR", "victim@example.com\rBcc: attacker@evil.com"},
	}
	for _, tt := range tests {
		email := tt.email
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			recipient := Recipient{UserID: "user-001", Username: "alice", Email: &email}
			err := sender.Send(context.Background(), sendGridTestApproval(), recipient)
			if err == nil {
				t.Fatal("expected error for email with injected headers")
			}
			if !strings.Contains(err.Error(), "invalid") {
				t.Errorf("expected 'invalid' in error, got: %s", err)
			}
		})
	}
}

func TestSMTP_InvalidEmailAddress(t *testing.T) {
	t.Parallel()
	sender := NewSMTPSender("smtp.example.com", "587", "", "", "from@example.com")

	bad := "not-an-email"
	recipient := Recipient{UserID: "user-001", Username: "alice", Email: &bad}
	err := sender.Send(context.Background(), sendGridTestApproval(), recipient)
	if err == nil {
		t.Fatal("expected error for invalid email address")
	}
}


func TestSMTP_ContextCancelled(t *testing.T) {
	t.Parallel()

	// Use an unreachable host so the SMTP dial will block until context cancels.
	sender := NewSMTPSender("192.0.2.1", "587", "", "", "from@example.com")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := sender.Send(ctx, sendGridTestApproval(), sendGridTestRecipient())
	if err == nil {
		t.Fatal("expected error for cancelled context")
	}
}

func TestBuildMIMEMessage(t *testing.T) {
	t.Parallel()

	msg := buildMIMEMessage(
		"sender@example.com",
		"recipient@example.com",
		"Test Subject",
		"Plain text body",
		"<html><body>HTML body</body></html>",
	)

	if !strings.Contains(msg, "From: sender@example.com") {
		t.Error("expected From header")
	}
	if !strings.Contains(msg, "To: recipient@example.com") {
		t.Error("expected To header")
	}
	if !strings.Contains(msg, "MIME-Version: 1.0") {
		t.Error("expected MIME-Version header")
	}
	if !strings.Contains(msg, "multipart/alternative") {
		t.Error("expected multipart/alternative content type")
	}
	if !strings.Contains(msg, "text/plain") {
		t.Error("expected text/plain content part")
	}
	if !strings.Contains(msg, "text/html") {
		t.Error("expected text/html content part")
	}
	if !strings.Contains(msg, "Plain text body") {
		t.Error("expected plain text body content")
	}
	if !strings.Contains(msg, "HTML body") {
		t.Error("expected HTML body content")
	}
}

func TestBuildMIMEMessage_RandomBoundary(t *testing.T) {
	t.Parallel()

	msg1 := buildMIMEMessage("a@b.com", "c@d.com", "subj", "plain", "html")
	msg2 := buildMIMEMessage("a@b.com", "c@d.com", "subj", "plain", "html")

	// Extract the boundary from each message's Content-Type header.
	b1 := extractBoundaryFromMsg(msg1)
	b2 := extractBoundaryFromMsg(msg2)

	if b1 == "" || b2 == "" {
		t.Fatal("failed to extract boundary from MIME message")
	}
	if b1 == b2 {
		t.Errorf("expected unique boundaries per message, got same: %s", b1)
	}
}

func extractBoundaryFromMsg(msg string) string {
	for _, line := range strings.Split(msg, "\r\n") {
		if strings.Contains(line, "boundary=") {
			// boundary="----=_PS_abc123..."
			idx := strings.Index(line, "boundary=\"")
			if idx == -1 {
				continue
			}
			rest := line[idx+len("boundary=\""):]
			end := strings.Index(rest, "\"")
			if end == -1 {
				continue
			}
			return rest[:end]
		}
	}
	return ""
}

func TestGenerateMIMEBoundary(t *testing.T) {
	t.Parallel()
	b := generateMIMEBoundary()
	if !strings.HasPrefix(b, "----=_PS_") {
		t.Errorf("expected boundary to start with ----=_PS_, got: %s", b)
	}
	// 16 random bytes = 32 hex chars, plus prefix
	if len(b) != len("----=_PS_")+32 {
		t.Errorf("expected boundary length %d, got %d: %s", len("----=_PS_")+32, len(b), b)
	}
}
