package api

import (
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestParseEd25519PublicKey_Valid(t *testing.T) {
	t.Parallel()

	pubKeySSH, _, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	pub, err := ParseEd25519PublicKey(pubKeySSH)
	if err != nil {
		t.Fatalf("parse key: %v", err)
	}
	if len(pub) != ed25519.PublicKeySize {
		t.Errorf("expected %d bytes, got %d", ed25519.PublicKeySize, len(pub))
	}
}

func TestParseEd25519PublicKey_InvalidFormat(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		key  string
	}{
		{"empty", ""},
		{"single field", "ssh-ed25519"},
		{"wrong type", "ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABgQC..."},
		{"invalid base64", "ssh-ed25519 !!!invalid!!!"},
		{"too short data", "ssh-ed25519 AAAA"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseEd25519PublicKey(tt.key)
			if err == nil {
				t.Errorf("expected error for key %q, got nil", tt.key)
			}
		})
	}
}

func TestFormatAndParseRoundtrip(t *testing.T) {
	t.Parallel()

	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	sshKey := FormatEd25519OpenSSH(pub)
	if !strings.HasPrefix(sshKey, "ssh-ed25519 ") {
		t.Errorf("expected ssh-ed25519 prefix, got %q", sshKey)
	}

	parsedPub, err := ParseEd25519PublicKey(sshKey)
	if err != nil {
		t.Fatalf("parse key: %v", err)
	}

	// Verify the round-tripped key can verify a signature.
	msg := []byte("test message")
	sig := ed25519.Sign(priv, msg)
	if !ed25519.Verify(parsedPub, msg, sig) {
		t.Error("signature verification failed after round-trip")
	}
}

func TestParseSignatureHeader_Valid(t *testing.T) {
	t.Parallel()

	header := `agent_id="42", algorithm="Ed25519", timestamp="1708000000", signature="dGVzdHNpZw"`
	sig, err := ParseSignatureHeader(header)
	if err != nil {
		t.Fatalf("parse header: %v", err)
	}
	if sig.AgentID != 42 {
		t.Errorf("expected agent_id 42, got %d", sig.AgentID)
	}
	if sig.Algorithm != "Ed25519" {
		t.Errorf("expected algorithm Ed25519, got %q", sig.Algorithm)
	}
	if sig.Timestamp != 1708000000 {
		t.Errorf("expected timestamp 1708000000, got %d", sig.Timestamp)
	}
	if len(sig.Signature) == 0 {
		t.Error("expected non-empty signature")
	}
}

func TestParseSignatureHeader_MissingFields(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		header string
	}{
		{"missing agent_id", `algorithm="Ed25519", timestamp="1708000000", signature="dGVzdA"`},
		{"missing algorithm", `agent_id="42", timestamp="1708000000", signature="dGVzdA"`},
		{"missing timestamp", `agent_id="42", algorithm="Ed25519", signature="dGVzdA"`},
		{"missing signature", `agent_id="42", algorithm="Ed25519", timestamp="1708000000"`},
		{"invalid agent_id", `agent_id="abc", algorithm="Ed25519", timestamp="1708000000", signature="dGVzdA"`},
		{"negative agent_id", `agent_id="-1", algorithm="Ed25519", timestamp="1708000000", signature="dGVzdA"`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseSignatureHeader(tt.header)
			if err == nil {
				t.Errorf("expected error for header %q, got nil", tt.header)
			}
		})
	}
}

func TestBuildCanonicalRequest(t *testing.T) {
	t.Parallel()

	canonical := BuildCanonicalRequest("POST", "/invite/PS-ABCD-EFGH", "", 1708000000, emptyBodyHash)
	parts := strings.Split(canonical, "\n")
	if len(parts) != 5 {
		t.Fatalf("expected 5 parts, got %d", len(parts))
	}
	if parts[0] != "POST" {
		t.Errorf("method: expected 'POST', got %q", parts[0])
	}
	if parts[1] != "/invite/PS-ABCD-EFGH" {
		t.Errorf("path: expected '/invite/PS-ABCD-EFGH', got %q", parts[1])
	}
	if parts[2] != "" {
		t.Errorf("query: expected empty, got %q", parts[2])
	}
	if parts[3] != "1708000000" {
		t.Errorf("timestamp: expected '1708000000', got %q", parts[3])
	}
	if parts[4] != emptyBodyHash {
		t.Errorf("body hash: expected empty body hash, got %q", parts[4])
	}
}

func TestBuildCanonicalRequest_WithQuery(t *testing.T) {
	t.Parallel()

	canonical := BuildCanonicalRequest("GET", "/v1/agents", "limit=10&offset=0", 1708000000, emptyBodyHash)
	parts := strings.Split(canonical, "\n")
	if parts[2] != "limit=10&offset=0" {
		t.Errorf("query: expected 'limit=10&offset=0', got %q", parts[2])
	}
}

func TestHashBody(t *testing.T) {
	t.Parallel()

	// Empty body.
	h := HashBody(nil)
	if h != emptyBodyHash {
		t.Errorf("expected empty body hash, got %q", h)
	}

	// Non-empty body should produce a different hash.
	h2 := HashBody([]byte(`{"test":"data"}`))
	if h2 == emptyBodyHash {
		t.Error("non-empty body should produce different hash than empty body")
	}

	// Deterministic.
	h3 := HashBody([]byte(`{"test":"data"}`))
	if h2 != h3 {
		t.Error("same input should produce same hash")
	}
}

func TestSignAndVerify_RoundTrip(t *testing.T) {
	t.Parallel()

	pubKeySSH, privKey, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	body := []byte(`{"request_id":"test-123","public_key":"` + pubKeySSH + `"}`)
	r := httptest.NewRequest("POST", "/invite/PS-ABCD-EFGH", nil)

	SignRequest(privKey, 0, r, body)

	sig, err := VerifyRegistrationSignature(pubKeySSH, r, body)
	if err != nil {
		t.Fatalf("verify signature: %v", err)
	}
	if sig == nil {
		t.Fatal("expected non-nil parsed signature")
	}
}

func TestVerifySignature_WrongKey(t *testing.T) {
	t.Parallel()

	pubKeySSH1, privKey1, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key 1: %v", err)
	}

	pubKeySSH2, _, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key 2: %v", err)
	}

	_ = pubKeySSH1 // Signed with key 1, verified with key 2 — should fail.

	body := []byte(`{"request_id":"test-456"}`)
	r := httptest.NewRequest("POST", "/invite/PS-XXXX-YYYY", nil)
	SignRequest(privKey1, 0, r, body)

	_, err = VerifyRegistrationSignature(pubKeySSH2, r, body)
	if err == nil {
		t.Error("expected error when verifying with wrong key")
	}
}

func TestVerifySignature_TamperedBody(t *testing.T) {
	t.Parallel()

	pubKeySSH, privKey, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	body := []byte(`{"request_id":"test-789"}`)
	r := httptest.NewRequest("POST", "/invite/PS-AAAA-BBBB", nil)
	SignRequest(privKey, 0, r, body)

	// Tamper with the body.
	tampered := []byte(`{"request_id":"test-XXX"}`)
	_, err = VerifyRegistrationSignature(pubKeySSH, r, tampered)
	if err == nil {
		t.Error("expected error when body is tampered")
	}
}

func TestVerifySignature_ExpiredTimestamp(t *testing.T) {
	t.Parallel()

	pubKeySSH, privKey, err := GenerateEd25519OpenSSHKey()
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}

	pubKey, err := ParseEd25519PublicKey(pubKeySSH)
	if err != nil {
		t.Fatalf("parse key: %v", err)
	}

	// Create a signature with a timestamp from 10 minutes ago.
	oldTimestamp := time.Now().Add(-10 * time.Minute).Unix()
	body := []byte(`{"test":"data"}`)
	bodyHash := HashBody(body)
	canonical := BuildCanonicalRequest("POST", "/invite/PS-TEST-CODE", "", oldTimestamp, bodyHash)
	sig := ed25519.Sign(privKey, []byte(canonical))

	r := httptest.NewRequest("POST", "/invite/PS-TEST-CODE", nil)
	r.Header.Set(signatureHeader, fmt.Sprintf(
		`agent_id="1", algorithm="Ed25519", timestamp="%d", signature="%s"`,
		oldTimestamp,
		base64.RawURLEncoding.EncodeToString(sig),
	))

	err = VerifyEd25519Signature(pubKey, &ParsedSignature{
		AgentID:   1,
		Algorithm: "Ed25519",
		Timestamp: oldTimestamp,
		Signature: sig,
	}, r, body)

	if err == nil {
		t.Error("expected error for expired timestamp")
	}
	if !errors.Is(err, ErrSigTimestampExpired) {
		t.Errorf("expected ErrSigTimestampExpired, got %q", err.Error())
	}
}

func TestNormalizeConfirmationCode(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected string
	}{
		{"XK7-M9P", "XK7M9P"},
		{"XK7M9P", "XK7M9P"},
		{"xk7-m9p", "XK7M9P"},
		{"xk7m9p", "XK7M9P"},
		{"Xk7-M9p", "XK7M9P"},
		{"---", ""},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := normalizeConfirmationCode(tt.input)
			if got != tt.expected {
				t.Errorf("normalizeConfirmationCode(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}
