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

	// maxResponseBytes caps the Slack API response body at 10 MB to prevent
	// memory exhaustion from unexpectedly large payloads (e.g., 1000 messages
	// with rich content). Slack's largest documented payloads are well under
	// this limit.
	maxResponseBytes = 10 << 20 // 10 MB
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
			{
				ActionType:  "slack.list_channels",
				Name:        "List Channels",
				Description: "List Slack channels visible to the bot",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"properties": {
						"types": {
							"type": "string",
							"default": "public_channel",
							"description": "Comma-separated channel types: public_channel, private_channel, mpim, im"
						},
						"limit": {
							"type": "integer",
							"default": 100,
							"description": "Max channels to return (1-1000)"
						},
						"cursor": {
							"type": "string",
							"description": "Pagination cursor from a previous response"
						},
						"exclude_archived": {
							"type": "boolean",
							"default": true,
							"description": "Exclude archived channels from results"
						}
					}
				}`)),
			},
			{
				ActionType:  "slack.read_channel_messages",
				Name:        "Read Channel Messages",
				Description: "Read recent messages from a Slack channel",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["channel"],
					"properties": {
						"channel": {
							"type": "string",
							"description": "Channel ID (e.g. C01234567)"
						},
						"limit": {
							"type": "integer",
							"default": 20,
							"description": "Max messages to return (1-1000)"
						},
						"oldest": {
							"type": "string",
							"description": "Only messages after this Unix timestamp"
						},
						"latest": {
							"type": "string",
							"description": "Only messages before this Unix timestamp"
						},
						"cursor": {
							"type": "string",
							"description": "Pagination cursor from a previous response"
						}
					}
				}`)),
			},
			{
				ActionType:  "slack.read_thread",
				Name:        "Read Thread",
				Description: "Read replies in a Slack thread",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["channel", "thread_ts"],
					"properties": {
						"channel": {
							"type": "string",
							"description": "Channel ID containing the thread (e.g. C01234567)"
						},
						"thread_ts": {
							"type": "string",
							"description": "Timestamp of the parent message (e.g. 1234567890.123456)"
						},
						"limit": {
							"type": "integer",
							"default": 50,
							"description": "Max replies to return (1-1000)"
						},
						"cursor": {
							"type": "string",
							"description": "Pagination cursor from a previous response"
						}
					}
				}`)),
			},
		},
		{
			ActionType:  "slack.schedule_message",
			Name:        "Schedule Message",
			Description: "Schedule a message for future delivery to a Slack channel",
			RiskLevel:   "low",
			ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
				"type": "object",
				"required": ["channel", "message", "post_at"],
				"properties": {
					"channel": {
						"type": "string",
						"description": "Channel name (e.g. #general) or ID (e.g. C01234567)"
					},
					"message": {
						"type": "string",
						"description": "Message text (supports Slack mrkdwn formatting)"
					},
					"post_at": {
						"type": "integer",
						"description": "Unix timestamp for when the message should be sent (must be in the future)"
					}
				}
			}`)),
		},
		{
			ActionType:  "slack.set_topic",
			Name:        "Set Topic",
			Description: "Update a Slack channel's topic",
			RiskLevel:   "medium",
			ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
				"type": "object",
				"required": ["channel", "topic"],
				"properties": {
					"channel": {
						"type": "string",
						"description": "Channel ID (e.g. C01234567)"
					},
					"topic": {
						"type": "string",
						"description": "New channel topic (max 250 characters)"
					}
				}
			}`)),
		},
		{
			ActionType:  "slack.invite_to_channel",
			Name:        "Invite to Channel",
			Description: "Invite one or more users to a Slack channel",
			RiskLevel:   "medium",
			ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
				"type": "object",
				"required": ["channel", "users"],
				"properties": {
					"channel": {
						"type": "string",
						"description": "Channel ID (e.g. C01234567)"
					},
					"users": {
						"type": "string",
						"description": "Comma-separated list of user IDs to invite (e.g. U01234567,U09876543)"
					}
				}
			}`)),
		},
		{
			ActionType:  "slack.upload_file",
			Name:        "Upload File",
			Description: "Upload a file to a Slack channel",
			RiskLevel:   "low",
			ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
				"type": "object",
				"required": ["channel", "filename", "content"],
				"properties": {
					"channel": {
						"type": "string",
						"description": "Channel ID (e.g. C01234567)"
					},
					"filename": {
						"type": "string",
						"description": "Name of the file (e.g. report.csv)"
					},
					"content": {
						"type": "string",
						"description": "File content as text"
					},
					"title": {
						"type": "string",
						"description": "Display title for the file (defaults to filename)"
					}
				}
			}`)),
		},
		{
			ActionType:  "slack.add_reaction",
			Name:        "Add Reaction",
			Description: "Add an emoji reaction to a Slack message",
			RiskLevel:   "low",
			ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
				"type": "object",
				"required": ["channel", "timestamp", "name"],
				"properties": {
					"channel": {
						"type": "string",
						"description": "Channel ID containing the message (e.g. C01234567)"
					},
					"timestamp": {
						"type": "string",
						"description": "Timestamp of the message to react to (e.g. 1234567890.123456)"
					},
					"name": {
						"type": "string",
						"description": "Emoji name without colons (e.g. thumbsup, white_check_mark)"
					}
				}
			}`)),
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
			{
				ID:          "tpl_slack_list_channels",
				ActionType:  "slack.list_channels",
				Name:        "List channels",
				Description: "Agent can list channels visible to the bot.",
				Parameters:  json.RawMessage(`{"types":"*","limit":"*","cursor":"*"}`),
			},
			{
				ID:          "tpl_slack_read_channel",
				ActionType:  "slack.read_channel_messages",
				Name:        "Read channel messages",
				Description: "Agent can read messages from any channel.",
				Parameters:  json.RawMessage(`{"channel":"*","limit":"*","oldest":"*","latest":"*","cursor":"*"}`),
			},
			{
				ID:          "tpl_slack_read_thread",
				ActionType:  "slack.read_thread",
				Name:        "Read thread replies",
				Description: "Agent can read thread replies from any channel.",
				Parameters:  json.RawMessage(`{"channel":"*","thread_ts":"*","limit":"*","cursor":"*"}`),
			},
			{
				ID:          "tpl_slack_schedule_message",
				ActionType:  "slack.schedule_message",
				Name:        "Schedule messages",
				Description: "Agent can schedule messages to any channel.",
				Parameters:  json.RawMessage(`{"channel":"*","message":"*","post_at":"*"}`),
			},
			{
				ID:          "tpl_slack_set_topic",
				ActionType:  "slack.set_topic",
				Name:        "Set channel topics",
				Description: "Agent can update the topic on any channel.",
				Parameters:  json.RawMessage(`{"channel":"*","topic":"*"}`),
			},
			{
				ID:          "tpl_slack_invite_to_channel",
				ActionType:  "slack.invite_to_channel",
				Name:        "Invite users to channels",
				Description: "Agent can invite users to any channel.",
				Parameters:  json.RawMessage(`{"channel":"*","users":"*"}`),
			},
			{
				ID:          "tpl_slack_upload_file",
				ActionType:  "slack.upload_file",
				Name:        "Upload files",
				Description: "Agent can upload files to any channel.",
				Parameters:  json.RawMessage(`{"channel":"*","filename":"*","content":"*","title":"*"}`),
			},
			{
				ID:          "tpl_slack_add_reaction",
				ActionType:  "slack.add_reaction",
				Name:        "Add reactions",
				Description: "Agent can add emoji reactions to messages.",
				Parameters:  json.RawMessage(`{"channel":"*","timestamp":"*","name":"*"}`),
			},
		},
	}
}

// Actions returns the registered action handlers keyed by action_type.
func (c *SlackConnector) Actions() map[string]connectors.Action {
	return map[string]connectors.Action{
		"slack.send_message":          &sendMessageAction{conn: c},
		"slack.create_channel":        &createChannelAction{conn: c},
		"slack.list_channels":         &listChannelsAction{conn: c},
		"slack.read_channel_messages": &readChannelMessagesAction{conn: c},
		"slack.read_thread":           &readThreadAction{conn: c},
		"slack.schedule_message":      &scheduleMessageAction{conn: c},
		"slack.set_topic":             &setTopicAction{conn: c},
		"slack.invite_to_channel":     &inviteToChannelAction{conn: c},
		"slack.upload_file":           &uploadFileAction{conn: c},
		"slack.add_reaction":          &addReactionAction{conn: c},
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

// paginationMeta is the shared response_metadata shape for paginated endpoints.
type paginationMeta struct {
	NextCursor string `json:"next_cursor"`
}

// validateChannelID checks that a channel parameter looks like a valid Slack
// channel ID (starts with C, G, or D). This catches common mistakes like
// passing a channel name instead of an ID before hitting the Slack API.
func validateChannelID(channel string) error {
	if channel == "" {
		return &connectors.ValidationError{Message: "missing required parameter: channel"}
	}
	if len(channel) < 2 {
		return &connectors.ValidationError{
			Message: fmt.Sprintf("invalid channel ID %q: expected a Slack channel ID starting with C, G, or D (e.g. C01234567)", channel),
		}
	}
	switch channel[0] {
	case 'C', 'G', 'D':
		return nil
	default:
		return &connectors.ValidationError{
			Message: fmt.Sprintf("invalid channel ID %q: expected a Slack channel ID starting with C, G, or D — did you pass a channel name instead?", channel),
		}
	}
}

// validateLimit checks that a pagination limit is within the Slack API range (1-1000).
// A zero value means "use default" and is always valid.
func validateLimit(limit int) error {
	if limit != 0 && (limit < 1 || limit > 1000) {
		return &connectors.ValidationError{
			Message: fmt.Sprintf("limit must be between 1 and 1000, got %d", limit),
		}
	}
	return nil
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

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
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
// connector error type with user-friendly messages for common errors.
func mapSlackError(slackErr string) error {
	switch slackErr {
	// Auth errors
	case "not_authed", "invalid_auth", "token_revoked", "token_expired", "account_inactive":
		return &connectors.AuthError{Message: fmt.Sprintf("Slack auth error: %s", slackErr)}
	case "missing_scope":
		return &connectors.AuthError{Message: "Slack bot token is missing a required OAuth scope — check your app's permissions at https://api.slack.com/apps"}

	// Rate limiting
	case "ratelimited":
		return &connectors.RateLimitError{Message: "Slack API rate limit exceeded"}

	// Channel errors
	case "channel_not_found":
		return &connectors.ExternalError{StatusCode: 200, Message: "Slack channel not found — verify the channel ID exists and the bot has access"}
	case "not_in_channel":
		return &connectors.ExternalError{StatusCode: 200, Message: "bot is not a member of this channel — invite the bot first"}
	case "is_archived":
		return &connectors.ExternalError{StatusCode: 200, Message: "cannot perform this action on an archived channel"}

	// Reaction errors
	case "already_reacted":
		return &connectors.ExternalError{StatusCode: 200, Message: "this emoji reaction has already been added to this message"}
	case "too_many_emoji":
		return &connectors.ExternalError{StatusCode: 200, Message: "too many emoji reactions on this message"}

	// Invite errors
	case "already_in_channel":
		return &connectors.ExternalError{StatusCode: 200, Message: "one or more users are already members of this channel"}
	case "cant_invite_self":
		return &connectors.ExternalError{StatusCode: 200, Message: "the bot cannot invite itself to a channel"}
	case "user_not_found":
		return &connectors.ExternalError{StatusCode: 200, Message: "one or more user IDs were not found — verify the user IDs are correct"}

	// Message errors
	case "time_in_past":
		return &connectors.ExternalError{StatusCode: 200, Message: "scheduled message time is in the past — post_at must be a future Unix timestamp"}
	case "message_too_long":
		return &connectors.ExternalError{StatusCode: 200, Message: "message exceeds Slack's maximum length"}

	default:
		return &connectors.ExternalError{
			StatusCode: 200,
			Message:    fmt.Sprintf("Slack API error: %s", slackErr),
		}
	}
}
