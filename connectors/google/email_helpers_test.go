package google

import (
	"encoding/base64"
	"strings"
	"testing"
)

func TestBuildGmailRaw_HTMLMultipartStructure(t *testing.T) {
	t.Parallel()

	htmlBody := "<p>Hello <strong>world</strong></p>"
	raw := buildGmailRaw("a@b.com", "Subj", htmlBody, true, nil)
	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	msg := string(decoded)
	if !strings.Contains(msg, "MIME-Version: 1.0\r\n") {
		t.Error("expected MIME-Version header")
	}
	if !strings.Contains(msg, "Content-Type: multipart/alternative; boundary=") {
		t.Error("expected multipart/alternative")
	}
	if !strings.Contains(msg, "Content-Type: text/plain; charset=\"UTF-8\"") {
		t.Error("expected text/plain part")
	}
	if !strings.Contains(msg, "Content-Type: text/html; charset=\"UTF-8\"") {
		t.Error("expected text/html part")
	}
	if !strings.Contains(msg, htmlBody) {
		t.Error("expected HTML body in message")
	}
	if !strings.Contains(msg, "Hello world") {
		t.Error("expected plain fallback to strip tags")
	}
}

func TestBuildGmailRaw_PlaintextOnly(t *testing.T) {
	t.Parallel()

	body := "Line1\nLine2"
	raw := buildGmailRaw("x@y.com", "S", body, false, nil)
	decoded, err := base64.RawURLEncoding.DecodeString(raw)
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	msg := string(decoded)
	if strings.Contains(msg, "multipart/alternative") {
		t.Error("did not expect multipart for plaintext mode")
	}
	if !strings.Contains(msg, "Content-Type: text/plain; charset=\"UTF-8\"") {
		t.Error("expected single text/plain")
	}
	if !strings.HasSuffix(strings.TrimSpace(msg), body) {
		t.Errorf("body not at end: %q", msg)
	}
}

func TestHtmlToPlainText(t *testing.T) {
	t.Parallel()

	got := htmlToPlainText("<p>a &amp; b</p>")
	if got != "a & b" {
		t.Errorf("got %q", got)
	}
}
