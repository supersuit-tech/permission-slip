// Package discord implements the Discord connector for the Permission Slip
// connector execution layer. It uses the Discord REST API v10 with plain
// net/http (no third-party SDK) to keep the dependency footprint minimal.
package discord

import (
	_ "embed"
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
	credKeyToken = "bot_token"

	// defaultRetryAfter is used when Discord returns a rate limit response
	// without a Retry-After header (or an unparseable one).
	defaultRetryAfter = 5 * time.Second

	// maxResponseBytes caps the Discord API response body at 10 MB.
	maxResponseBytes = 10 << 20 // 10 MB
)

// OAuthScopes is the canonical list of OAuth scopes required by the Discord
// connector. This is the single source of truth — referenced by both the
// connector manifest and the built-in OAuth provider registration.
var OAuthScopes = []string{
	"bot",
	"guilds",
}

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

//go:embed logo.svg
var logoSVG string

// Manifest returns the connector's metadata manifest.
func (c *DiscordConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "discord",
		Name:        "Discord",
		Description: "Discord integration for community management and messaging",
		LogoSVG:     logoSVG,
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
							"pattern": "^[0-9]+$",
							"description": "Discord channel ID — a numeric snowflake (e.g. 1234567890123456789)"
						},
						"content": {
							"type": "string",
							"minLength": 1,
							"maxLength": 2000,
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
							"pattern": "^[0-9]+$",
							"description": "Discord guild (server) ID — a numeric snowflake (e.g. 1234567890123456789)"
						},
						"name": {
							"type": "string",
							"minLength": 2,
							"maxLength": 100,
							"pattern": "^[a-z0-9_-]+$",
							"description": "Channel name (2-100 characters, lowercase, no spaces — e.g. general-chat)"
						},
						"type": {
							"type": "integer",
							"enum": [0, 2, 4],
							"default": 0,
							"description": "Channel type: 0 = text, 2 = voice, 4 = category"
						},
						"parent_id": {
							"type": "string",
							"pattern": "^[0-9]+$",
							"description": "Parent category channel ID (snowflake)"
						},
						"topic": {
							"type": "string",
							"maxLength": 1024,
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
							"pattern": "^[0-9]+$",
							"description": "Discord guild (server) ID (snowflake)"
						},
						"user_id": {
							"type": "string",
							"pattern": "^[0-9]+$",
							"description": "Discord user ID (snowflake)"
						},
						"role_id": {
							"type": "string",
							"pattern": "^[0-9]+$",
							"description": "Discord role ID (snowflake) — use discord.list_roles to discover valid IDs"
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
							"pattern": "^[0-9]+$",
							"description": "Discord guild (server) ID (snowflake)"
						},
						"name": {
							"type": "string",
							"minLength": 1,
							"maxLength": 100,
							"description": "Event name (1-100 characters)"
						},
						"description": {
							"type": "string",
							"maxLength": 1000,
							"description": "Event description (up to 1000 characters)"
						},
						"scheduled_start_time": {
							"type": "string",
							"format": "date-time",
							"description": "ISO 8601 timestamp for event start (e.g. 2026-04-01T18:00:00Z)"
						},
						"scheduled_end_time": {
							"type": "string",
							"format": "date-time",
							"description": "ISO 8601 timestamp for event end (required for external events)"
						},
						"privacy_level": {
							"type": "integer",
							"enum": [2],
							"default": 2,
							"description": "Privacy level: 2 = guild only (currently the only supported value)"
						},
						"entity_type": {
							"type": "integer",
							"enum": [1, 2, 3],
							"description": "Entity type: 1 = stage instance, 2 = voice channel, 3 = external location"
						},
						"channel_id": {
							"type": "string",
							"pattern": "^[0-9]+$",
							"description": "Channel ID for stage/voice events (required when entity_type is 1 or 2)"
						},
						"entity_metadata": {
							"type": "object",
							"properties": {
								"location": {
									"type": "string",
									"description": "Location for external events (required when entity_type is 3)"
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
							"pattern": "^[0-9]+$",
							"description": "Discord guild (server) ID (snowflake)"
						},
						"user_id": {
							"type": "string",
							"pattern": "^[0-9]+$",
							"description": "Discord user ID to ban (snowflake)"
						},
						"delete_message_seconds": {
							"type": "integer",
							"minimum": 0,
							"maximum": 604800,
							"default": 0,
							"description": "Seconds of message history to delete (0 = none, 604800 = 7 days)"
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
							"pattern": "^[0-9]+$",
							"description": "Discord guild (server) ID (snowflake)"
						},
						"user_id": {
							"type": "string",
							"pattern": "^[0-9]+$",
							"description": "Discord user ID to kick (snowflake)"
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
							"pattern": "^[0-9]+$",
							"description": "Discord channel ID (snowflake)"
						},
						"message_id": {
							"type": "string",
							"pattern": "^[0-9]+$",
							"description": "Discord message ID to pin (snowflake)"
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
							"pattern": "^[0-9]+$",
							"description": "Discord channel ID (snowflake)"
						},
						"message_id": {
							"type": "string",
							"pattern": "^[0-9]+$",
							"description": "Discord message ID to unpin (snowflake)"
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
							"pattern": "^[0-9]+$",
							"description": "Discord guild (server) ID (snowflake)"
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
							"pattern": "^[0-9]+$",
							"description": "Discord guild (server) ID (snowflake)"
						},
						"limit": {
							"type": "integer",
							"minimum": 1,
							"maximum": 1000,
							"default": 100,
							"description": "Max members to return (1-1000)"
						},
						"after": {
							"type": "string",
							"pattern": "^[0-9]+$",
							"description": "User ID to paginate after (snowflake)"
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
							"pattern": "^[0-9]+$",
							"description": "Discord channel ID to create the thread in (snowflake)"
						},
						"name": {
							"type": "string",
							"minLength": 1,
							"maxLength": 100,
							"description": "Thread name (1-100 characters)"
						},
						"message_id": {
							"type": "string",
							"pattern": "^[0-9]+$",
							"description": "Message ID to start the thread from (omit for a thread without a starter message)"
						},
						"auto_archive_duration": {
							"type": "integer",
							"enum": [0, 60, 1440, 4320, 10080],
							"default": 1440,
							"description": "Minutes of inactivity before auto-archive: 0 = server default, 60 = 1h, 1440 = 24h, 4320 = 3d, 10080 = 7d"
						}
					}
				}`)),
			},
			{
				ActionType:  "discord.list_roles",
				Name:        "List Roles",
				Description: "List roles in a Discord guild",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["guild_id"],
					"properties": {
						"guild_id": {
							"type": "string",
							"pattern": "^[0-9]+$",
							"description": "Discord guild (server) ID (snowflake)"
						}
					}
				}`)),
			},
		},
		RequiredCredentials: []connectors.ManifestCredential{
			{Service: "discord", AuthType: "custom", InstructionsURL: "https://discord.com/developers/applications"},
		},
		Templates: []connectors.ManifestTemplate{
			{
				ID:          "tpl_discord_send_message",
				ActionType:  "discord.send_message",
				Name:        "Send messages freely",
				Description: "Agent can send messages to any channel the bot has access to.",
				Parameters:  json.RawMessage(`{"channel_id":"*","content":"*"}`),
			},
			{
				ID:          "tpl_discord_create_channel",
				ActionType:  "discord.create_channel",
				Name:        "Create channels",
				Description: "Agent can create text, voice, or category channels in any guild. Requires Manage Channels permission on the bot.",
				Parameters:  json.RawMessage(`{"guild_id":"*","name":"*","type":"*","parent_id":"*","topic":"*"}`),
			},
			{
				ID:          "tpl_discord_manage_roles",
				ActionType:  "discord.manage_roles",
				Name:        "Manage roles",
				Description: "Agent can assign or remove roles from members. The bot's highest role must be above the target role.",
				Parameters:  json.RawMessage(`{"guild_id":"*","user_id":"*","role_id":"*","action":"*"}`),
			},
			{
				ID:          "tpl_discord_create_event",
				ActionType:  "discord.create_event",
				Name:        "Create events",
				Description: "Agent can create scheduled events (stage, voice, or external) in any guild. Requires Manage Events permission.",
				Parameters:  json.RawMessage(`{"guild_id":"*","name":"*","description":"*","scheduled_start_time":"*","scheduled_end_time":"*","privacy_level":"*","entity_type":"*","channel_id":"*","entity_metadata":"*"}`),
			},
			{
				ID:          "tpl_discord_moderate",
				ActionType:  "discord.ban_user",
				Name:        "Ban users",
				Description: "Agent can ban users and optionally delete their recent messages. Requires Ban Members permission.",
				Parameters:  json.RawMessage(`{"guild_id":"*","user_id":"*","delete_message_seconds":"*"}`),
			},
			{
				ID:          "tpl_discord_kick_user",
				ActionType:  "discord.kick_user",
				Name:        "Kick users",
				Description: "Agent can kick users from guilds. Requires Kick Members permission.",
				Parameters:  json.RawMessage(`{"guild_id":"*","user_id":"*"}`),
			},
			{
				ID:          "tpl_discord_pin_message",
				ActionType:  "discord.pin_message",
				Name:        "Pin messages",
				Description: "Agent can pin messages in any channel. Requires Manage Messages permission.",
				Parameters:  json.RawMessage(`{"channel_id":"*","message_id":"*"}`),
			},
			{
				ID:          "tpl_discord_unpin_message",
				ActionType:  "discord.unpin_message",
				Name:        "Unpin messages",
				Description: "Agent can unpin messages in any channel. Requires Manage Messages permission.",
				Parameters:  json.RawMessage(`{"channel_id":"*","message_id":"*"}`),
			},
			{
				ID:          "tpl_discord_list_channels",
				ActionType:  "discord.list_channels",
				Name:        "List channels",
				Description: "Agent can list channels in any guild (read-only).",
				Parameters:  json.RawMessage(`{"guild_id":"*"}`),
			},
			{
				ID:          "tpl_discord_list_members",
				ActionType:  "discord.list_members",
				Name:        "List members",
				Description: "Agent can list members of any guild (read-only). Requires Server Members Intent enabled on the bot.",
				Parameters:  json.RawMessage(`{"guild_id":"*","limit":"*","after":"*"}`),
			},
			{
				ID:          "tpl_discord_create_thread",
				ActionType:  "discord.create_thread",
				Name:        "Create threads",
				Description: "Agent can create threads in any channel. Requires Create Public Threads permission.",
				Parameters:  json.RawMessage(`{"channel_id":"*","name":"*","message_id":"*","auto_archive_duration":"*"}`),
			},
			{
				ID:          "tpl_discord_list_roles",
				ActionType:  "discord.list_roles",
				Name:        "List roles",
				Description: "Agent can list roles in any guild (read-only). Useful for discovering role IDs before assigning them.",
				Parameters:  json.RawMessage(`{"guild_id":"*"}`),
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
		"discord.list_roles":     &listRolesAction{conn: c},
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

// mapDiscordError converts a Discord API error code and HTTP status to the
// appropriate connector error type with user-friendly, actionable messages.
func mapDiscordError(statusCode int, errResp discordErrorResponse) error {
	// Map well-known Discord JSON error codes first.
	switch errResp.Code {
	case 10003: // Unknown Channel
		return &connectors.ExternalError{StatusCode: statusCode, Message: "Discord channel not found — verify the channel ID is correct and the bot has access"}
	case 10004: // Unknown Guild
		return &connectors.ExternalError{StatusCode: statusCode, Message: "Discord guild not found — verify the guild (server) ID is correct and the bot is a member"}
	case 10007: // Unknown Member
		return &connectors.ExternalError{StatusCode: statusCode, Message: "Discord member not found — verify the user is a member of the guild"}
	case 10008: // Unknown Message
		return &connectors.ExternalError{StatusCode: statusCode, Message: "Discord message not found — verify the message ID exists in the specified channel"}
	case 10011: // Unknown Role
		return &connectors.ExternalError{StatusCode: statusCode, Message: "Discord role not found — use discord.list_roles to find valid role IDs"}
	case 10013: // Unknown User
		return &connectors.ExternalError{StatusCode: statusCode, Message: "Discord user not found — verify the user ID is correct"}
	case 30003: // Max pins reached
		return &connectors.ExternalError{StatusCode: statusCode, Message: "this channel has reached the maximum of 50 pinned messages — unpin one first"}
	case 40001: // Unauthorized
		return &connectors.AuthError{Message: "Discord request unauthorized — verify the bot token is valid and has not been regenerated"}
	case 50001: // Missing Access
		return &connectors.AuthError{Message: "bot is missing access to this resource — check that the bot has been invited to the guild and has the required channel permissions"}
	case 50013: // Missing Permissions
		return &connectors.AuthError{Message: "bot is missing a required permission — check the bot's role permissions in Discord server settings"}
	case 50028: // Invalid role
		return &connectors.ExternalError{StatusCode: statusCode, Message: "invalid role — the bot's highest role must be above the target role in the role hierarchy"}
	case 50035: // Invalid form body
		return &connectors.ValidationError{Message: fmt.Sprintf("Discord rejected the request body: %s — check parameter formats and constraints", errResp.Message)}
	}

	// Fall back to HTTP status classification.
	msg := errResp.Message
	if msg == "" {
		msg = fmt.Sprintf("HTTP %d", statusCode)
	}

	switch {
	case statusCode == http.StatusUnauthorized:
		return &connectors.AuthError{Message: "Discord auth error: invalid bot token — generate a new token at https://discord.com/developers/applications"}
	case statusCode == http.StatusForbidden:
		return &connectors.AuthError{Message: fmt.Sprintf("Discord auth error: %s — check the bot's permissions in server settings", msg)}
	default:
		return &connectors.ExternalError{
			StatusCode: statusCode,
			Message:    fmt.Sprintf("Discord API error: %s", msg),
		}
	}
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

	// Non-success status codes — use mapDiscordError for actionable messages.
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		var errResp discordErrorResponse
		_ = json.Unmarshal(respBody, &errResp)
		return mapDiscordError(resp.StatusCode, errResp)
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
				Message: fmt.Sprintf("invalid %s %q: must be a numeric Discord snowflake ID (e.g. 1234567890123456789)", paramName, value),
			}
		}
	}
	return nil
}
