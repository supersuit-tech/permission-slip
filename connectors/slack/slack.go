// Package slack implements the Slack connector for the Permission Slip
// connector execution layer. It uses the Slack Web API with plain net/http
// (no third-party SDK) to keep the dependency footprint minimal.
package slack

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

const (
	defaultBaseURL = "https://slack.com/api"
	defaultTimeout = 30 * time.Second
	credKeyToken   = "bot_token"
	tokenPrefix    = "xoxb-"

	// defaultRetryAfter is used when the Slack API returns a rate limit
	// response without a Retry-After header (or an unparseable one).
	defaultRetryAfter = 30 * time.Second
)

// SlackConnector owns the shared HTTP client and base URL used by all
// Slack actions. Actions hold a pointer back to the connector to access
// these shared resources.
type SlackConnector struct {
	client  *http.Client
	baseURL string
}

// New creates a SlackConnector with sensible defaults (30s timeout,
// https://slack.com/api base URL).
func New() *SlackConnector {
	return &SlackConnector{
		client:  &http.Client{Timeout: defaultTimeout},
		baseURL: defaultBaseURL,
	}
}

// newForTest creates a SlackConnector that points at a test server.
func newForTest(client *http.Client, baseURL string) *SlackConnector {
	return &SlackConnector{
		client:  client,
		baseURL: baseURL,
	}
}

// ID returns "slack", matching the connectors.id in the database.
func (c *SlackConnector) ID() string { return "slack" }

// Manifest returns the connector's metadata manifest. Used by the server to
// auto-seed DB rows on startup, replacing manual seed.go files.
func (c *SlackConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "slack",
		Name:        "Slack",
		Description: "Slack integration for team communication",
		Actions: []connectors.ManifestAction{
			{
				ActionType:  "slack.send_message",
				Name:        "Send Message",
				Description: "Send a message to a Slack channel",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["channel", "message"],
					"properties": {
						"channel": {
							"type": "string",
							"description": "Channel name (e.g. #general) or ID (e.g. C01234567)"
						},
						"message": {
							"type": "string",
							"description": "Message text (supports Slack mrkdwn formatting)"
						}
					}
				}`)),
			},
			{
				ActionType:  "slack.create_channel",
				Name:        "Create Channel",
				Description: "Create a new Slack channel",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["name"],
					"properties": {
						"name": {
							"type": "string",
							"description": "Channel name (lowercase, no spaces, max 80 chars)"
						},
						"is_private": {
							"type": "boolean",
							"default": false,
							"description": "Create as a private channel"
						}
					}
				}`)),
			},
		},
		RequiredCredentials: []connectors.ManifestCredential{
			{Service: "slack", AuthType: "custom", InstructionsURL: "https://api.slack.com/tutorials/tracks/getting-a-token"},
		},
		Templates: []connectors.ManifestTemplate{
			{
				ID:          "tpl_slack_send_to_channel",
				ActionType:  "slack.send_message",
				Name:        "Post to a channel",
				Description: "Locks the channel and lets the agent choose the message content.",
				Parameters:  json.RawMessage(`{"channel":"#general","message":"*"}`),
			},
			{
				ID:          "tpl_slack_send_any",
				ActionType:  "slack.send_message",
				Name:        "Send messages freely",
				Description: "Agent can send any message to any channel.",
				Parameters:  json.RawMessage(`{"channel":"*","message":"*"}`),
			},
			{
				ID:          "tpl_slack_create_channel",
				ActionType:  "slack.create_channel",
				Name:        "Create channels",
				Description: "Agent can create public channels with any name.",
				Parameters:  json.RawMessage(`{"name":"*","is_private":false}`),
			},
		},
	}
}

// Actions returns the registered action handlers keyed by action_type.
func (c *SlackConnector) Actions() map[string]connectors.Action {
	return map[string]connectors.Action{
		"slack.send_message":   &sendMessageAction{conn: c},
		"slack.create_channel": &createChannelAction{conn: c},
	}
}

// ValidateCredentials checks that the provided credentials contain a
// non-empty bot_token with the required xoxb- prefix.
func (c *SlackConnector) ValidateCredentials(_ context.Context, creds connectors.Credentials) error {
	token, ok := creds.Get(credKeyToken)
	if !ok || token == "" {
		return &connectors.ValidationError{Message: "missing required credential: bot_token"}
	}
	if len(token) < len(tokenPrefix) || token[:len(tokenPrefix)] != tokenPrefix {
		return &connectors.ValidationError{Message: "bot_token must start with \"xoxb-\""}
	}
	return nil
}

// slackResponse is the common envelope for Slack Web API responses.
// Every endpoint returns {"ok": true/false, ...}.
type slackResponse struct {
	OK    bool   `json:"ok"`
	Error string `json:"error,omitempty"`
}

// doPost is the shared request lifecycle for all Slack actions. It marshals
// body as JSON, sends a POST to the given Slack API method with auth headers,
// handles rate limiting and timeouts, and unmarshals the response into dest.
// Callers are responsible for checking the Slack-level ok/error fields in dest.
func (c *SlackConnector) doPost(ctx context.Context, method string, creds connectors.Credentials, body any, dest any) error {
	token, ok := creds.Get(credKeyToken)
	if !ok || token == "" {
		return &connectors.ValidationError{Message: "bot_token credential is missing or empty"}
	}

	payload, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshaling request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/"+method, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json; charset=utf-8")

	resp, err := c.client.Do(req)
	if err != nil {
		if connectors.IsTimeout(err) {
			return &connectors.TimeoutError{Message: fmt.Sprintf("Slack API request timed out: %v", err)}
		}
		if errors.Is(err, context.Canceled) {
			return &connectors.TimeoutError{Message: "Slack API request canceled"}
		}
		return &connectors.ExternalError{Message: fmt.Sprintf("Slack API request failed: %v", err)}
	}
	defer resp.Body.Close()

	// Slack returns 429 for rate limiting with a Retry-After header.
	if resp.StatusCode == http.StatusTooManyRequests {
		retryAfter := connectors.ParseRetryAfter(resp.Header.Get("Retry-After"), defaultRetryAfter)
		return &connectors.RateLimitError{
			Message:    "Slack API rate limit exceeded",
			RetryAfter: retryAfter,
		}
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return &connectors.ExternalError{Message: fmt.Sprintf("reading response body: %v", err)}
	}

	if err := json.Unmarshal(respBody, dest); err != nil {
		return &connectors.ExternalError{
			StatusCode: resp.StatusCode,
			Message:    "failed to decode Slack API response",
		}
	}

	return nil
}

// mapSlackError converts a Slack API error string to the appropriate
// connector error type.
func mapSlackError(slackErr string) error {
	switch slackErr {
	case "not_authed", "invalid_auth", "token_revoked", "token_expired", "account_inactive":
		return &connectors.AuthError{Message: fmt.Sprintf("Slack auth error: %s", slackErr)}
	case "ratelimited":
		return &connectors.RateLimitError{Message: "Slack API rate limit exceeded"}
	default:
		return &connectors.ExternalError{
			StatusCode: 200,
			Message:    fmt.Sprintf("Slack API error: %s", slackErr),
		}
	}
}
