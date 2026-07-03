package agentwake

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// WakeRequest describes an approval resolution wake to deliver to OpenClaw.
type WakeRequest struct {
	ApprovalID string
	Status     string
	SessionKey string // optional; when set, targets /hooks/agent with wakeMode next-heartbeat
}

// Delivery describes a single HTTP POST to the OpenClaw gateway hooks API.
type Delivery struct {
	URL     string
	Headers map[string]string
	Body    []byte
}

// WakeMessage returns the human-readable wake text for an approval outcome.
func WakeMessage(approvalID, status string) string {
	switch status {
	case "expired":
		return fmt.Sprintf("Permission Slip %s expired unanswered", approvalID)
	default:
		return fmt.Sprintf("Permission Slip %s resolved: %s — continue the task", approvalID, status)
	}
}

// BuildOpenClawDelivery constructs the HTTP request for the OpenClaw hooks API.
// webhookBaseURL is the registered hooks base (e.g. http://100.x.x.x:18789/hooks).
// When sessionKey is non-empty, POSTs to /agent with wakeMode next-heartbeat.
func BuildOpenClawDelivery(webhookBaseURL, token string, req WakeRequest) (*Delivery, error) {
	base := strings.TrimRight(strings.TrimSpace(webhookBaseURL), "/")
	if base == "" {
		return nil, fmt.Errorf("webhook URL is empty")
	}
	message := WakeMessage(req.ApprovalID, req.Status)

	var targetURL string
	var body []byte
	var err error

	if sk := strings.TrimSpace(req.SessionKey); sk != "" {
		targetURL = base + "/agent"
		body, err = json.Marshal(map[string]string{
			"message":    message,
			"wakeMode":   "next-heartbeat",
			"sessionKey": sk,
		})
	} else {
		targetURL = base + "/wake"
		body, err = json.Marshal(map[string]string{
			"text": message,
			"mode": "now",
		})
	}
	if err != nil {
		return nil, err
	}

	if _, err := url.Parse(targetURL); err != nil {
		return nil, fmt.Errorf("invalid webhook URL: %w", err)
	}

	headers := map[string]string{
		"Content-Type":  "application/json",
		"Authorization": "Bearer " + token,
	}
	return &Delivery{
		URL:     targetURL,
		Headers: headers,
		Body:    body,
	}, nil
}

// BuildTestDelivery returns a test wake delivery (no session key).
func BuildTestDelivery(webhookBaseURL, token string) (*Delivery, error) {
	return BuildOpenClawDelivery(webhookBaseURL, token, WakeRequest{
		ApprovalID: "test_wake",
		Status:     "test",
	})
}

// SessionKeyFromApprovalContext extracts an optional session_key from approval context JSON.
func SessionKeyFromApprovalContext(contextJSON []byte) string {
	if len(contextJSON) == 0 {
		return ""
	}
	var ctx map[string]json.RawMessage
	if err := json.Unmarshal(contextJSON, &ctx); err != nil {
		return ""
	}
	raw, ok := ctx["session_key"]
	if !ok {
		return ""
	}
	var sk string
	if err := json.Unmarshal(raw, &sk); err != nil {
		return ""
	}
	return strings.TrimSpace(sk)
}
