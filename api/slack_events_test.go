package api

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/supersuit-tech/permission-slip-web/connectors"
	_ "github.com/supersuit-tech/permission-slip-web/connectors/all"
)

const testSlackSecret = "test-signing-secret-abc123"

// slackSign creates valid Slack signature headers for a test payload.
func slackSign(t *testing.T, payload string, secret string) (string, string) {
	t.Helper()
	ts := strconv.FormatInt(time.Now().Unix(), 10)
	sigBase := fmt.Sprintf("v0:%s:%s", ts, payload)
	mac := hmac.New(sha256.New, []byte(secret))
	mac.Write([]byte(sigBase))
	sig := "v0=" + hex.EncodeToString(mac.Sum(nil))
	return ts, sig
}

func newSlackEventDeps() *Deps {
	registry := connectors.NewRegistry()
	for _, c := range connectors.BuiltInConnectors() {
		registry.Register(c)
	}
	return &Deps{
		SlackSigningSecret: testSlackSecret,
		Connectors:         registry,
		EventBroker:        connectors.NewEventBroker(),
	}
}

func TestSlackEvents_URLVerification(t *testing.T) {
	deps := newSlackEventDeps()
	handler := handleSlackEvent(deps)

	payload := `{"type":"url_verification","challenge":"test-challenge-xyz","token":"deprecated"}`
	ts, sig := slackSign(t, payload, testSlackSecret)

	req := httptest.NewRequest("POST", "/api/webhooks/slack/events", strings.NewReader(payload))
	req.Header.Set("X-Slack-Request-Timestamp", ts)
	req.Header.Set("X-Slack-Signature", sig)

	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp map[string]string
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if resp["challenge"] != "test-challenge-xyz" {
		t.Errorf("expected challenge %q, got %q", "test-challenge-xyz", resp["challenge"])
	}
}

func TestSlackEvents_ValidEvent(t *testing.T) {
	deps := newSlackEventDeps()

	var dispatched *connectors.Event
	deps.EventBroker.Subscribe("message.im", connectors.EventHandlerFunc(func(_ context.Context, event *connectors.Event) error {
		dispatched = event
		return nil
	}))

	handler := handleSlackEvent(deps)

	payload := `{
		"type": "event_callback",
		"team_id": "T01234",
		"event": {
			"type": "message",
			"channel_type": "im",
			"channel": "D01234",
			"user": "U01234",
			"text": "hello",
			"ts": "1710000000.123456"
		},
		"event_id": "Ev01234",
		"event_time": 1710000000
	}`
	ts, sig := slackSign(t, payload, testSlackSecret)

	req := httptest.NewRequest("POST", "/api/webhooks/slack/events", strings.NewReader(payload))
	req.Header.Set("X-Slack-Request-Timestamp", ts)
	req.Header.Set("X-Slack-Signature", sig)

	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	if dispatched == nil {
		t.Fatal("expected event to be dispatched")
	}
	if dispatched.EventType != "message.im" {
		t.Errorf("expected event type %q, got %q", "message.im", dispatched.EventType)
	}
}

func TestSlackEvents_InvalidSignature(t *testing.T) {
	deps := newSlackEventDeps()
	handler := handleSlackEvent(deps)

	payload := `{"type":"event_callback","event":{}}`
	req := httptest.NewRequest("POST", "/api/webhooks/slack/events", strings.NewReader(payload))
	req.Header.Set("X-Slack-Request-Timestamp", strconv.FormatInt(time.Now().Unix(), 10))
	req.Header.Set("X-Slack-Signature", "v0=invalid")

	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestSlackEvents_MissingSigningSecret(t *testing.T) {
	deps := newSlackEventDeps()
	deps.SlackSigningSecret = ""
	handler := handleSlackEvent(deps)

	req := httptest.NewRequest("POST", "/api/webhooks/slack/events", strings.NewReader("{}"))
	rr := httptest.NewRecorder()
	handler(rr, req)

	if rr.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected 503, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestSlackEvents_Deduplication(t *testing.T) {
	deps := newSlackEventDeps()
	dispatchCount := 0
	deps.EventBroker.Subscribe("message.im", connectors.EventHandlerFunc(func(_ context.Context, _ *connectors.Event) error {
		dispatchCount++
		return nil
	}))

	handler := handleSlackEvent(deps)

	payload := `{
		"type": "event_callback",
		"team_id": "T01234",
		"event": {
			"type": "message",
			"channel_type": "im",
			"channel": "D01234",
			"user": "U01234",
			"text": "hello",
			"ts": "1710000000.123456"
		},
		"event_id": "Ev_dedup_test",
		"event_time": 1710000000
	}`

	// Send the same event twice.
	for i := 0; i < 2; i++ {
		ts, sig := slackSign(t, payload, testSlackSecret)
		req := httptest.NewRequest("POST", "/api/webhooks/slack/events", strings.NewReader(payload))
		req.Header.Set("X-Slack-Request-Timestamp", ts)
		req.Header.Set("X-Slack-Signature", sig)

		rr := httptest.NewRecorder()
		handler(rr, req)

		if rr.Code != http.StatusOK {
			t.Fatalf("request %d: expected 200, got %d", i, rr.Code)
		}
	}

	if dispatchCount != 1 {
		t.Errorf("expected event to be dispatched once, got %d", dispatchCount)
	}
}

func TestEventDeduplicator_TTLExpiry(t *testing.T) {
	dedup := newEventDeduplicator(10 * time.Millisecond)

	if dedup.isDuplicate("ev1") {
		t.Fatal("first occurrence should not be duplicate")
	}
	if !dedup.isDuplicate("ev1") {
		t.Fatal("second occurrence should be duplicate")
	}

	// Wait for TTL to expire.
	time.Sleep(20 * time.Millisecond)

	if dedup.isDuplicate("ev1") {
		t.Fatal("after TTL expiry, should not be duplicate")
	}
}
