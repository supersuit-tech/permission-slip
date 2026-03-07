// Package discord implements the Discord connector for the Permission Slip
// connector execution layer. It uses the Discord REST API v10 with plain
// net/http (no third-party SDK) to keep the dependency footprint minimal.
package discord

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
	defaultBaseURL = "https://discord.com/api/v10"
	defaultTimeout = 30 * time.Second
	credKeyToken   = "bot_token"

	// defaultRetryAfter is used when Discord returns a rate limit response
	// without a Retry-After header (or an unparseable one).
	defaultRetryAfter = 5 * time.Second

	// maxResponseBytes caps the Discord API response body at 10 MB.
	maxResponseBytes = 10 << 20 // 10 MB
)

// DiscordConnector owns the shared HTTP client and base URL used by all
// Discord actions. Actions hold a pointer back to the connector to access
// these shared resources.
type DiscordConnector struct {
	client  *http.Client
	baseURL string
}

// New creates a DiscordConnector with sensible defaults.
func New() *DiscordConnector {
	return &DiscordConnector{
		client:  &http.Client{Timeout: defaultTimeout},
		baseURL: defaultBaseURL,
	}
}

// newForTest creates a DiscordConnector that points at a test server.
func newForTest(client *http.Client, baseURL string) *DiscordConnector {
	return &DiscordConnector{
		client:  client,
		baseURL: baseURL,
	}
}

// ID returns "discord", matching the connectors.id in the database.
func (c *DiscordConnector) ID() string { return "discord" }

// Manifest returns the connector's metadata manifest.
func (c *DiscordConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "discord",
		Name:        "Discord",
		Description: "Discord integration for community management and messaging",
		Actions: []connectors.ManifestAction{
			{
				ActionType:  "discord.send_message",
				Name:        "Send Message",
				Description: "Send a message to a Discord channel",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["channel_id", "content"],
					"properties": {
						"channel_id": {
							"type": "string",
							"description": "Discord channel ID (snowflake)"
						},
						"content": {
							"type": "string",
							"description": "Message content (up to 2000 characters)"
						}
					}
				}`)),
			},
			{
				ActionType:  "discord.create_channel",
				Name:        "Create Channel",
				Description: "Create a new channel in a Discord guild",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["guild_id", "name"],
					"properties": {
						"guild_id": {
							"type": "string",
							"description": "Discord guild (server) ID"
						},
						"name": {
							"type": "string",
							"description": "Channel name (2-100 characters, lowercase, no spaces)"
						},
						"type": {
							"type": "integer",
							"default": 0,
							"description": "Channel type: 0 = text, 2 = voice, 4 = category"
						},
						"parent_id": {
							"type": "string",
							"description": "Parent category channel ID"
						},
						"topic": {
							"type": "string",
							"description": "Channel topic (up to 1024 characters, text channels only)"
						}
					}
				}`)),
			},
			{
				ActionType:  "discord.manage_roles",
				Name:        "Manage Roles",
				Description: "Assign or remove a role from a guild member",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["guild_id", "user_id", "role_id", "action"],
					"properties": {
						"guild_id": {
							"type": "string",
							"description": "Discord guild (server) ID"
						},
						"user_id": {
							"type": "string",
							"description": "Discord user ID"
						},
						"role_id": {
							"type": "string",
							"description": "Discord role ID"
						},
						"action": {
							"type": "string",
							"enum": ["assign", "remove"],
							"description": "Whether to assign or remove the role"
						}
					}
				}`)),
			},
			{
				ActionType:  "discord.create_event",
				Name:        "Create Event",
				Description: "Create a scheduled event in a Discord guild",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["guild_id", "name", "scheduled_start_time", "privacy_level", "entity_type"],
					"properties": {
						"guild_id": {
							"type": "string",
							"description": "Discord guild (server) ID"
						},
						"name": {
							"type": "string",
							"description": "Event name (1-100 characters)"
						},
						"description": {
							"type": "string",
							"description": "Event description (up to 1000 characters)"
						},
						"scheduled_start_time": {
							"type": "string",
							"description": "ISO 8601 timestamp for event start"
						},
						"scheduled_end_time": {
							"type": "string",
							"description": "ISO 8601 timestamp for event end (required for external events)"
						},
						"privacy_level": {
							"type": "integer",
							"default": 2,
							"description": "Privacy level: 2 = guild only"
						},
						"entity_type": {
							"type": "integer",
							"description": "Entity type: 1 = stage, 2 = voice, 3 = external"
						},
						"channel_id": {
							"type": "string",
							"description": "Channel ID for stage/voice events"
						},
						"entity_metadata": {
							"type": "object",
							"properties": {
								"location": {
									"type": "string",
									"description": "Location for external events"
								}
							}
						}
					}
				}`)),
			},
			{
				ActionType:  "discord.ban_user",
				Name:        "Ban User",
				Description: "Ban a user from a Discord guild",
				RiskLevel:   "high",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["guild_id", "user_id"],
					"properties": {
						"guild_id": {
							"type": "string",
							"description": "Discord guild (server) ID"
						},
						"user_id": {
							"type": "string",
							"description": "Discord user ID to ban"
						},
						"delete_message_seconds": {
							"type": "integer",
							"default": 0,
							"description": "Seconds of message history to delete (0-604800)"
						}
					}
				}`)),
			},
			{
				ActionType:  "discord.kick_user",
				Name:        "Kick User",
				Description: "Kick a user from a Discord guild",
				RiskLevel:   "high",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["guild_id", "user_id"],
					"properties": {
						"guild_id": {
							"type": "string",
							"description": "Discord guild (server) ID"
						},
						"user_id": {
							"type": "string",
							"description": "Discord user ID to kick"
						}
					}
				}`)),
			},
			{
				ActionType:  "discord.pin_message",
				Name:        "Pin Message",
				Description: "Pin a message in a Discord channel",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["channel_id", "message_id"],
					"properties": {
						"channel_id": {
							"type": "string",
							"description": "Discord channel ID"
						},
						"message_id": {
							"type": "string",
							"description": "Discord message ID to pin"
						}
					}
				}`)),
			},
			{
				ActionType:  "discord.unpin_message",
				Name:        "Unpin Message",
				Description: "Unpin a message in a Discord channel",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["channel_id", "message_id"],
					"properties": {
						"channel_id": {
							"type": "string",
							"description": "Discord channel ID"
						},
						"message_id": {
							"type": "string",
							"description": "Discord message ID to unpin"
						}
					}
				}`)),
			},
			{
				ActionType:  "discord.list_channels",
				Name:        "List Channels",
				Description: "List channels in a Discord guild",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["guild_id"],
					"properties": {
						"guild_id": {
							"type": "string",
							"description": "Discord guild (server) ID"
						}
					}
				}`)),
			},
			{
				ActionType:  "discord.list_members",
				Name:        "List Members",
				Description: "List members of a Discord guild",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["guild_id"],
					"properties": {
						"guild_id": {
							"type": "string",
							"description": "Discord guild (server) ID"
						},
						"limit": {
							"type": "integer",
							"default": 100,
							"description": "Max members to return (1-1000)"
						},
						"after": {
							"type": "string",
							"description": "User ID to paginate after"
						}
					}
				}`)),
			},
			{
				ActionType:  "discord.create_thread",
				Name:        "Create Thread",
				Description: "Create a thread in a Discord channel",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["channel_id", "name"],
					"properties": {
						"channel_id": {
							"type": "string",
							"description": "Discord channel ID to create the thread in"
						},
						"name": {
							"type": "string",
							"description": "Thread name (1-100 characters)"
						},
						"message_id": {
							"type": "string",
							"description": "Message ID to start the thread from (omit for a thread without a starter message)"
						},
						"auto_archive_duration": {
							"type": "integer",
							"default": 1440,
							"description": "Minutes of inactivity before auto-archive: 60, 1440, 4320, or 10080"
						}
					}
				}`)),
			},
		},
		RequiredCredentials: []connectors.ManifestCredential{
			{Service: "discord", AuthType: "custom", InstructionsURL: "https://discord.com/developers/docs/getting-started"},
		},
		Templates: []connectors.ManifestTemplate{
			{
				ID:          "tpl_discord_send_message",
				ActionType:  "discord.send_message",
				Name:        "Send messages freely",
				Description: "Agent can send messages to any channel.",
				Parameters:  json.RawMessage(`{"channel_id":"*","content":"*"}`),
			},
			{
				ID:          "tpl_discord_create_channel",
				ActionType:  "discord.create_channel",
				Name:        "Create channels",
				Description: "Agent can create channels in any guild.",
				Parameters:  json.RawMessage(`{"guild_id":"*","name":"*","type":"*","parent_id":"*","topic":"*"}`),
			},
			{
				ID:          "tpl_discord_manage_roles",
				ActionType:  "discord.manage_roles",
				Name:        "Manage roles",
				Description: "Agent can assign or remove roles from members.",
				Parameters:  json.RawMessage(`{"guild_id":"*","user_id":"*","role_id":"*","action":"*"}`),
			},
			{
				ID:          "tpl_discord_create_event",
				ActionType:  "discord.create_event",
				Name:        "Create events",
				Description: "Agent can create scheduled events in any guild.",
				Parameters:  json.RawMessage(`{"guild_id":"*","name":"*","description":"*","scheduled_start_time":"*","scheduled_end_time":"*","privacy_level":"*","entity_type":"*","channel_id":"*","entity_metadata":"*"}`),
			},
			{
				ID:          "tpl_discord_ban_user",
				ActionType:  "discord.ban_user",
				Name:        "Ban users",
				Description: "Agent can ban users from guilds.",
				Parameters:  json.RawMessage(`{"guild_id":"*","user_id":"*","delete_message_seconds":"*"}`),
			},
			{
				ID:          "tpl_discord_kick_user",
				ActionType:  "discord.kick_user",
				Name:        "Kick users",
				Description: "Agent can kick users from guilds.",
				Parameters:  json.RawMessage(`{"guild_id":"*","user_id":"*"}`),
			},
			{
				ID:          "tpl_discord_pin_message",
				ActionType:  "discord.pin_message",
				Name:        "Pin messages",
				Description: "Agent can pin messages in any channel.",
				Parameters:  json.RawMessage(`{"channel_id":"*","message_id":"*"}`),
			},
			{
				ID:          "tpl_discord_unpin_message",
				ActionType:  "discord.unpin_message",
				Name:        "Unpin messages",
				Description: "Agent can unpin messages in any channel.",
				Parameters:  json.RawMessage(`{"channel_id":"*","message_id":"*"}`),
			},
			{
				ID:          "tpl_discord_list_channels",
				ActionType:  "discord.list_channels",
				Name:        "List channels",
				Description: "Agent can list channels in any guild.",
				Parameters:  json.RawMessage(`{"guild_id":"*"}`),
			},
			{
				ID:          "tpl_discord_list_members",
				ActionType:  "discord.list_members",
				Name:        "List members",
				Description: "Agent can list members of any guild.",
				Parameters:  json.RawMessage(`{"guild_id":"*","limit":"*","after":"*"}`),
			},
			{
				ID:          "tpl_discord_create_thread",
				ActionType:  "discord.create_thread",
				Name:        "Create threads",
				Description: "Agent can create threads in any channel.",
				Parameters:  json.RawMessage(`{"channel_id":"*","name":"*","message_id":"*","auto_archive_duration":"*"}`),
			},
		},
	}
}

// Actions returns the registered action handlers keyed by action_type.
func (c *DiscordConnector) Actions() map[string]connectors.Action {
	return map[string]connectors.Action{
		"discord.send_message":   &sendMessageAction{conn: c},
		"discord.create_channel": &createChannelAction{conn: c},
		"discord.manage_roles":   &manageRolesAction{conn: c},
		"discord.create_event":   &createEventAction{conn: c},
		"discord.ban_user":       &banUserAction{conn: c},
		"discord.kick_user":      &kickUserAction{conn: c},
		"discord.pin_message":    &pinMessageAction{conn: c},
		"discord.unpin_message":  &unpinMessageAction{conn: c},
		"discord.list_channels":  &listChannelsAction{conn: c},
		"discord.list_members":   &listMembersAction{conn: c},
		"discord.create_thread":  &createThreadAction{conn: c},
	}
}

// ValidateCredentials checks that the provided credentials contain a
// non-empty bot_token.
func (c *DiscordConnector) ValidateCredentials(_ context.Context, creds connectors.Credentials) error {
	token, ok := creds.Get(credKeyToken)
	if !ok || token == "" {
		return &connectors.ValidationError{Message: "missing required credential: bot_token"}
	}
	return nil
}

// discordErrorResponse is the standard Discord API error envelope.
type discordErrorResponse struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// doRequest is the shared request lifecycle for Discord API calls. It sends
// the request with Bot auth, handles rate limiting and timeouts, and
// unmarshals the response into dest (if non-nil, for 2xx with body).
// For endpoints that return 204 No Content, pass nil for dest.
func (c *DiscordConnector) doRequest(ctx context.Context, method, path string, creds connectors.Credentials, body any, dest any) error {
	token, ok := creds.Get(credKeyToken)
	if !ok || token == "" {
		return &connectors.ValidationError{Message: "bot_token credential is missing or empty"}
	}

	var bodyReader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("marshaling request body: %w", err)
		}
		bodyReader = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("Authorization", "Bot "+token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.client.Do(req)
	if err != nil {
		if connectors.IsTimeout(err) {
			return &connectors.TimeoutError{Message: fmt.Sprintf("Discord API request timed out: %v", err)}
		}
		if errors.Is(err, context.Canceled) {
			return &connectors.TimeoutError{Message: "Discord API request canceled"}
		}
		return &connectors.ExternalError{Message: fmt.Sprintf("Discord API request failed: %v", err)}
	}
	defer resp.Body.Close()

	// Discord returns 429 for rate limiting with a Retry-After header.
	if resp.StatusCode == http.StatusTooManyRequests {
		retryAfter := connectors.ParseRetryAfter(resp.Header.Get("Retry-After"), defaultRetryAfter)
		return &connectors.RateLimitError{
			Message:    "Discord API rate limit exceeded",
			RetryAfter: retryAfter,
		}
	}

	respBody, err := io.ReadAll(io.LimitReader(resp.Body, maxResponseBytes))
	if err != nil {
		return &connectors.ExternalError{Message: fmt.Sprintf("reading response body: %v", err)}
	}

	// 204 No Content — success with no body.
	if resp.StatusCode == http.StatusNoContent {
		return nil
	}

	// Auth errors.
	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		var errResp discordErrorResponse
		_ = json.Unmarshal(respBody, &errResp)
		msg := errResp.Message
		if msg == "" {
			msg = fmt.Sprintf("HTTP %d", resp.StatusCode)
		}
		return &connectors.AuthError{Message: fmt.Sprintf("Discord auth error: %s", msg)}
	}

	// Other error status codes.
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errResp discordErrorResponse
		_ = json.Unmarshal(respBody, &errResp)
		msg := errResp.Message
		if msg == "" {
			msg = fmt.Sprintf("HTTP %d", resp.StatusCode)
		}
		return &connectors.ExternalError{
			StatusCode: resp.StatusCode,
			Message:    fmt.Sprintf("Discord API error: %s", msg),
		}
	}

	// Success with body — unmarshal if dest is provided.
	if dest != nil && len(respBody) > 0 {
		if err := json.Unmarshal(respBody, dest); err != nil {
			return &connectors.ExternalError{
				StatusCode: resp.StatusCode,
				Message:    "failed to decode Discord API response",
			}
		}
	}

	return nil
}

// validateSnowflake checks that a string looks like a valid Discord snowflake ID.
func validateSnowflake(value, paramName string) error {
	if value == "" {
		return &connectors.ValidationError{Message: fmt.Sprintf("missing required parameter: %s", paramName)}
	}
	// Snowflakes are numeric strings, typically 17-20 digits.
	for _, ch := range value {
		if ch < '0' || ch > '9' {
			return &connectors.ValidationError{
				Message: fmt.Sprintf("invalid %s %q: must be a numeric Discord snowflake ID", paramName, value),
			}
		}
	}
	return nil
}
