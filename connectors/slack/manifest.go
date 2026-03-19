package slack

import (
	_ "embed"
	"encoding/json"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// Manifest returns the connector's metadata manifest. Used by the server to
// auto-seed DB rows on startup, replacing manual seed.go files.
//go:embed logo.svg
var logoSVG string

func (c *SlackConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "slack",
		Name:        "Slack",
		Description: "Slack integration for team communication",
		Status:      "early_preview",
		LogoSVG:     logoSVG,
		Actions: []connectors.ManifestAction{
			{
				ActionType:  "slack.send_message",
				Name:        "Send Message",
				Description: "Send a message to a Slack channel",
				RiskLevel:   "low",
				Preview: &connectors.ActionPreview{
					Layout: "message",
					Fields: map[string]string{"to": "channel", "body": "message"},
				},
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
							"description": "Message text (supports Slack mrkdwn formatting)",
							"x-ui": {"widget": "textarea"}
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
							"description": "Message text (supports Slack mrkdwn formatting)",
							"x-ui": {"widget": "textarea"}
						},
						"post_at": {
							"type": "string",
							"format": "date-time",
							"description": "When the message should be sent in RFC 3339 format (e.g. 2026-03-20T09:00:00Z)"
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
							"description": "File content as text",
							"x-ui": {"widget": "textarea"}
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
			{
				ActionType:  "slack.send_dm",
				Name:        "Send Direct Message",
				Description: "Send a direct message to a Slack user",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["user_id", "message"],
					"properties": {
						"user_id": {
							"type": "string",
							"description": "User ID to send the DM to (e.g. U01234567)"
						},
						"message": {
							"type": "string",
							"description": "Message text (supports Slack mrkdwn formatting)",
							"x-ui": {"widget": "textarea"}
						}
					}
				}`)),
			},
			{
				ActionType:  "slack.update_message",
				Name:        "Update Message",
				Description: "Edit an existing message in a Slack channel",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["channel", "ts", "message"],
					"properties": {
						"channel": {
							"type": "string",
							"description": "Channel ID containing the message (e.g. C01234567)"
						},
						"ts": {
							"type": "string",
							"description": "Timestamp of the message to update (e.g. 1234567890.123456)"
						},
						"message": {
							"type": "string",
							"description": "New message text (supports Slack mrkdwn formatting)",
							"x-ui": {"widget": "textarea"}
						}
					}
				}`)),
			},
			{
				ActionType:  "slack.delete_message",
				Name:        "Delete Message",
				Description: "Delete a message from a Slack channel",
				RiskLevel:   "high",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["channel", "ts"],
					"properties": {
						"channel": {
							"type": "string",
							"description": "Channel ID containing the message (e.g. C01234567)"
						},
						"ts": {
							"type": "string",
							"description": "Timestamp of the message to delete (e.g. 1234567890.123456)"
						}
					}
				}`)),
			},
			{
				ActionType:  "slack.list_users",
				Name:        "List Users",
				Description: "List workspace users visible to the bot",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"properties": {
						"limit": {
							"type": "integer",
							"default": 100,
							"description": "Max users to return (1-1000)"
						},
						"cursor": {
							"type": "string",
							"description": "Pagination cursor from a previous response"
						}
					}
				}`)),
			},
			{
				ActionType:  "slack.search_messages",
				Name:        "Search Messages",
				Description: "Search messages across Slack channels (requires a user token with search:read.* scopes; bot tokens are not supported by Slack for this endpoint)",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["query"],
					"properties": {
						"query": {
							"type": "string",
							"description": "Search query (supports Slack search modifiers like in:#channel, from:@user)"
						},
						"count": {
							"type": "integer",
							"default": 20,
							"description": "Max results per page (1-100)"
						},
						"page": {
							"type": "integer",
							"default": 1,
							"description": "Page number for pagination"
						},
						"sort": {
							"type": "string",
							"description": "Sort order: score (relevance) or timestamp"
						}
					}
				}`)),
			},
		},
		RequiredCredentials: []connectors.ManifestCredential{
			{
				Service:       "slack",
				AuthType:      "oauth2",
				OAuthProvider: "slack",
				OAuthScopes:   append(OAuthScopes, OAuthUserScopes...),
			},
			{Service: "slack_bot", AuthType: "custom", InstructionsURL: "https://api.slack.com/tutorials/tracks/getting-a-token"},
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
			{
				ID:          "tpl_slack_send_dm",
				ActionType:  "slack.send_dm",
				Name:        "Send direct messages",
				Description: "Agent can send direct messages to any user.",
				Parameters:  json.RawMessage(`{"user_id":"*","message":"*"}`),
			},
			{
				ID:          "tpl_slack_update_message",
				ActionType:  "slack.update_message",
				Name:        "Update messages",
				Description: "Agent can edit messages in any channel.",
				Parameters:  json.RawMessage(`{"channel":"*","ts":"*","message":"*"}`),
			},
			{
				ID:          "tpl_slack_delete_message",
				ActionType:  "slack.delete_message",
				Name:        "Delete messages",
				Description: "Agent can delete messages from any channel.",
				Parameters:  json.RawMessage(`{"channel":"*","ts":"*"}`),
			},
			{
				ID:          "tpl_slack_list_users",
				ActionType:  "slack.list_users",
				Name:        "List users",
				Description: "Agent can list workspace users.",
				Parameters:  json.RawMessage(`{"limit":"*","cursor":"*"}`),
			},
			{
				ID:          "tpl_slack_search_messages",
				ActionType:  "slack.search_messages",
				Name:        "Search messages",
				Description: "Agent can search messages across channels.",
				Parameters:  json.RawMessage(`{"query":"*","count":"*","page":"*","sort":"*"}`),
			},
		},
	}
}
