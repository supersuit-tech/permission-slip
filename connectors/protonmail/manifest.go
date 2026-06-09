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
		LogoSVG:     logoSVG,
		Actions: []connectors.ManifestAction{
			{
				ActionType:  "protonmail.send_email",
				Name:        "Send Email",
				Description: "Send an email via SMTP through the local Proton IMAP/SMTP proxy",
				RiskLevel:   "high",
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
				ActionType:  "protonmail.read_inbox",
				Name:        "Read Inbox",
				Description: "Fetch recent emails from a mailbox folder via IMAP",
				RiskLevel:   "low",
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
						}
					}
				}`)),
			},
			{
				ActionType:  "protonmail.search_emails",
				Name:        "Search Emails",
				Description: "Search emails by subject, sender, or date range via IMAP",
				RiskLevel:   "low",
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
						}
					}
				}`)),
			},
			{
				ActionType:  "protonmail.read_email",
				Name:        "Read Email",
				Description: "Fetch a specific email by stable IMAP UID with full body",
				RiskLevel:   "low",
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
				ActionType:  "protonmail.archive_email",
				Name:        "Archive Email",
				Description: "Move one or more emails to the Archive folder via IMAP MOVE",
				RiskLevel:   "medium",
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
					},
					{
						Key:         "password",
						Label:       "Bridge password",
						Placeholder: "From Bridge info (not your Proton account password)",
						Secret:      ptrBool(true),
						Required:    ptrBool(true),
						HelpText:    "Run protonmail-bridge info to get the Bridge-generated password. This is not your Proton account login password.",
					},
					{
						Key:         "imap_host",
						Label:       "IMAP host",
						Placeholder: "127.0.0.1",
						Secret:      ptrBool(false),
						Required:    ptrBool(false),
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
