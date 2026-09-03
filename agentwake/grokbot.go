package agentwake

import (
	"encoding/json"
	"fmt"
	"net/url"
	"path"
	"strings"
)

const (
	grokBotWebhookHost       = "api2.cursor.sh"
	grokBotWebhookPathPrefix = "/automations/webhook/"
)

type grokBotWakeBody struct {
	Source     string `json:"source"`
	ApprovalID string `json:"approval_id"`
	Status     string `json:"status"`
	AgentID    int64  `json:"agent_id"`
	Text       string `json:"text,omitempty"`
}

// ValidateGrokBotWebhookURL accepts only public Cursor automation webhook URLs.
// OpenClaw's private-network check is not applied on this path.
func ValidateGrokBotWebhookURL(raw string) error {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return fmt.Errorf("url is required")
	}
	u, err := url.Parse(trimmed)
	if err != nil || u.Host == "" {
		return fmt.Errorf("url is not a valid URL")
	}
	if u.Scheme != "https" {
		return fmt.Errorf("Grok Bot webhook URL must use https")
	}
	if u.User != nil {
		return fmt.Errorf("Grok Bot webhook URL must not include credentials")
	}
	if port := u.Port(); port != "" && port != "443" {
		return fmt.Errorf("Grok Bot webhook URL host must be %s", grokBotWebhookHost)
	}
	if strings.ToLower(u.Hostname()) != grokBotWebhookHost {
		return fmt.Errorf("Grok Bot webhook URL host must be %s", grokBotWebhookHost)
	}
	cleanedPath := path.Clean(u.EscapedPath())
	if !strings.HasPrefix(cleanedPath, grokBotWebhookPathPrefix) {
		return fmt.Errorf("Grok Bot webhook URL path must start with %s", grokBotWebhookPathPrefix)
	}
	id := strings.Trim(strings.TrimPrefix(cleanedPath, grokBotWebhookPathPrefix), "/")
	if id == "" || strings.Contains(id, "/") {
		return fmt.Errorf("Grok Bot webhook URL must be https://%s%s<id>", grokBotWebhookHost, grokBotWebhookPathPrefix)
	}
	return nil
}

// BuildGrokBotDelivery POSTs to the registered Cursor webhook URL as-is.
// The token is sent as the Authorization header (Bearer is prefixed when absent).
func BuildGrokBotDelivery(webhookURL, token string, req WakeRequest) (*Delivery, error) {
	target := strings.TrimSpace(webhookURL)
	if target == "" {
		return nil, fmt.Errorf("webhook URL is empty")
	}
	if err := ValidateGrokBotWebhookURL(target); err != nil {
		return nil, err
	}
	auth := grokBotAuthorizationHeader(token)
	if auth == "" {
		return nil, fmt.Errorf("webhook token is empty")
	}

	body, err := json.Marshal(grokBotWakeBody{
		Source:     "permission-slip",
		ApprovalID: req.ApprovalID,
		Status:     req.Status,
		AgentID:    req.AgentID,
		Text:       WakeMessage(req.ApprovalID, req.Status),
	})
	if err != nil {
		return nil, err
	}

	return &Delivery{
		URL: target,
		Headers: map[string]string{
			"Content-Type":  "application/json",
			"Authorization": auth,
		},
		Body: body,
	}, nil
}

func grokBotAuthorizationHeader(token string) string {
	t := strings.TrimSpace(token)
	if t == "" {
		return ""
	}
	if strings.Contains(t, " ") {
		return t
	}
	return "Bearer " + t
}
