package slack

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// Slack Events API envelope types.
const (
	slackEventTypeURLVerification = "url_verification"
	slackEventTypeEventCallback   = "event_callback"
)

// maxTimestampAge is the maximum age of a Slack event timestamp before it's
// rejected. Slack recommends rejecting requests older than 5 minutes to
// prevent replay attacks.
const maxTimestampAge = 5 * time.Minute

// slackEventEnvelope is the outer JSON structure of a Slack Events API payload.
type slackEventEnvelope struct {
	Type      string          `json:"type"`
	Challenge string          `json:"challenge"` // only for url_verification
	TeamID    string          `json:"team_id"`
	APIAppID  string          `json:"api_app_id"`
	Event     json.RawMessage `json:"event"`
	EventID   string          `json:"event_id"`
	EventTime int64           `json:"event_time"`
}

// slackInnerEvent is the inner event structure within an event_callback.
type slackInnerEvent struct {
	Type        string            `json:"type"`
	ChannelType string            `json:"channel_type"` // "im", "mpim", "channel", "group"
	Channel     string            `json:"channel"`
	User        string            `json:"user"`
	Text        slackNullableText `json:"text"`
	TS          string            `json:"ts"`
	BotID       string            `json:"bot_id,omitempty"`
}

// IMMessagePayload is the typed payload for message.im events.
// It contains the fields specified for DM event routing.
type IMMessagePayload struct {
	User      string `json:"user"`    // sender user ID (e.g., "U01234567")
	Channel   string `json:"channel"` // DM channel ID (e.g., "D01234567")
	Text      string `json:"text"`    // message text
	Timestamp string `json:"ts"`      // Slack message timestamp (e.g., "1234567890.123456")
}

// VerifyAndParseEvent implements connectors.EventSource for Slack.
// It verifies the request signature using HMAC-SHA256, handles URL verification
// challenges, and parses event_callback payloads into connector Events.
func (c *SlackConnector) VerifyAndParseEvent(payload []byte, headers http.Header, signingSecret string) (*connectors.EventEnvelope, error) {
	if err := verifySlackSignature(payload, headers, signingSecret); err != nil {
		return nil, err
	}

	var envelope slackEventEnvelope
	if err := json.Unmarshal(payload, &envelope); err != nil {
		return nil, fmt.Errorf("invalid event payload: %w", err)
	}

	// Handle URL verification challenge.
	if envelope.Type == slackEventTypeURLVerification {
		return &connectors.EventEnvelope{Challenge: envelope.Challenge}, nil
	}

	if envelope.Type != slackEventTypeEventCallback {
		return nil, fmt.Errorf("unsupported envelope type: %s", envelope.Type)
	}

	// Parse the inner event to determine the connector event type.
	var inner slackInnerEvent
	if err := json.Unmarshal(envelope.Event, &inner); err != nil {
		return nil, fmt.Errorf("invalid inner event: %w", err)
	}

	eventType := mapSlackEventType(inner)

	// Build typed payload for known event types; pass through raw for others.
	var eventPayload json.RawMessage
	if eventType == "message.im" {
		p := IMMessagePayload{
			User:      inner.User,
			Channel:   inner.Channel,
			Text:      inner.Text.String(),
			Timestamp: inner.TS,
		}
		var marshalErr error
		eventPayload, marshalErr = json.Marshal(p)
		if marshalErr != nil {
			return nil, fmt.Errorf("marshalling message.im payload: %w", marshalErr)
		}
	} else {
		eventPayload = envelope.Event
	}

	return &connectors.EventEnvelope{
		Event: &connectors.Event{
			ConnectorID: "slack",
			EventType:   eventType,
			EventID:     envelope.EventID,
			TeamID:      envelope.TeamID,
			Timestamp:   time.Unix(envelope.EventTime, 0),
			Payload:     eventPayload,
		},
	}, nil
}

// mapSlackEventType maps a Slack inner event to a connector event type string.
// message events with channel_type "im" are mapped to "message.im" to match
// Slack's event subscription naming.
func mapSlackEventType(inner slackInnerEvent) string {
	if inner.Type == "message" && inner.ChannelType == "im" {
		return "message.im"
	}
	return inner.Type
}

// verifySlackSignature verifies the HMAC-SHA256 signature of a Slack Events API
// request. Slack computes the signature as:
//
//	v0=HMAC-SHA256(signing_secret, "v0:{timestamp}:{body}")
//
// The signature is sent in the X-Slack-Signature header and the timestamp in
// X-Slack-Request-Timestamp. Requests older than 5 minutes are rejected to
// prevent replay attacks.
func verifySlackSignature(payload []byte, headers http.Header, signingSecret string) error {
	timestamp := headers.Get("X-Slack-Request-Timestamp")
	signature := headers.Get("X-Slack-Signature")

	if timestamp == "" || signature == "" {
		return fmt.Errorf("missing required headers: X-Slack-Request-Timestamp and X-Slack-Signature")
	}

	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid timestamp: %w", err)
	}

	age := time.Duration(math.Abs(float64(time.Now().Unix()-ts))) * time.Second
	if age > maxTimestampAge {
		return fmt.Errorf("request timestamp too old (%v > %v)", age, maxTimestampAge)
	}

	// Compute expected signature: v0=HMAC-SHA256(secret, "v0:{ts}:{body}")
	sigBase := fmt.Sprintf("v0:%s:%s", timestamp, string(payload))
	mac := hmac.New(sha256.New, []byte(signingSecret))
	mac.Write([]byte(sigBase))
	expected := "v0=" + hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(expected), []byte(signature)) {
		return fmt.Errorf("invalid signature")
	}

	return nil
}
