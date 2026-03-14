// Package protonmail implements the Proton Mail connector for the Permission
// Slip connector execution layer. It uses IMAP/SMTP to communicate with
// Proton Mail Bridge, which must be running locally.
//
// Actions: send_email (SMTP), read_inbox, search_emails, read_email, archive_email (IMAP).
// Registration is gated behind the ENABLE_PROTONMAIL_CONNECTOR env var.
package protonmail

import (
	_ "embed"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

const (
	defaultTimeout = 30 * time.Second

	credKeyUsername = "username"
	credKeyPassword = "password"
	credKeySMTPHost = "smtp_host"
	credKeySMTPPort = "smtp_port"
	credKeyIMAPHost = "imap_host"
	credKeyIMAPPort = "imap_port"

	defaultSMTPHost = "127.0.0.1"
	defaultSMTPPort = "1025"
	defaultIMAPHost = "127.0.0.1"
	defaultIMAPPort = "1143"
)

// ProtonMailConnector owns the shared configuration for all Proton Mail actions.
type ProtonMailConnector struct {
	timeout time.Duration
}

// New creates a ProtonMailConnector with sensible defaults.
func New() *ProtonMailConnector {
	return &ProtonMailConnector{
		timeout: defaultTimeout,
	}
}

// ID returns "protonmail", matching the connectors.id in the database.
func (c *ProtonMailConnector) ID() string { return "protonmail" }

//go:embed logo.svg
var logoSVG string

// Manifest returns the connector's metadata manifest.
func (c *ProtonMailConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "protonmail",
		Name:        "Proton Mail",
		Description: "Send and read emails through Proton Mail via IMAP/SMTP Bridge. Requires Proton Mail Bridge running locally.",
		LogoSVG:     logoSVG,
		Actions: []connectors.ManifestAction{
			{
				ActionType:  "protonmail.send_email",
				Name:        "Send Email",
				Description: "Send an email via SMTP through Proton Mail Bridge",
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
				Description: "Fetch a specific email by sequence number with full body",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["message_id"],
					"properties": {
						"message_id": {
							"type": "integer",
							"minimum": 1,
							"description": "The sequence number of the email to read"
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
							"description": "Sequence number of a single email to archive (shorthand for message_ids with one item)"
						},
						"message_ids": {
							"type": "array",
							"items": {"type": "integer", "minimum": 1},
							"minItems": 1,
							"maxItems": 50,
							"description": "Sequence numbers of emails to archive (batch). Combined unique count of message_id + message_ids must not exceed 50."
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
				InstructionsURL: "https://proton.me/support/bridge",
			},
		},
		Templates: []connectors.ManifestTemplate{
			{
				ID:          "tpl_protonmail_send",
				ActionType:  "protonmail.send_email",
				Name:        "Send emails from your Proton Mail account",
				Description: "Agent can send emails on your behalf via Proton Mail Bridge.",
				Parameters:  json.RawMessage(`{"to":[],"subject":"","body":""}`),
			},
			{
				ID:          "tpl_protonmail_read_inbox",
				ActionType:  "protonmail.read_inbox",
				Name:        "Read recent inbox emails",
				Description: "Agent can read your recent inbox emails.",
				Parameters:  json.RawMessage(`{"folder":"INBOX","limit":10}`),
			},
			{
				ID:          "tpl_protonmail_search",
				ActionType:  "protonmail.search_emails",
				Name:        "Search emails",
				Description: "Agent can search your emails by subject, sender, or date.",
				Parameters:  json.RawMessage(`{"folder":"INBOX","limit":10}`),
			},
			{
				ID:          "tpl_protonmail_read_email",
				ActionType:  "protonmail.read_email",
				Name:        "Read a specific email",
				Description: "Agent can read the full content of a specific email.",
				Parameters:  json.RawMessage(`{"folder":"INBOX"}`),
			},
			{
				ID:          "tpl_protonmail_archive",
				ActionType:  "protonmail.archive_email",
				Name:        "Archive emails",
				Description: "Agent can move emails to the Archive folder.",
				Parameters:  json.RawMessage(`{"folder":"INBOX"}`),
			},
		},
	}
}

// Actions returns the registered action handlers keyed by action_type.
func (c *ProtonMailConnector) Actions() map[string]connectors.Action {
	return map[string]connectors.Action{
		"protonmail.send_email":    &sendEmailAction{conn: c},
		"protonmail.read_inbox":    &readInboxAction{conn: c},
		"protonmail.search_emails": &searchEmailsAction{conn: c},
		"protonmail.read_email":    &readEmailAction{conn: c},
		"protonmail.archive_email": &archiveEmailAction{conn: c},
	}
}

// ValidateCredentials checks that the provided credentials contain the
// required fields for connecting to Proton Mail Bridge.
func (c *ProtonMailConnector) ValidateCredentials(_ context.Context, creds connectors.Credentials) error {
	username, ok := creds.Get(credKeyUsername)
	if !ok || username == "" {
		return &connectors.ValidationError{Message: "missing required credential: username"}
	}
	password, ok := creds.Get(credKeyPassword)
	if !ok || password == "" {
		return &connectors.ValidationError{Message: "missing required credential: password"}
	}

	// Validate optional port fields are numeric if provided.
	for _, key := range []string{credKeySMTPPort, credKeyIMAPPort} {
		if v, exists := creds.Get(key); exists && v != "" {
			if _, err := strconv.Atoi(v); err != nil {
				return &connectors.ValidationError{Message: fmt.Sprintf("invalid %s: must be a numeric port value", key)}
			}
		}
	}

	return nil
}

// smtpConfig extracts SMTP connection settings from credentials, using defaults
// for Proton Mail Bridge when not specified.
func smtpConfig(creds connectors.Credentials) (host, port, username, password string) {
	host, _ = creds.Get(credKeySMTPHost)
	if host == "" {
		host = defaultSMTPHost
	}
	port, _ = creds.Get(credKeySMTPPort)
	if port == "" {
		port = defaultSMTPPort
	}
	username, _ = creds.Get(credKeyUsername)
	password, _ = creds.Get(credKeyPassword)
	return host, port, username, password
}

// imapConfig extracts IMAP connection settings from credentials, using defaults
// for Proton Mail Bridge when not specified.
func imapConfig(creds connectors.Credentials) (host, port, username, password string) {
	host, _ = creds.Get(credKeyIMAPHost)
	if host == "" {
		host = defaultIMAPHost
	}
	port, _ = creds.Get(credKeyIMAPPort)
	if port == "" {
		port = defaultIMAPPort
	}
	username, _ = creds.Get(credKeyUsername)
	password, _ = creds.Get(credKeyPassword)
	return host, port, username, password
}
