package agentwake

import (
	"fmt"
	"strings"
)

const (
	// ProviderOpenClaw delivers wakes to an OpenClaw gateway hooks API
	// (private URL, /hooks/wake or /hooks/agent suffix, Bearer token).
	ProviderOpenClaw = "openclaw"
	// ProviderGrokBot delivers wakes to a Grok Bot Cursor automation webhook
	// (public https://api2.cursor.sh/automations/webhook/… URL, POST as-is).
	ProviderGrokBot = "grokbot"
)

// NormalizeProvider returns the canonical provider id.
// Empty input defaults to OpenClaw (legacy registrations).
func NormalizeProvider(raw string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "", ProviderOpenClaw:
		return ProviderOpenClaw, nil
	case ProviderGrokBot:
		return ProviderGrokBot, nil
	default:
		return "", fmt.Errorf("provider must be %s or %s", ProviderOpenClaw, ProviderGrokBot)
	}
}

// BuildDelivery constructs the provider-specific wake HTTP request.
func BuildDelivery(provider, webhookURL, token string, req WakeRequest) (*Delivery, error) {
	normalized, err := NormalizeProvider(provider)
	if err != nil {
		return nil, err
	}
	if normalized == ProviderGrokBot {
		return BuildGrokBotDelivery(webhookURL, token, req)
	}
	return BuildOpenClawDelivery(webhookURL, token, req)
}

// BuildTestDelivery returns a test wake for the given provider.
func BuildTestDelivery(provider, webhookURL, token string, agentID int64) (*Delivery, error) {
	return BuildDelivery(provider, webhookURL, token, WakeRequest{
		ApprovalID: "test_wake",
		Status:     "test",
		AgentID:    agentID,
	})
}
