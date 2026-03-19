package slack

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"testing"
	"time"
)

const testSigningSecret = "test-signing-secret-abc123"

// signPayload creates valid Slack signature headers for a test payload.
func signPayload(t *testing.T, payload []byte, signingSecret string) http.Header {
	t.Helper()
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	sigBase := fmt.Sprintf("v0:%s:%s", ts, string(payload))
	mac := hmac.New(sha256.New, []byte(signingSecret))
	mac.Write([]byte(sigBase))
	sig := "v0=" + hex.EncodeToString(mac.Sum(nil))

	headers := http.Header{}
	headers.Set("X-Slack-Request-Timestamp", ts)
	headers.Set("X-Slack-Signature", sig)
	return headers
}

func TestVerifyAndParseEvent_URLVerification(t *testing.T) {
	conn := New()
	payload := []byte(`{"type":"url_verification","challenge":"test-challenge-123","token":"deprecated"}`)
	headers := signPayload(t, payload, testSigningSecret)

	envelope, err := conn.VerifyAndParseEvent(payload, headers, testSigningSecret)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if envelope.Challenge != "test-challenge-123" {
		t.Errorf("expected challenge %q, got %q", "test-challenge-123", envelope.Challenge)
	}
	if envelope.Event != nil {
		t.Error("expected nil event for url_verification")
	}
}

func TestVerifyAndParseEvent_MessageIM(t *testing.T) {
	conn := New()
	payload := []byte(`{
		"type": "event_callback",
		"team_id": "T01234",
		"api_app_id": "A01234",
		"event": {
			"type": "message",
			"channel_type": "im",
			"channel": "D01234",
			"user": "U01234",
			"text": "hello from DM",
			"ts": "1710000000.123456"
		},
		"event_id": "Ev01234",
		"event_time": 1710000000
	}`)
	headers := signPayload(t, payload, testSigningSecret)

	envelope, err := conn.VerifyAndParseEvent(payload, headers, testSigningSecret)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if envelope.Challenge != "" {
		t.Error("expected empty challenge for event_callback")
	}
	if envelope.Event == nil {
		t.Fatal("expected non-nil event")
	}

	event := envelope.Event
	if event.ConnectorID != "slack" {
		t.Errorf("expected connector_id %q, got %q", "slack", event.ConnectorID)
	}
	if event.EventType != "message.im" {
		t.Errorf("expected event_type %q, got %q", "message.im", event.EventType)
	}
	if event.EventID != "Ev01234" {
		t.Errorf("expected event_id %q, got %q", "Ev01234", event.EventID)
	}
	if event.TeamID != "T01234" {
		t.Errorf("expected team_id %q, got %q", "T01234", event.TeamID)
	}

	// Verify typed payload.
	var msg IMMessagePayload
	if err := json.Unmarshal(event.Payload, &msg); err != nil {
		t.Fatalf("failed to unmarshal payload: %v", err)
	}
	if msg.User != "U01234" {
		t.Errorf("expected user %q, got %q", "U01234", msg.User)
	}
	if msg.Channel != "D01234" {
		t.Errorf("expected channel %q, got %q", "D01234", msg.Channel)
	}
	if msg.Text != "hello from DM" {
		t.Errorf("expected text %q, got %q", "hello from DM", msg.Text)
	}
	if msg.Timestamp != "1710000000.123456" {
		t.Errorf("expected ts %q, got %q", "1710000000.123456", msg.Timestamp)
	}
}

func TestVerifyAndParseEvent_NonIMMessage(t *testing.T) {
	conn := New()
	payload := []byte(`{
		"type": "event_callback",
		"team_id": "T01234",
		"event": {
			"type": "message",
			"channel_type": "channel",
			"channel": "C01234",
			"user": "U01234",
			"text": "hello from channel",
			"ts": "1710000000.123456"
		},
		"event_id": "Ev01234",
		"event_time": 1710000000
	}`)
	headers := signPayload(t, payload, testSigningSecret)

	envelope, err := conn.VerifyAndParseEvent(payload, headers, testSigningSecret)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Non-IM messages get the raw event type, not "message.im".
	if envelope.Event.EventType != "message" {
		t.Errorf("expected event_type %q, got %q", "message", envelope.Event.EventType)
	}
}

func TestVerifyAndParseEvent_InvalidSignature(t *testing.T) {
	conn := New()
	payload := []byte(`{"type":"url_verification","challenge":"test"}`)
	headers := signPayload(t, payload, testSigningSecret)
	// Tamper with the signature.
	headers.Set("X-Slack-Signature", "v0=invalid")

	_, err := conn.VerifyAndParseEvent(payload, headers, testSigningSecret)
	if err == nil {
		t.Fatal("expected error for invalid signature")
	}
}

func TestVerifyAndParseEvent_MissingHeaders(t *testing.T) {
	conn := New()
	payload := []byte(`{"type":"url_verification","challenge":"test"}`)

	_, err := conn.VerifyAndParseEvent(payload, http.Header{}, testSigningSecret)
	if err == nil {
		t.Fatal("expected error for missing headers")
	}
}

func TestVerifyAndParseEvent_ExpiredTimestamp(t *testing.T) {
	conn := New()
	payload := []byte(`{"type":"url_verification","challenge":"test"}`)

	// Use a timestamp from 10 minutes ago.
	oldTS := strconv.FormatInt(time.Now().Add(-10*time.Minute).Unix(), 10)
	sigBase := fmt.Sprintf("v0:%s:%s", oldTS, string(payload))
	mac := hmac.New(sha256.New, []byte(testSigningSecret))
	mac.Write([]byte(sigBase))
	sig := "v0=" + hex.EncodeToString(mac.Sum(nil))

	headers := http.Header{}
	headers.Set("X-Slack-Request-Timestamp", oldTS)
	headers.Set("X-Slack-Signature", sig)

	_, err := conn.VerifyAndParseEvent(payload, headers, testSigningSecret)
	if err == nil {
		t.Fatal("expected error for expired timestamp")
	}
}

func TestVerifyAndParseEvent_UnsupportedEnvelopeType(t *testing.T) {
	conn := New()
	payload := []byte(`{"type":"unknown_type"}`)
	headers := signPayload(t, payload, testSigningSecret)

	_, err := conn.VerifyAndParseEvent(payload, headers, testSigningSecret)
	if err == nil {
		t.Fatal("expected error for unsupported envelope type")
	}
}

func TestMapSlackEventType(t *testing.T) {
	tests := []struct {
		name     string
		inner    slackInnerEvent
		expected string
	}{
		{
			name:     "message in IM",
			inner:    slackInnerEvent{Type: "message", ChannelType: "im"},
			expected: "message.im",
		},
		{
			name:     "message in channel",
			inner:    slackInnerEvent{Type: "message", ChannelType: "channel"},
			expected: "message",
		},
		{
			name:     "reaction_added",
			inner:    slackInnerEvent{Type: "reaction_added"},
			expected: "reaction_added",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := mapSlackEventType(tt.inner)
			if got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}
