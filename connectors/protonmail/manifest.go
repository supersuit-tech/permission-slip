package protonmail

import (
	_ "embed"
	"encoding/json"

	"github.com/supersuit-tech/permission-slip/connectors"
)

//go:embed logo.svg
var logoSVG string

func (c *ProtonMailConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "protonmail",
		Name:        "Proton Mail",
		Description: "Send and read emails through Proton Mail via a local IMAP/SMTP proxy (Proton Mail Bridge on x86_64, or hydroxide on ARM/Raspberry Pi). The proxy must be running on the same host as Permission Slip.",
		Status:      "early_preview",
		LogoSVG:     logoSVG,
		Actions: []connectors.ManifestAction{
			{
				ActionType:      "protonmail.send_email",
				Name:            "Send Email",
				Description:     "Send an email via SMTP through the local Proton IMAP/SMTP proxy",
				RiskLevel:       "high",
				DisplayTemplate: "Send email to {{to}} — {{subject}}",
				Preview: &connectors.ActionPreview{
					Layout: "message",
					Fields: map[string]string{"to": "to", "subject": "subject", "body": "body"},
				},
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["to", "subject", "body"],
					"properties": {
						"to": {
							"type": "array",
							"items": {"type": "string", "format": "email"},
							"minItems": 1,
							"description": "Recipient email addresses"
						},
						"cc": {
							"type": "array",
							"items": {"type": "string", "format": "email"},
							"description": "CC recipient email addresses"
						},
						"bcc": {
							"type": "array",
							"items": {"type": "string", "format": "email"},
							"description": "BCC recipient email addresses"
						},
						"subject": {
							"type": "string",
							"description": "Email subject line"
						},
						"body": {
							"type": "string",
							"description": "Email body content"
						},
						"content_type": {
							"type": "string",
							"enum": ["text/plain", "text/html"],
							"default": "text/plain",
							"description": "Content type of the email body"
						},
						"reply_to": {
							"type": "string",
							"format": "email",
							"description": "Reply-To email address"
						}
					}
				}`)),
			},
			{
				ActionType:      "protonmail.reply_email",
				Name:            "Reply to Email",
				Description:     "Reply to an existing email with correct In-Reply-To threading via SMTP",
				RiskLevel:       "medium",
				DisplayTemplate: "Reply to email in {{folder}}",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["in_reply_to_message_id", "body"],
					"properties": {
						"in_reply_to_message_id": {
							"type": "integer",
							"minimum": 1,
							"description": "Stable IMAP UID of the email being replied to (from read_inbox or search_emails results)"
						},
						"folder": {
							"type": "string",
							"default": "INBOX",
							"description": "Mailbox folder containing the source email"
						},
						"to": {
							"type": "array",
							"items": {"type": "string", "format": "email"},
							"description": "Reply recipients. Defaults to the source email sender when omitted."
						},
						"cc": {
							"type": "array",
							"items": {"type": "string", "format": "email"},
							"description": "CC recipient email addresses"
						},
						"bcc": {
							"type": "array",
							"items": {"type": "string", "format": "email"},
							"description": "BCC recipient email addresses"
						},
						"subject": {
							"type": "string",
							"description": "Reply subject. Defaults to Re: plus the source subject when omitted."
						},
						"body": {
							"type": "string",
							"description": "Reply body content"
						},
						"content_type": {
							"type": "string",
							"enum": ["text/plain", "text/html"],
							"default": "text/plain",
							"description": "Content type of the reply body"
						}
					}
				}`)),
			},
			{
				ActionType:      "protonmail.read_inbox",
				Name:            "Read Inbox",
				Description:     "Fetch recent emails from a mailbox folder via IMAP",
				RiskLevel:       "low",
				DisplayTemplate: "Read {{limit:count}} most recent in {{folder}}",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"properties": {
						"folder": {
							"type": "string",
							"default": "INBOX",
							"description": "Mailbox folder to read from"
						},
						"limit": {
							"type": "integer",
							"minimum": 1,
							"maximum": 50,
							"default": 10,
							"description": "Maximum number of emails to fetch"
						},
						"unread_only": {
							"type": "boolean",
							"default": false,
							"description": "Only fetch unread emails"
						},
						"group_by_thread": {
							"type": "boolean",
							"default": true,
							"description": "Collapse results to one entry per conversation: the latest message, with thread_size and thread_uids covering the fetched window (a long thread may be partially represented when it exceeds the limit). Set to false for a flat per-email listing. To act on a whole conversation (archive, delete, move), pass its thread_uids as the message_ids of the batch action."
						}
					}
				}`)),
			},
			{
				ActionType:      "protonmail.search_emails",
				Name:            "Search Emails",
				Description:     "Search emails by subject, sender, or date range via IMAP",
				RiskLevel:       "low",
				DisplayTemplate: "Search {{folder}} for emails",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"properties": {
						"folder": {
							"type": "string",
							"default": "INBOX",
							"description": "Mailbox folder to search in"
						},
						"subject": {
							"type": "string",
							"description": "Search by subject (substring match)"
						},
						"from": {
							"type": "string",
							"description": "Search by sender email address"
						},
						"since": {
							"type": "string",
							"format": "date",
							"description": "Search for emails on or after this date (YYYY-MM-DD)"
						},
						"before": {
							"type": "string",
							"format": "date",
							"description": "Search for emails before this date (YYYY-MM-DD)"
						},
						"limit": {
							"type": "integer",
							"minimum": 1,
							"maximum": 50,
							"default": 10,
							"description": "Maximum number of results to return"
						},
						"group_by_thread": {
							"type": "boolean",
							"default": true,
							"description": "Collapse results to one entry per conversation: the latest message, with thread_size and thread_uids covering the fetched window (a long thread may be partially represented when it exceeds the limit). Set to false for a flat per-email listing. To act on a whole conversation (archive, delete, move), pass its thread_uids as the message_ids of the batch action."
						}
					}
				}`)),
			},
			{
				ActionType:      "protonmail.read_email",
				Name:            "Read Email",
				Description:     "Fetch a specific email by stable IMAP UID with full body",
				RiskLevel:       "low",
				DisplayTemplate: "Read email in {{folder}}",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["message_id"],
					"properties": {
						"message_id": {
							"type": "integer",
							"minimum": 1,
							"description": "Stable IMAP UID of the email within the folder (from read_inbox or search_emails results)"
						},
						"folder": {
							"type": "string",
							"default": "INBOX",
							"description": "Mailbox folder containing the email"
						}
					}
				}`)),
			},
			{
				ActionType:      "protonmail.archive_email",
				Name:            "Archive Email",
				Description:     "Move one or more emails to the Archive folder via IMAP MOVE",
				RiskLevel:       "medium",
				DisplayTemplate: "Archive email in {{folder}}",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"anyOf": [
						{"required": ["message_id"]},
						{"required": ["message_ids"]}
					],
					"properties": {
						"message_id": {
							"type": "integer",
							"minimum": 1,
							"description": "Stable IMAP UID of a single email to archive (shorthand for message_ids with one item)"
						},
						"message_ids": {
							"type": "array",
							"items": {"type": "integer", "minimum": 1},
							"minItems": 1,
							"maxItems": 50,
							"description": "Stable IMAP UIDs of emails to archive (batch). Combined unique count of message_id + message_ids must not exceed 50."
						},
						"folder": {
							"type": "string",
							"default": "INBOX",
							"description": "Source mailbox folder containing the emails"
						}
					}
				}`)),
			},
			{
				ActionType:      "protonmail.list_folders",
				Name:            "List Folders",
				Description:     "List mailbox folders available via IMAP",
				RiskLevel:       "low",
				DisplayTemplate: "List mailbox folders",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"properties": {}
				}`)),
			},
			{
				ActionType:       "protonmail.mark_read",
				Name:             "Mark Read",
				Description:      "Mark one or more emails as read via IMAP STORE \\Seen",
				RiskLevel:        "low",
				DisplayTemplate:  "Mark email as read",
				ParametersSchema: uidMessageParametersSchema(),
			},
			{
				ActionType:       "protonmail.mark_unread",
				Name:             "Mark Unread",
				Description:      "Mark one or more emails as unread via IMAP STORE \\Seen removal",
				RiskLevel:        "low",
				DisplayTemplate:  "Mark email as unread",
				ParametersSchema: uidMessageParametersSchema(),
			},
			{
				ActionType:       "protonmail.flag",
				Name:             "Flag Email",
				Description:      "Flag one or more emails via IMAP STORE \\Flagged",
				RiskLevel:        "low",
				DisplayTemplate:  "Flag email",
				ParametersSchema: uidMessageParametersSchema(),
			},
			{
				ActionType:       "protonmail.unflag",
				Name:             "Unflag Email",
				Description:      "Remove the flag from one or more emails via IMAP STORE \\Flagged removal",
				RiskLevel:        "low",
				DisplayTemplate:  "Unflag email",
				ParametersSchema: uidMessageParametersSchema(),
			},
			{
				ActionType:       "protonmail.move_to_folder",
				Name:             "Move to Folder",
				Description:      "Move one or more emails to another mailbox folder via IMAP MOVE",
				RiskLevel:        "medium",
				DisplayTemplate:  "Move email to {{target_folder}}",
				ParametersSchema: moveToFolderParametersSchema(),
			},
			{
				ActionType:       "protonmail.delete",
				Name:             "Delete Email",
				Description:      "Move one or more emails to the Trash folder via IMAP MOVE",
				RiskLevel:        "high",
				DisplayTemplate:  "Delete email",
				ParametersSchema: uidMessageParametersSchema(),
			},
		},

		RequiredCredentials: []connectors.ManifestCredential{
			{
				Service:         "protonmail",
				AuthType:        "custom",
				InstructionsURL: "https://github.com/supersuit-tech/permission-slip/blob/main/docs/connectors/protonmail.md",
				Fields: []connectors.ManifestCredentialField{
					{
						Key:         "username",
						Label:       "Bridge username",
						Placeholder: "Your Proton address as shown by Bridge info",
						Secret:      ptrBool(false),
						Required:    ptrBool(true),
						HelpText:    "Run protonmail-bridge --cli, then type info. Use the username Bridge prints — it must match the running Bridge instance.",
					},
					{
						Key:         "password",
						Label:       "Bridge password",
						Placeholder: "From Bridge info (not your Proton account password)",
						Secret:      ptrBool(true),
						Required:    ptrBool(true),
						HelpText:    "From protonmail-bridge info — the Bridge-generated password, not your Proton account login password.",
					},
					{
						Key:         "imap_host",
						Label:       "IMAP host",
						Placeholder: "127.0.0.1",
						Secret:      ptrBool(false),
						Required:    ptrBool(false),
						HelpText:    "Leave blank when Bridge runs on the same host (default 127.0.0.1:1143).",
					},
					{
						Key:         "imap_port",
						Label:       "IMAP port",
						Placeholder: "1143",
						Secret:      ptrBool(false),
						Required:    ptrBool(false),
					},
					{
						Key:         "smtp_host",
						Label:       "SMTP host",
						Placeholder: "127.0.0.1",
						Secret:      ptrBool(false),
						Required:    ptrBool(false),
						HelpText:    "Leave blank when Bridge runs on the same host (default 127.0.0.1:1025).",
					},
					{
						Key:         "smtp_port",
						Label:       "SMTP port",
						Placeholder: "1025",
						Secret:      ptrBool(false),
						Required:    ptrBool(false),
					},
				},
			},
		},
		Templates: protonmailTemplates(),
	}
}

func ptrBool(b bool) *bool { return &b }
