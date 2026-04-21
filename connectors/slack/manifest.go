package slack

import (
	_ "embed"
	"encoding/json"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// Manifest returns the connector's metadata manifest. Used by the server to
// auto-seed DB rows on startup, replacing manual seed.go files.
//
//go:embed logo.svg
var logoSVG string

// neverExpire is the standing-approval spec applied to templates that opt in
// to a never-expiring auto-approval when the template is applied. Using the
// shared value keeps each template's declaration to a single line and makes
// the opt-in explicit rather than relying on a cross-cutting default.
var neverExpire = &connectors.ManifestStandingApproval{}

func (c *SlackConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "slack",
		Name:        "Slack",
		Description: "Slack integration for team communication via user OAuth (xoxp-). Actions run as the authorizing Slack user.",
		Status:      "early_preview",
		LogoSVG:     logoSVG,
		Actions: []connectors.ManifestAction{
			{
				ActionType:      "slack.send_message",
				Name:            "Send Message",
				Description:     "Send a message to a Slack channel as the authorizing user.",
				RiskLevel:       "low",
				DisplayTemplate: "Send message to {{channel_name}}",
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
							"description": "Slack channel to post to (picker uses channel ID)",
							"x-ui": {
								"widget": "remote-select",
								"remote_select_options_path": "/v1/agents/{agent_id}/connectors/{connector_id}/channels",
								"remote_select_id_key": "id",
								"remote_select_label_key": "display_label",
								"remote_select_fallback_placeholder": "Channel ID (e.g. C01234567)",
								"help_text": "Choose a channel or enter an ID. Private channels and DMs need your profile email to match Slack."
							}
						},
						"message": {
							"type": "string",
							"description": "Message text (supports Slack mrkdwn formatting)",
							"x-ui": {"widget": "textarea", "placeholder": "Enter the message text"}
						}
					}
				}`)),
			},
			{
				ActionType:      "slack.create_channel",
				Name:            "Create Channel",
				Description:     "Create a new Slack channel",
				RiskLevel:       "medium",
				DisplayTemplate: "Create channel #{{name}}",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["name"],
					"properties": {
						"name": {
							"type": "string",
							"description": "Channel name (lowercase, no spaces, max 80 chars)",
							"x-ui": {"placeholder": "new-channel-name", "help_text": "Lowercase, no spaces, max 80 characters (e.g. project-updates)"}
						},
						"is_private": {
							"type": "boolean",
							"default": false,
							"description": "Create as a private channel",
							"x-ui": {"label": "Private channel", "help_text": "Private channels are only visible to invited members"}
						}
					}
				}`)),
			},
			{
				ActionType:      "slack.list_channels",
				Name:            "List Channels",
				Description:     "List Slack channels via conversations.list, merged with the authorizing user's DMs and private conversations from users.conversations when a matching profile email is available. Returns all channel types (public, private, group DMs, DMs) by default when email is set.",
				RiskLevel:       "low",
				DisplayTemplate: "List Slack channels",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"properties": {
						"types": {
							"type": "string",
							"default": "public_channel,private_channel,mpim,im",
							"enum": ["public_channel", "private_channel", "mpim", "im"],
							"description": "Comma-separated channel types: public_channel, private_channel, mpim, im. Defaults to all types when a user email is available; falls back to public_channel only when no email is set. im/mpim/private_channel results are filtered to the authorizing user; user-token merge fills in human-only DMs when configured.",
							"x-ui": {"label": "Channel types", "widget": "multi-select", "help_text": "public_channel, private_channel, mpim (group DMs), im (direct messages)"}
						},
						"limit": {
							"type": "integer",
							"default": 100,
							"description": "Max channels to return (1-1000)",
							"x-ui": {"label": "Max results", "help_text": "Maximum number of channels to return (1–1000, default 100)"}
						},
						"cursor": {
							"type": "string",
							"description": "Pagination cursor from a previous response",
							"x-ui": {"hidden": true}
						},
						"exclude_archived": {
							"type": "boolean",
							"default": true,
							"description": "Exclude archived channels from results",
							"x-ui": {"label": "Exclude archived", "help_text": "When enabled, archived channels are hidden from results"}
						}
					}
				}`)),
			},
			{
				ActionType:      "slack.read_channel_messages",
				Name:            "Read Channel Messages",
				Description:     "Read recent messages from a Slack channel, DM (D…), or group DM (G…).",
				RiskLevel:       "low",
				DisplayTemplate: "Read messages from {{channel_name}}",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["channel"],
					"properties": {
						"channel": {
							"type": "string",
							"description": "Channel ID: C… (channel), D… (DM), or G… (group DM)",
							"x-ui": {
								"widget": "remote-select",
								"remote_select_options_path": "/v1/agents/{agent_id}/connectors/{connector_id}/channels",
								"remote_select_id_key": "id",
								"remote_select_label_key": "display_label",
								"remote_select_fallback_placeholder": "Channel ID (e.g. C01234567)",
								"help_text": "Choose a channel or enter an ID. Private channels and DMs need your profile email to match Slack."
							}
						},
						"limit": {
							"type": "integer",
							"default": 20,
							"description": "Max messages to return (1-1000)",
							"x-ui": {"label": "Max messages", "help_text": "Maximum number of messages to return (1–1000, default 20)"}
						},
						"oldest": {
							"type": "string",
							"format": "date-time",
							"description": "Only messages after this date/time (RFC 3339)",
							"x-ui": {"label": "After", "widget": "datetime", "help_text": "Only include messages sent after this date and time", "datetime_range_pair": "latest", "datetime_range_role": "lower"}
						},
						"latest": {
							"type": "string",
							"format": "date-time",
							"description": "Only messages before this date/time (RFC 3339)",
							"x-ui": {"label": "Before", "widget": "datetime", "help_text": "Only include messages sent before this date and time", "datetime_range_pair": "oldest", "datetime_range_role": "upper"}
						},
						"cursor": {
							"type": "string",
							"description": "Pagination cursor from a previous response",
							"x-ui": {"hidden": true}
						}
					}
				}`)),
			},
			{
				ActionType:      "slack.read_thread",
				Name:            "Read Thread",
				Description:     "Read replies in a Slack thread. For threads in DMs or group DMs, uses the authorizing user's OAuth token when available.",
				RiskLevel:       "low",
				DisplayTemplate: "Read thread in {{channel_name}}",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["channel"],
					"properties": {
						"channel": {
							"type": "string",
							"description": "Channel ID containing the thread: C…, D…, or G…",
							"x-ui": {
								"widget": "remote-select",
								"remote_select_options_path": "/v1/agents/{agent_id}/connectors/{connector_id}/channels",
								"remote_select_id_key": "id",
								"remote_select_label_key": "display_label",
								"remote_select_fallback_placeholder": "Channel ID (e.g. C01234567)",
								"help_text": "Choose a channel or enter an ID. Private channels and DMs need your profile email to match Slack."
							}
						},
						"thread_ts": {
							"type": "string",
							"description": "Timestamp of the parent message (e.g. 1234567890.123456)",
							"x-ui": {"hidden": true}
						},
						"limit": {
							"type": "integer",
							"default": 50,
							"description": "Max replies to return (1-1000)",
							"x-ui": {"label": "Max replies", "help_text": "Maximum number of replies to return (1–1000, default 50)"}
						},
						"cursor": {
							"type": "string",
							"description": "Pagination cursor from a previous response",
							"x-ui": {"hidden": true}
						}
					}
				}`)),
			},
			{
				ActionType:      "slack.schedule_message",
				Name:            "Schedule Message",
				Description:     "Schedule a message for future delivery to a Slack channel as the authorizing user.",
				RiskLevel:       "low",
				DisplayTemplate: "Schedule message to {{channel_name}}",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["channel", "message", "post_at"],
					"properties": {
						"channel": {
							"type": "string",
							"description": "Slack channel to schedule in (picker uses channel ID)",
							"x-ui": {
								"widget": "remote-select",
								"remote_select_options_path": "/v1/agents/{agent_id}/connectors/{connector_id}/channels",
								"remote_select_id_key": "id",
								"remote_select_label_key": "display_label",
								"remote_select_fallback_placeholder": "Channel ID (e.g. C01234567)",
								"help_text": "Choose a channel or enter an ID. Private channels and DMs need your profile email to match Slack."
							}
						},
						"message": {
							"type": "string",
							"description": "Message text (supports Slack mrkdwn formatting)",
							"x-ui": {"widget": "textarea", "placeholder": "Enter the message text"}
						},
						"post_at": {
							"type": "string",
							"format": "date-time",
							"description": "When the message should be sent in RFC 3339 format (e.g. 2026-03-20T09:00:00Z)",
							"x-ui": {"label": "Send at", "help_text": "The date and time to deliver the message"}
						}
					}
				}`)),
			},
			{
				ActionType:      "slack.set_topic",
				OperationType:   "edit",
				Name:            "Set Topic",
				Description:     "Update a Slack channel's topic",
				RiskLevel:       "medium",
				DisplayTemplate: "Set topic in {{channel_name}}",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["channel", "topic"],
					"properties": {
						"channel": {
							"type": "string",
							"description": "Channel ID (e.g. C01234567)",
							"x-ui": {
								"widget": "remote-select",
								"remote_select_options_path": "/v1/agents/{agent_id}/connectors/{connector_id}/channels",
								"remote_select_id_key": "id",
								"remote_select_label_key": "display_label",
								"remote_select_fallback_placeholder": "Channel ID (e.g. C01234567)",
								"help_text": "Choose a channel or enter an ID. Private channels and DMs need your profile email to match Slack."
							}
						},
						"topic": {
							"type": "string",
							"description": "New channel topic (max 250 characters)",
							"x-ui": {"placeholder": "Enter the new topic", "help_text": "The new topic text (max 250 characters)"}
						}
					}
				}`)),
			},
			{
				ActionType:      "slack.invite_to_channel",
				Name:            "Invite to Channel",
				Description:     "Invite one or more users to a Slack channel",
				RiskLevel:       "medium",
				DisplayTemplate: "Invite users to {{channel_name}}",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["channel", "users"],
					"properties": {
						"channel": {
							"type": "string",
							"description": "Channel ID (e.g. C01234567)",
							"x-ui": {
								"widget": "remote-select",
								"remote_select_options_path": "/v1/agents/{agent_id}/connectors/{connector_id}/channels",
								"remote_select_id_key": "id",
								"remote_select_label_key": "display_label",
								"remote_select_fallback_placeholder": "Channel ID (e.g. C01234567)",
								"help_text": "Choose a channel or enter an ID. Private channels and DMs need your profile email to match Slack."
							}
						},
						"users": {
							"type": "string",
							"description": "Comma-separated list of user IDs to invite (e.g. U01234567,U09876543)",
							"x-ui": {
								"widget": "remote-multi-select",
								"label": "Users",
								"remote_select_options_path": "/v1/agents/{agent_id}/connectors/{connector_id}/users",
								"remote_select_id_key": "id",
								"remote_select_label_key": "display_label",
								"remote_select_fallback_placeholder": "Comma-separated user IDs (e.g. U01234567,U09876543)",
								"help_text": "Select one or more users to invite."
							}
						}
					}
				}`)),
			},
			{
				ActionType:      "slack.upload_file",
				Name:            "Upload File",
				Description:     "Upload a file to a Slack channel",
				RiskLevel:       "low",
				DisplayTemplate: "Upload file to {{channel_name}}",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["channel", "filename", "content"],
					"properties": {
						"channel": {
							"type": "string",
							"description": "Channel ID (e.g. C01234567)",
							"x-ui": {
								"widget": "remote-select",
								"remote_select_options_path": "/v1/agents/{agent_id}/connectors/{connector_id}/channels",
								"remote_select_id_key": "id",
								"remote_select_label_key": "display_label",
								"remote_select_fallback_placeholder": "Channel ID (e.g. C01234567)",
								"help_text": "Choose a channel or enter an ID. Private channels and DMs need your profile email to match Slack."
							}
						},
						"filename": {
							"type": "string",
							"description": "Name of the file (e.g. report.csv)",
							"x-ui": {"label": "File name", "placeholder": "report.csv", "help_text": "The name for the uploaded file (e.g. report.csv)"}
						},
						"content": {
							"type": "string",
							"description": "File content as text",
							"x-ui": {"widget": "textarea", "label": "File content", "placeholder": "Paste or enter file content"}
						},
						"title": {
							"type": "string",
							"description": "Display title for the file (defaults to filename)",
							"x-ui": {"placeholder": "Optional display title", "help_text": "A display title shown in Slack (defaults to the file name)"}
						}
					}
				}`)),
			},
			{
				ActionType:      "slack.add_reaction",
				Name:            "Add Reaction",
				Description:     "Add an emoji reaction to a Slack message",
				RiskLevel:       "low",
				DisplayTemplate: "Add reaction in {{channel_name}}",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["channel", "name"],
					"properties": {
						"channel": {
							"type": "string",
							"description": "Channel ID containing the message (e.g. C01234567)",
							"x-ui": {
								"widget": "remote-select",
								"remote_select_options_path": "/v1/agents/{agent_id}/connectors/{connector_id}/channels",
								"remote_select_id_key": "id",
								"remote_select_label_key": "display_label",
								"remote_select_fallback_placeholder": "Channel ID (e.g. C01234567)",
								"help_text": "Choose a channel or enter an ID. Private channels and DMs need your profile email to match Slack."
							}
						},
						"timestamp": {
							"type": "string",
							"description": "Timestamp of the message to react to (e.g. 1234567890.123456)",
							"x-ui": {"hidden": true}
						},
						"name": {
							"type": "string",
							"description": "Emoji name without colons (e.g. thumbsup, white_check_mark)",
							"x-ui": {"label": "Emoji", "placeholder": "thumbsup", "help_text": "Emoji name without colons (e.g. thumbsup, heart, white_check_mark)"}
						}
					}
				}`)),
			},
			{
				ActionType:      "slack.send_dm",
				Name:            "Send Direct Message",
				Description:     "Send a direct message to a Slack user as the authorizing user.",
				RiskLevel:       "low",
				DisplayTemplate: "Send DM to {{user_name}}",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["user_id", "message"],
					"properties": {
						"user_id": {
							"type": "string",
							"description": "User ID to send the DM to (e.g. U01234567)",
							"x-ui": {
								"widget": "remote-select",
								"label": "Recipient",
								"remote_select_options_path": "/v1/agents/{agent_id}/connectors/{connector_id}/users",
								"remote_select_id_key": "id",
								"remote_select_label_key": "display_label",
								"remote_select_fallback_placeholder": "User ID (e.g. U01234567)",
								"help_text": "Choose a user or enter a user ID."
							}
						},
						"message": {
							"type": "string",
							"description": "Message text (supports Slack mrkdwn formatting)",
							"x-ui": {"widget": "textarea", "placeholder": "Enter the message text"}
						}
					}
				}`)),
			},
			{
				ActionType:      "slack.update_message",
				OperationType:   "edit",
				Name:            "Update Message",
				Description:     "Edit an existing message you are allowed to change.",
				RiskLevel:       "medium",
				DisplayTemplate: "Edit message in {{channel_name}}",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["channel", "message"],
					"properties": {
						"channel": {
							"type": "string",
							"description": "Channel ID containing the message (e.g. C01234567)",
							"x-ui": {
								"widget": "remote-select",
								"remote_select_options_path": "/v1/agents/{agent_id}/connectors/{connector_id}/channels",
								"remote_select_id_key": "id",
								"remote_select_label_key": "display_label",
								"remote_select_fallback_placeholder": "Channel ID (e.g. C01234567)",
								"help_text": "Choose a channel or enter an ID. Private channels and DMs need your profile email to match Slack."
							}
						},
						"ts": {
							"type": "string",
							"description": "Timestamp of the message to update (e.g. 1234567890.123456)",
							"x-ui": {"hidden": true}
						},
						"message": {
							"type": "string",
							"description": "New message text (supports Slack mrkdwn formatting)",
							"x-ui": {"widget": "textarea", "label": "New message", "placeholder": "Enter the updated message text"}
						}
					}
				}`)),
			},
			{
				ActionType:      "slack.delete_message",
				Name:            "Delete Message",
				Description:     "Delete a message you are allowed to remove.",
				RiskLevel:       "high",
				DisplayTemplate: "Delete message in {{channel_name}}",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["channel"],
					"properties": {
						"channel": {
							"type": "string",
							"description": "Channel ID containing the message (e.g. C01234567)",
							"x-ui": {
								"widget": "remote-select",
								"remote_select_options_path": "/v1/agents/{agent_id}/connectors/{connector_id}/channels",
								"remote_select_id_key": "id",
								"remote_select_label_key": "display_label",
								"remote_select_fallback_placeholder": "Channel ID (e.g. C01234567)",
								"help_text": "Choose a channel or enter an ID. Private channels and DMs need your profile email to match Slack."
							}
						},
						"ts": {
							"type": "string",
							"description": "Timestamp of the message to delete (e.g. 1234567890.123456)",
							"x-ui": {"hidden": true}
						}
					}
				}`)),
			},
			{
				ActionType:      "slack.list_users",
				Name:            "List Users",
				Description:     "List workspace users visible to the authorizing user",
				RiskLevel:       "low",
				DisplayTemplate: "List Slack users",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"properties": {
						"limit": {
							"type": "integer",
							"default": 100,
							"description": "Max users to return (1-1000)",
							"x-ui": {"label": "Max results", "help_text": "Maximum number of users to return (1–1000, default 100)"}
						},
						"cursor": {
							"type": "string",
							"description": "Pagination cursor from a previous response",
							"x-ui": {"hidden": true}
						}
					}
				}`)),
			},
			{
				ActionType:      "slack.search_messages",
				Name:            "Search Messages",
				Description:     "Search messages across Slack channels (requires search:read)",
				RiskLevel:       "low",
				DisplayTemplate: "Search {{channel_name}} for {{query}}",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["query"],
					"properties": {
						"query": {
							"type": "string",
							"description": "Search query (supports Slack search modifiers like in:#channel, from:@user)",
							"x-ui": {"placeholder": "search terms or in:#channel from:@user", "help_text": "Search query — supports Slack modifiers like in:#channel, from:@user"}
						},
						"channel": {
							"type": "string",
							"description": "Optional channel ID to scope the search (C…, G…, or D…)",
							"x-ui": {
								"widget": "remote-select",
								"remote_select_options_path": "/v1/agents/{agent_id}/connectors/{connector_id}/channels",
								"remote_select_id_key": "id",
								"remote_select_label_key": "display_label",
								"remote_select_fallback_placeholder": "Channel ID (e.g. C01234567)",
								"help_text": "Optional — limit search to this channel; shown by name in the approval summary when set"
							}
						},
						"count": {
							"type": "integer",
							"default": 20,
							"description": "Max results per page (1-100)",
							"x-ui": {"label": "Max results", "help_text": "Maximum number of results per page (1–100, default 20)"}
						},
						"page": {
							"type": "integer",
							"default": 1,
							"description": "Page number for pagination",
							"x-ui": {"hidden": true}
						},
						"sort": {
							"type": "string",
							"description": "Sort order: score (relevance) or timestamp",
							"x-ui": {"label": "Sort by", "help_text": "Sort results by relevance (score) or date (timestamp)"}
						}
					}
				}`)),
			},
			{
				ActionType:      "slack.remove_reaction",
				Name:            "Remove Reaction",
				Description:     "Remove an emoji reaction from a message (reactions.remove).",
				RiskLevel:       "low",
				DisplayTemplate: "Remove reaction in {{channel_name}}",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["channel", "name"],
					"properties": {
						"channel": {
							"type": "string",
							"description": "Channel ID containing the message (e.g. C01234567)",
							"x-ui": {
								"widget": "remote-select",
								"remote_select_options_path": "/v1/agents/{agent_id}/connectors/{connector_id}/channels",
								"remote_select_id_key": "id",
								"remote_select_label_key": "display_label",
								"remote_select_fallback_placeholder": "Channel ID (e.g. C01234567)",
								"help_text": "Choose a channel or enter an ID."
							}
						},
						"timestamp": {
							"type": "string",
							"description": "Timestamp of the message (e.g. 1234567890.123456)",
							"x-ui": {"hidden": true}
						},
						"name": {
							"type": "string",
							"description": "Emoji name without colons (e.g. thumbsup)",
							"x-ui": {"label": "Emoji", "placeholder": "thumbsup"}
						}
					}
				}`)),
			},
			{
				ActionType:      "slack.archive_channel",
				Name:            "Archive Channel",
				Description:     "Archive a Slack channel (conversations.archive).",
				RiskLevel:       "high",
				DisplayTemplate: "Archive {{channel_name}}",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["channel"],
					"properties": {
						"channel": {
							"type": "string",
							"description": "Channel ID to archive",
							"x-ui": {
								"widget": "remote-select",
								"remote_select_options_path": "/v1/agents/{agent_id}/connectors/{connector_id}/channels",
								"remote_select_id_key": "id",
								"remote_select_label_key": "display_label",
								"remote_select_fallback_placeholder": "Channel ID (e.g. C01234567)"
							}
						}
					}
				}`)),
			},
			{
				ActionType:      "slack.rename_channel",
				OperationType:   "edit",
				Name:            "Rename Channel",
				Description:     "Rename a Slack channel (conversations.rename).",
				RiskLevel:       "medium",
				DisplayTemplate: "Rename {{channel_name}}",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["channel", "name"],
					"properties": {
						"channel": {
							"type": "string",
							"description": "Channel ID to rename",
							"x-ui": {
								"widget": "remote-select",
								"remote_select_options_path": "/v1/agents/{agent_id}/connectors/{connector_id}/channels",
								"remote_select_id_key": "id",
								"remote_select_label_key": "display_label",
								"remote_select_fallback_placeholder": "Channel ID (e.g. C01234567)"
							}
						},
						"name": {
							"type": "string",
							"description": "New channel name (lowercase, no spaces)",
							"x-ui": {"placeholder": "new-channel-name"}
						}
					}
				}`)),
			},
			{
				ActionType:      "slack.remove_from_channel",
				Name:            "Remove from Channel",
				Description:     "Remove a user from a channel (conversations.kick).",
				RiskLevel:       "medium",
				DisplayTemplate: "Remove user from {{channel_name}}",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["channel", "user"],
					"properties": {
						"channel": {
							"type": "string",
							"description": "Channel ID",
							"x-ui": {
								"widget": "remote-select",
								"remote_select_options_path": "/v1/agents/{agent_id}/connectors/{connector_id}/channels",
								"remote_select_id_key": "id",
								"remote_select_label_key": "display_label",
								"remote_select_fallback_placeholder": "Channel ID (e.g. C01234567)"
							}
						},
						"user": {
							"type": "string",
							"description": "User ID to remove (e.g. U01234567)",
							"x-ui": {
								"widget": "remote-select",
								"label": "User",
								"remote_select_options_path": "/v1/agents/{agent_id}/connectors/{connector_id}/users",
								"remote_select_id_key": "id",
								"remote_select_label_key": "display_label",
								"remote_select_fallback_placeholder": "User ID (e.g. U01234567)"
							}
						}
					}
				}`)),
			},
			{
				ActionType:      "slack.get_user_profile",
				Name:            "Get User Profile",
				Description:     "Fetch a workspace user's profile (users.info).",
				RiskLevel:       "low",
				DisplayTemplate: "Get Slack profile for user",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["user_id"],
					"properties": {
						"user_id": {
							"type": "string",
							"description": "Slack user ID (e.g. U01234567)",
							"x-ui": {
								"widget": "remote-select",
								"label": "User",
								"remote_select_options_path": "/v1/agents/{agent_id}/connectors/{connector_id}/users",
								"remote_select_id_key": "id",
								"remote_select_label_key": "display_label",
								"remote_select_fallback_placeholder": "User ID (e.g. U01234567)"
							}
						}
					}
				}`)),
			},
			{
				ActionType:      "slack.pin_message",
				Name:            "Pin Message",
				Description:     "Pin a message in a channel (pins.add).",
				RiskLevel:       "low",
				DisplayTemplate: "Pin message in {{channel_name}}",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["channel", "ts"],
					"properties": {
						"channel": {
							"type": "string",
							"description": "Channel ID",
							"x-ui": {
								"widget": "remote-select",
								"remote_select_options_path": "/v1/agents/{agent_id}/connectors/{connector_id}/channels",
								"remote_select_id_key": "id",
								"remote_select_label_key": "display_label",
								"remote_select_fallback_placeholder": "Channel ID (e.g. C01234567)"
							}
						},
						"ts": {
							"type": "string",
							"description": "Message timestamp to pin",
							"x-ui": {"hidden": true}
						}
					}
				}`)),
			},
			{
				ActionType:      "slack.unpin_message",
				Name:            "Unpin Message",
				Description:     "Unpin a message in a channel (pins.remove).",
				RiskLevel:       "medium",
				DisplayTemplate: "Unpin message in {{channel_name}}",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["channel", "ts"],
					"properties": {
						"channel": {
							"type": "string",
							"description": "Channel ID",
							"x-ui": {
								"widget": "remote-select",
								"remote_select_options_path": "/v1/agents/{agent_id}/connectors/{connector_id}/channels",
								"remote_select_id_key": "id",
								"remote_select_label_key": "display_label",
								"remote_select_fallback_placeholder": "Channel ID (e.g. C01234567)"
							}
						},
						"ts": {
							"type": "string",
							"description": "Message timestamp to unpin",
							"x-ui": {"hidden": true}
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
				OAuthScopes:   OAuthScopes,
			},
		},
		Templates: []connectors.ManifestTemplate{
			{
				ID:               "tpl_slack_send_to_channel",
				ActionType:       "slack.send_message",
				Name:             "Post to a channel",
				Description:      "Locks the channel and lets the agent choose the message content as the authorizing user.",
				Parameters:       json.RawMessage(`{"channel":"#general","message":"*"}`),
				StandingApproval: neverExpire,
			},
			{
				ID:               "tpl_slack_send_any",
				ActionType:       "slack.send_message",
				Name:             "Send messages freely",
				Description:      "Agent can send any message to any channel as the authorizing user.",
				Parameters:       json.RawMessage(`{"channel":"*","message":"*"}`),
				StandingApproval: neverExpire,
			},
			{
				ID:               "tpl_slack_create_channel",
				ActionType:       "slack.create_channel",
				Name:             "Create channels",
				Description:      "Agent can create public channels with any name.",
				Parameters:       json.RawMessage(`{"name":"*","is_private":false}`),
				StandingApproval: neverExpire,
			},
			{
				ID:               "tpl_slack_list_channels",
				ActionType:       "slack.list_channels",
				Name:             "List channels",
				Description:      "Agent can list channels, including the authorizing user's DMs when profile email matches Slack.",
				Parameters:       json.RawMessage(`{"types":"*","limit":"*","cursor":"*"}`),
				StandingApproval: neverExpire,
			},
			{
				ID:               "tpl_slack_read_channel",
				ActionType:       "slack.read_channel_messages",
				Name:             "Read channel messages",
				Description:      "Agent can read messages from channels, DMs, or group DMs.",
				Parameters:       json.RawMessage(`{"channel":"*","limit":"*","oldest":"*","latest":"*","cursor":"*"}`),
				StandingApproval: neverExpire,
			},
			{
				ID:               "tpl_slack_read_thread",
				ActionType:       "slack.read_thread",
				Name:             "Read thread replies",
				Description:      "Agent can read thread replies.",
				Parameters:       json.RawMessage(`{"channel":"*","thread_ts":"*","limit":"*","cursor":"*"}`),
				StandingApproval: neverExpire,
			},
			{
				ID:               "tpl_slack_schedule_message",
				ActionType:       "slack.schedule_message",
				Name:             "Schedule messages",
				Description:      "Agent can schedule messages as the authorizing user.",
				Parameters:       json.RawMessage(`{"channel":"*","message":"*","post_at":"*"}`),
				StandingApproval: neverExpire,
			},
			{
				ID:               "tpl_slack_set_topic",
				ActionType:       "slack.set_topic",
				Name:             "Set channel topics",
				Description:      "Agent can update the topic on any channel.",
				Parameters:       json.RawMessage(`{"channel":"*","topic":"*"}`),
				StandingApproval: neverExpire,
			},
			{
				ID:               "tpl_slack_invite_to_channel",
				ActionType:       "slack.invite_to_channel",
				Name:             "Invite users to channels",
				Description:      "Agent can invite users to any channel.",
				Parameters:       json.RawMessage(`{"channel":"*","users":"*"}`),
				StandingApproval: neverExpire,
			},
			{
				ID:               "tpl_slack_upload_file",
				ActionType:       "slack.upload_file",
				Name:             "Upload files",
				Description:      "Agent can upload files to any channel.",
				Parameters:       json.RawMessage(`{"channel":"*","filename":"*","content":"*","title":"*"}`),
				StandingApproval: neverExpire,
			},
			{
				ID:               "tpl_slack_add_reaction",
				ActionType:       "slack.add_reaction",
				Name:             "Add reactions",
				Description:      "Agent can add emoji reactions to messages.",
				Parameters:       json.RawMessage(`{"channel":"*","timestamp":"*","name":"*"}`),
				StandingApproval: neverExpire,
			},
			{
				ID:               "tpl_slack_send_dm",
				ActionType:       "slack.send_dm",
				Name:             "Send direct messages",
				Description:      "Agent can send DMs as the authorizing user.",
				Parameters:       json.RawMessage(`{"user_id":"*","message":"*"}`),
				StandingApproval: neverExpire,
			},
			{
				ID:               "tpl_slack_update_message",
				ActionType:       "slack.update_message",
				Name:             "Update messages",
				Description:      "Agent can edit messages the authorizing user is allowed to change.",
				Parameters:       json.RawMessage(`{"channel":"*","ts":"*","message":"*"}`),
				StandingApproval: neverExpire,
			},
			{
				ID:               "tpl_slack_delete_message",
				ActionType:       "slack.delete_message",
				Name:             "Delete messages",
				Description:      "Agent can delete messages the authorizing user is allowed to remove.",
				Parameters:       json.RawMessage(`{"channel":"*","ts":"*"}`),
				StandingApproval: neverExpire,
			},
			{
				ID:               "tpl_slack_list_users",
				ActionType:       "slack.list_users",
				Name:             "List users",
				Description:      "Agent can list workspace users.",
				Parameters:       json.RawMessage(`{"limit":"*","cursor":"*"}`),
				StandingApproval: neverExpire,
			},
			{
				ID:               "tpl_slack_search_messages",
				ActionType:       "slack.search_messages",
				Name:             "Search messages",
				Description:      "Agent can search messages across channels.",
				Parameters:       json.RawMessage(`{"query":"*","channel":"*","count":"*","page":"*","sort":"*"}`),
				StandingApproval: neverExpire,
			},
			{
				ID:               "tpl_slack_read_channel_scoped",
				ActionType:       "slack.read_channel_messages",
				Name:             "Read one channel only",
				Description:      "Locks channel ID — agent can only read messages from that channel.",
				Parameters:       json.RawMessage(`{"channel":"C01234567","limit":"*","oldest":"*","latest":"*","cursor":"*"}`),
				StandingApproval: neverExpire,
			},
			{
				ID:               "tpl_slack_send_dm_scoped",
				ActionType:       "slack.send_dm",
				Name:             "DM one user only",
				Description:      "Locks recipient user ID — agent can only DM that user.",
				Parameters:       json.RawMessage(`{"user_id":"U01234567","message":"*"}`),
				StandingApproval: neverExpire,
			},
			{
				ID:               "tpl_slack_create_channel_private",
				ActionType:       "slack.create_channel",
				Name:             "Create private channels only",
				Description:      "Agent can create channels but they must be private.",
				Parameters:       json.RawMessage(`{"name":"*","is_private":true}`),
				StandingApproval: neverExpire,
			},
			{
				ID:               "tpl_slack_remove_reaction",
				ActionType:       "slack.remove_reaction",
				Name:             "Remove reactions",
				Description:      "Agent can remove emoji reactions from messages.",
				Parameters:       json.RawMessage(`{"channel":"*","timestamp":"*","name":"*"}`),
				StandingApproval: neverExpire,
			},
			{
				ID:               "tpl_slack_archive_channel",
				ActionType:       "slack.archive_channel",
				Name:             "Archive channels",
				Description:      "Agent can archive Slack channels.",
				Parameters:       json.RawMessage(`{"channel":"*"}`),
				StandingApproval: neverExpire,
			},
			{
				ID:               "tpl_slack_rename_channel",
				ActionType:       "slack.rename_channel",
				Name:             "Rename channels",
				Description:      "Agent can rename channels.",
				Parameters:       json.RawMessage(`{"channel":"*","name":"*"}`),
				StandingApproval: neverExpire,
			},
			{
				ID:               "tpl_slack_remove_from_channel",
				ActionType:       "slack.remove_from_channel",
				Name:             "Remove users from channels",
				Description:      "Agent can remove users from channels.",
				Parameters:       json.RawMessage(`{"channel":"*","user":"*"}`),
				StandingApproval: neverExpire,
			},
			{
				ID:               "tpl_slack_get_user_profile",
				ActionType:       "slack.get_user_profile",
				Name:             "Read user profiles",
				Description:      "Agent can fetch Slack user profiles by user ID.",
				Parameters:       json.RawMessage(`{"user_id":"*"}`),
				StandingApproval: neverExpire,
			},
			{
				ID:               "tpl_slack_pin_message",
				ActionType:       "slack.pin_message",
				Name:             "Pin messages",
				Description:      "Agent can pin messages in channels.",
				Parameters:       json.RawMessage(`{"channel":"*","ts":"*"}`),
				StandingApproval: neverExpire,
			},
			{
				ID:               "tpl_slack_unpin_message",
				ActionType:       "slack.unpin_message",
				Name:             "Unpin messages",
				Description:      "Agent can unpin messages in channels.",
				Parameters:       json.RawMessage(`{"channel":"*","ts":"*"}`),
				StandingApproval: neverExpire,
			},
		},
	}
}
