// Package zapier implements the Zapier connector for the Permission Slip
// connector execution layer. It uses Zapier's webhook-based approach to
// trigger Zaps via simple HTTP POST requests. Each Zap has a unique
// webhook URL that acts as its trigger endpoint.
package zapier

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

const (
	defaultTimeout = 30 * time.Second

	// credKeyWebhookURL is the credential key for the Zapier webhook URL.
	// Webhook URLs are effectively secrets since they allow triggering Zaps.
	credKeyWebhookURL = "webhook_url"

	// maxResponseBytes caps the response body at 1 MB.
	maxResponseBytes = 1 << 20

	// webhookURLPrefix is the expected prefix for Zapier webhook URLs.
	webhookURLPrefix = "https://hooks.zapier.com/"
)

// ZapierConnector owns the shared HTTP client used by all Zapier actions.
type ZapierConnector struct {
	client *http.Client
}

// New creates a ZapierConnector with sensible defaults (30s timeout).
func New() *ZapierConnector {
	return &ZapierConnector{
		client: &http.Client{Timeout: defaultTimeout},
	}
}

// newForTest creates a ZapierConnector that uses a test HTTP client.
func newForTest(client *http.Client) *ZapierConnector {
	return &ZapierConnector{
		client: client,
	}
}

// ID returns "zapier", matching the connectors.id in the database.
func (c *ZapierConnector) ID() string { return "zapier" }

// Manifest returns the connector's metadata manifest.
func (c *ZapierConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "zapier",
		Name:        "Zapier",
		Description: "Trigger Zapier workflows (Zaps) via webhooks — one connector that unlocks thousands of integrations",
		Actions: []connectors.ManifestAction{
			{
				ActionType:  "zapier.trigger_webhook",
				Name:        "Trigger Webhook",
				Description: "Send a JSON payload to a Zapier webhook URL to trigger a Zap. Supports both fire-and-forget and synchronous (wait for response) modes.",
				RiskLevel:   "high",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["payload"],
					"properties": {
						"payload": {
							"type": "object",
							"description": "JSON data to send to the Zap webhook"
						},
						"wait_for_response": {
							"type": "boolean",
							"default": false,
							"description": "If true, wait for the Zap to complete and return its response (Zapier must be configured for 'Webhooks by Zapier' with a response step)"
						}
					}
				}`)),
			},
		},
		RequiredCredentials: []connectors.ManifestCredential{
			{
				Service:         "zapier",
				AuthType:        "custom",
				InstructionsURL: "https://help.zapier.com/hc/en-us/articles/8496293271053-How-to-get-started-with-Webhooks-by-Zapier",
			},
		},
		Templates: []connectors.ManifestTemplate{
			{
				ID:          "tpl_zapier_trigger_fire_and_forget",
				ActionType:  "zapier.trigger_webhook",
				Name:        "Fire-and-forget webhook",
				Description: "Trigger a Zap without waiting for a response.",
				Parameters:  json.RawMessage(`{"payload":"*","wait_for_response":false}`),
			},
			{
				ID:          "tpl_zapier_trigger_sync",
				ActionType:  "zapier.trigger_webhook",
				Name:        "Synchronous webhook",
				Description: "Trigger a Zap and wait for its response.",
				Parameters:  json.RawMessage(`{"payload":"*","wait_for_response":true}`),
			},
		},
	}
}

// Actions returns the registered action handlers keyed by action_type.
func (c *ZapierConnector) Actions() map[string]connectors.Action {
	return map[string]connectors.Action{
		"zapier.trigger_webhook": &triggerWebhookAction{conn: c},
	}
}

// ValidateCredentials checks that the provided credentials contain a
// valid Zapier webhook URL.
func (c *ZapierConnector) ValidateCredentials(_ context.Context, creds connectors.Credentials) error {
	webhookURL, ok := creds.Get(credKeyWebhookURL)
	if !ok || webhookURL == "" {
		return &connectors.ValidationError{Message: "missing required credential: webhook_url"}
	}
	u, err := url.Parse(webhookURL)
	if err != nil {
		return &connectors.ValidationError{Message: fmt.Sprintf("webhook_url is not a valid URL: %v", err)}
	}
	if u.Scheme != "https" {
		return &connectors.ValidationError{Message: "webhook_url must use HTTPS"}
	}
	if u.Host == "" {
		return &connectors.ValidationError{Message: "webhook_url must include a host"}
	}
	// Warn if the URL doesn't look like a standard Zapier webhook — this
	// catches common mistakes like pasting a regular Zapier page URL.
	if !strings.HasPrefix(webhookURL, webhookURLPrefix) {
		return &connectors.ValidationError{
			Message: fmt.Sprintf("webhook_url should start with %q — make sure you're using a Webhooks by Zapier trigger URL, not a regular Zapier page URL", webhookURLPrefix),
		}
	}
	return nil
}

// doPost sends a POST request with JSON body to the given URL and reads the response.
func (c *ZapierConnector) doPost(ctx context.Context, targetURL string, body any) ([]byte, int, error) {
	payload, err := json.Marshal(body)
	if err != nil {
		return nil, 0, fmt.Errorf("marshaling request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, targetURL, bytes.NewReader(payload))
	if err != nil {
		return nil, 0, fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		if connectors.IsTimeout(err) {
			return nil, 0, &connectors.TimeoutError{Message: fmt.Sprintf("Zapier webhook request timed out: %v", err)}
		}
		if errors.Is(err, context.Canceled) {
			return nil, 0, &connectors.TimeoutError{Message: "Zapier webhook request canceled"}
		}
		return nil, 0, &connectors.ExternalError{Message: fmt.Sprintf("Zapier webhook request failed: %v", err)}
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusTooManyRequests {
		retryAfter := connectors.ParseRetryAfter(resp.Header.Get("Retry-After"), 30*time.Second)
		return nil, resp.StatusCode, &connectors.RateLimitError{
			Message:    "Zapier webhook rate limit exceeded",
			RetryAfter: retryAfter,
		}
	}

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return nil, resp.StatusCode, &connectors.ExternalError{Message: fmt.Sprintf("reading response body: %v", err)}
	}

	return respBody, resp.StatusCode, nil
}
