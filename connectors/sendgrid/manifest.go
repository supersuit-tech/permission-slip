package sendgrid

import (
	_ "embed"
	"encoding/json"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// Manifest returns the connector's metadata manifest. Used by the server to
// auto-seed DB rows on startup.
//
//go:embed logo.svg
var logoSVG string

func (c *SendGridConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "sendgrid",
		Name:        "SendGrid",
		Description: "SendGrid integration for email marketing — campaigns, subscriber lists, templates, and analytics",
		LogoSVG:     logoSVG,
		Actions: []connectors.ManifestAction{
			{
				ActionType:  "sendgrid.send_campaign",
				Name:        "Send Email Campaign",
				Description: "Send a single send email campaign to a list of recipients. WARNING: This immediately sends to all contacts in the specified lists.",
				RiskLevel:   "high",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["name", "subject", "list_ids", "sender_id"],
					"properties": {
						"name": {
							"type": "string",
							"maxLength": 100,
							"description": "Internal name for this campaign (e.g. 'March 2026 Newsletter')",
							"x-ui": {"label": "Campaign name", "placeholder": "March 2026 Newsletter"}
						},
						"subject": {
							"type": "string",
							"maxLength": 998,
							"description": "Email subject line seen by recipients",
							"x-ui": {"label": "Subject", "placeholder": "Your Monthly Newsletter"}
						},
						"html_content": {
							"type": "string",
							"description": "HTML body of the email. At least one of html_content or plain_content is required.",
							"x-ui": {"label": "HTML content", "widget": "textarea"}
						},
						"plain_content": {
							"type": "string",
							"description": "Plain text body of the email. At least one of html_content or plain_content is required.",
							"x-ui": {"label": "Plain text content", "widget": "textarea"}
						},
						"list_ids": {
							"type": "array",
							"items": {"type": "string"},
							"minItems": 1,
							"description": "Contact list IDs to send to. Use sendgrid.list_lists to find available list IDs.",
							"x-ui": {"label": "List IDs", "help_text": "SendGrid contact list IDs — find in Marketing > Lists"}
						},
						"sender_id": {
							"type": "integer",
							"description": "Verified sender identity ID. Use sendgrid.list_senders to find your sender ID.",
							"x-ui": {"label": "Sender", "help_text": "Verified sender ID — find in Settings > Sender Authentication"}
						},
						"suppression_group_id": {
							"type": "integer",
							"description": "Unsubscribe group ID for managing opt-outs (optional but recommended)",
							"x-ui": {"label": "Unsubscribe group", "help_text": "Suppression group ID — find in Suppressions > Unsubscribe Groups"}
						}
					}
				}`)),
			},
			{
				ActionType:  "sendgrid.schedule_campaign",
				Name:        "Schedule Email Campaign",
				Description: "Schedule a single send email campaign for future delivery. The campaign can be cancelled before the scheduled time.",
				RiskLevel:   "high",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["name", "subject", "list_ids", "sender_id", "send_at"],
					"properties": {
						"name": {
							"type": "string",
							"maxLength": 100,
							"description": "Internal name for this campaign (e.g. 'April Product Launch')",
							"x-ui": {"label": "Campaign name", "placeholder": "April Product Launch"}
						},
						"subject": {
							"type": "string",
							"maxLength": 998,
							"description": "Email subject line seen by recipients",
							"x-ui": {"label": "Subject", "placeholder": "Your Monthly Newsletter"}
						},
						"html_content": {
							"type": "string",
							"description": "HTML body of the email. At least one of html_content or plain_content is required.",
							"x-ui": {"label": "HTML content", "widget": "textarea"}
						},
						"plain_content": {
							"type": "string",
							"description": "Plain text body of the email. At least one of html_content or plain_content is required.",
							"x-ui": {"label": "Plain text content", "widget": "textarea"}
						},
						"list_ids": {
							"type": "array",
							"items": {"type": "string"},
							"minItems": 1,
							"description": "Contact list IDs to send to. Use sendgrid.list_lists to find available list IDs.",
							"x-ui": {"label": "List IDs", "help_text": "SendGrid contact list IDs — find in Marketing > Lists"}
						},
						"sender_id": {
							"type": "integer",
							"description": "Verified sender identity ID. Use sendgrid.list_senders to find your sender ID.",
							"x-ui": {"label": "Sender", "help_text": "Verified sender ID — find in Settings > Sender Authentication"}
						},
						"send_at": {
							"type": "string",
							"format": "date-time",
							"description": "ISO 8601 timestamp for when to send (must be in the future, e.g. 2026-03-15T10:00:00Z)",
							"x-ui": {"widget": "datetime", "label": "Send at", "help_text": "Scheduled send time in ISO 8601 format"}
						},
						"suppression_group_id": {
							"type": "integer",
							"description": "Unsubscribe group ID for managing opt-outs (optional but recommended)",
							"x-ui": {"label": "Unsubscribe group", "help_text": "Suppression group ID — find in Suppressions > Unsubscribe Groups"}
						}
					}
				}`)),
			},
			{
				ActionType:  "sendgrid.add_to_list",
				Name:        "Add Subscriber to List",
				Description: "Add a contact to a SendGrid contact list. The operation is asynchronous — the returned job_id can be used to track import progress.",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["list_id", "email"],
					"properties": {
						"list_id": {
							"type": "string",
							"description": "Contact list ID. Use sendgrid.list_lists to find available list IDs.",
							"x-ui": {"label": "List ID", "help_text": "SendGrid contact list IDs — find in Marketing > Lists"}
						},
						"email": {
							"type": "string",
							"format": "email",
							"description": "Subscriber email address",
							"x-ui": {"label": "Email", "placeholder": "jane@example.com"}
						},
						"first_name": {
							"type": "string",
							"description": "Subscriber first name (optional)",
							"x-ui": {"label": "First name", "placeholder": "Jane"}
						},
						"last_name": {
							"type": "string",
							"description": "Subscriber last name (optional)",
							"x-ui": {"label": "Last name", "placeholder": "Doe"}
						}
					}
				}`)),
			},
			{
				ActionType:  "sendgrid.remove_from_list",
				Name:        "Remove Subscriber from List",
				Description: "Remove a contact from a SendGrid contact list. This only removes the list association — the contact itself is not deleted.",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["list_id", "contact_id"],
					"properties": {
						"list_id": {
							"type": "string",
							"description": "Contact list ID. Use sendgrid.list_lists to find available list IDs.",
							"x-ui": {"label": "List ID", "help_text": "SendGrid contact list IDs — find in Marketing > Lists"}
						},
						"contact_id": {
							"type": "string",
							"description": "Contact ID to remove from the list",
							"x-ui": {"label": "Contact ID", "help_text": "SendGrid contact identifier"}
						}
					}
				}`)),
			},
			{
				ActionType:  "sendgrid.create_template",
				Name:        "Create Email Template",
				Description: "Create a dynamic transactional email template. After creating, add versions with HTML content via the SendGrid dashboard.",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["name"],
					"properties": {
						"name": {
							"type": "string",
							"maxLength": 100,
							"description": "Template name (e.g. 'Welcome Email', 'Order Confirmation')",
							"x-ui": {"label": "Template name", "placeholder": "Welcome Email"}
						},
						"generation": {
							"type": "string",
							"enum": ["legacy", "dynamic"],
							"description": "Template generation — use 'dynamic' for Handlebars support (default: dynamic)",
							"x-ui": {"label": "Generation", "widget": "select", "help_text": "'dynamic' for modern templates with Handlebars, 'legacy' for older substitution tags"}
						}
					}
				}`)),
			},
			{
				ActionType:  "sendgrid.get_campaign_stats",
				Name:        "Get Campaign Stats",
				Description: "Get analytics for a single send campaign including delivery, open, click, bounce, and unsubscribe metrics",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["singlesend_id"],
					"properties": {
						"singlesend_id": {
							"type": "string",
							"description": "Single send campaign ID (returned by send_campaign or schedule_campaign)",
							"x-ui": {"label": "Campaign ID", "help_text": "Single send campaign ID"}
						}
					}
				}`)),
			},
			{
				ActionType:  "sendgrid.list_segments",
				Name:        "List Segments",
				Description: "List all contact segments in the account with subscriber counts",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"properties": {}
				}`)),
			},
			{
				ActionType:  "sendgrid.list_senders",
				Name:        "List Verified Senders",
				Description: "List all verified sender identities — use this to find sender_id values needed for campaigns",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"properties": {}
				}`)),
			},
			{
				ActionType:  "sendgrid.list_lists",
				Name:        "List Contact Lists",
				Description: "List all contact lists with subscriber counts — use this to find list_id values for campaigns and subscriber management",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"properties": {}
				}`)),
			},
			{
				ActionType:  "sendgrid.send_transactional_email",
				Name:        "Send Transactional Email",
				Description: "Send a single transactional email (welcome, password reset, order confirmation, etc.) via the SendGrid v3 Mail Send API. Supports dynamic templates with Handlebars substitution.",
				RiskLevel:   "high",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["to", "from"],
					"properties": {
						"to": {
							"type": "string",
							"format": "email",
							"description": "Recipient email address",
							"x-ui": {"label": "To", "placeholder": "jane@example.com"}
						},
						"to_name": {
							"type": "string",
							"description": "Recipient display name (optional)",
							"x-ui": {"label": "Recipient name", "placeholder": "Jane Doe"}
						},
						"from": {
							"type": "string",
							"format": "email",
							"description": "Sender email address — must be a verified sender in your SendGrid account",
							"x-ui": {"label": "From", "placeholder": "jane@example.com"}
						},
						"from_name": {
							"type": "string",
							"description": "Sender display name (optional)",
							"x-ui": {"label": "Sender name", "placeholder": "Jane Doe"}
						},
						"subject": {
							"type": "string",
							"maxLength": 998,
							"description": "Email subject line. Required when template_id is not provided.",
							"x-ui": {"label": "Subject", "placeholder": "Your Monthly Newsletter"}
						},
						"html_content": {
							"type": "string",
							"description": "HTML body. Required when template_id is not provided and plain_content is also absent.",
							"x-ui": {"label": "HTML content", "widget": "textarea"}
						},
						"plain_content": {
							"type": "string",
							"description": "Plain-text body. Required when template_id is not provided and html_content is also absent.",
							"x-ui": {"label": "Plain text content", "widget": "textarea"}
						},
						"template_id": {
							"type": "string",
							"description": "SendGrid dynamic template ID (e.g. d-xxxx). When set, html_content/plain_content/subject can be omitted if defined in the template.",
							"x-ui": {"label": "Template ID", "help_text": "Dynamic template ID — find in Email API > Dynamic Templates"}
						},
						"dynamic_template_data": {
							"type": "object",
							"description": "Key/value pairs substituted into the dynamic template via Handlebars (e.g. {\"first_name\": \"Jane\"}). Only used when template_id is set.",
							"additionalProperties": true,
							"x-ui": {"label": "Template data", "help_text": "Handlebars substitution variables as key-value pairs"}
						},
						"reply_to": {
							"type": "string",
							"format": "email",
							"description": "Reply-to email address (optional)",
							"x-ui": {"label": "Reply-to email", "placeholder": "jane@example.com"}
						},
						"cc": {
							"type": "array",
							"items": {"type": "string", "format": "email"},
							"description": "CC recipients (optional). Useful for sending copies to account managers, team inboxes, etc.",
							"x-ui": {"label": "CC", "help_text": "Carbon copy recipients — max 1000 per SendGrid API limits"}
						},
						"bcc": {
							"type": "array",
							"items": {"type": "string", "format": "email"},
							"description": "BCC recipients (optional). Useful for silent compliance copies or audit trails.",
							"x-ui": {"label": "BCC", "help_text": "Blind carbon copy recipients — max 1000 per SendGrid API limits"}
						},
						"categories": {
							"type": "array",
							"items": {"type": "string"},
							"maxItems": 10,
							"description": "Labels for filtering and grouping in the SendGrid Activity Feed and stats (e.g. [\"welcome\", \"onboarding\"]). Maximum 10.",
							"x-ui": {"label": "Categories"}
						}
					}
				}`)),
			},
			{
				ActionType:  "sendgrid.create_contact",
				Name:        "Create Contact",
				Description: "Add or update a contact in SendGrid without assigning them to a specific list. Useful for building a global contact database.",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["email"],
					"properties": {
						"email": {
							"type": "string",
							"format": "email",
							"description": "Contact email address",
							"x-ui": {"label": "Email", "placeholder": "jane@example.com"}
						},
						"first_name": {
							"type": "string",
							"description": "Contact first name (optional)",
							"x-ui": {"label": "First name", "placeholder": "Jane"}
						},
						"last_name": {
							"type": "string",
							"description": "Contact last name (optional)",
							"x-ui": {"label": "Last name", "placeholder": "Doe"}
						},
						"phone_number": {
							"type": "string",
							"description": "Contact phone number (optional)",
							"x-ui": {"label": "Phone number", "placeholder": "+1 (555) 123-4567"}
						},
						"city": {
							"type": "string",
							"description": "Contact city (optional)",
							"x-ui": {"label": "City", "placeholder": "San Francisco"}
						},
						"country": {
							"type": "string",
							"description": "Contact country (optional)",
							"x-ui": {"label": "Country", "placeholder": "United States"}
						}
					}
				}`)),
			},
			{
				ActionType:  "sendgrid.get_bounces",
				Name:        "Get Bounce List",
				Description: "Retrieve the list of email addresses that have bounced, with bounce reason and status. Useful for deliverability management.",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"properties": {
						"start_time": {
							"type": "string",
							"format": "date-time",
							"description": "Filter bounces created after this time (ISO 8601, e.g. 2026-01-01T00:00:00Z)",
							"x-ui": {"widget": "datetime", "label": "Start time", "help_text": "Only include bounces after this date"}
						},
						"end_time": {
							"type": "string",
							"format": "date-time",
							"description": "Filter bounces created before this time (ISO 8601, e.g. 2026-01-31T23:59:59Z)",
							"x-ui": {"widget": "datetime", "label": "End time", "help_text": "Only include bounces before this date"}
						},
						"limit": {
							"type": "integer",
							"minimum": 1,
							"description": "Maximum number of results to return",
							"x-ui": {"label": "Max results"}
						},
						"offset": {
							"type": "integer",
							"minimum": 0,
							"description": "Number of results to skip for pagination",
							"x-ui": {"label": "Offset"}
						}
					}
				}`)),
			},
			{
				ActionType:  "sendgrid.get_suppressions",
				Name:        "Get Suppression List",
				Description: "Retrieve the global unsubscribe list — contacts who have opted out of all email. Useful for compliance and deliverability auditing.",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"properties": {
						"limit": {
							"type": "integer",
							"minimum": 1,
							"description": "Maximum number of results to return",
							"x-ui": {"label": "Max results"}
						},
						"offset": {
							"type": "integer",
							"minimum": 0,
							"description": "Number of results to skip for pagination",
							"x-ui": {"label": "Offset"}
						}
					}
				}`)),
			},
		},
		RequiredCredentials: []connectors.ManifestCredential{
			{
				Service:       "sendgrid_oauth",
				AuthType:      "oauth2",
				OAuthProvider: "sendgrid",
				OAuthScopes: []string{
					"openid",
					"profile",
					"email",
				},
			},
			{
				Service:         "sendgrid",
				AuthType:        "api_key",
				InstructionsURL: "https://docs.sendgrid.com/ui/account-and-settings/api-keys",
			},
		},
		Templates: []connectors.ManifestTemplate{
			{
				ID:          "tpl_sendgrid_send_campaign",
				ActionType:  "sendgrid.send_campaign",
				Name:        "Send email campaign",
				Description: "Agent can create and send email campaigns to any list. High risk — consider using the locked-list template instead.",
				Parameters:  json.RawMessage(`{"name":"*","subject":"*","html_content":"*","plain_content":"*","list_ids":"*","sender_id":"*"}`),
			},
			{
				ID:          "tpl_sendgrid_send_to_list",
				ActionType:  "sendgrid.send_campaign",
				Name:        "Send campaign to specific list",
				Description: "Locks the recipient list and sender — agent can only customize email content. Safer than the unrestricted template.",
				Parameters:  json.RawMessage(`{"name":"*","subject":"*","html_content":"*","plain_content":"*","list_ids":["YOUR_LIST_ID"],"sender_id":0}`),
			},
			{
				ID:          "tpl_sendgrid_schedule_campaign",
				ActionType:  "sendgrid.schedule_campaign",
				Name:        "Schedule email campaign",
				Description: "Agent can create and schedule email campaigns for future delivery.",
				Parameters:  json.RawMessage(`{"name":"*","subject":"*","html_content":"*","plain_content":"*","list_ids":"*","sender_id":"*","send_at":"*"}`),
			},
			{
				ID:          "tpl_sendgrid_add_subscriber",
				ActionType:  "sendgrid.add_to_list",
				Name:        "Add subscriber to list",
				Description: "Agent can add contacts to any list.",
				Parameters:  json.RawMessage(`{"list_id":"*","email":"*","first_name":"*","last_name":"*"}`),
			},
			{
				ID:          "tpl_sendgrid_add_to_specific_list",
				ActionType:  "sendgrid.add_to_list",
				Name:        "Add subscriber to specific list",
				Description: "Locks the target list — agent can only add contacts to this specific list.",
				Parameters:  json.RawMessage(`{"list_id":"YOUR_LIST_ID","email":"*","first_name":"*","last_name":"*"}`),
			},
			{
				ID:          "tpl_sendgrid_remove_subscriber",
				ActionType:  "sendgrid.remove_from_list",
				Name:        "Remove subscriber from list",
				Description: "Agent can remove contacts from any list.",
				Parameters:  json.RawMessage(`{"list_id":"*","contact_id":"*"}`),
			},
			{
				ID:          "tpl_sendgrid_create_template",
				ActionType:  "sendgrid.create_template",
				Name:        "Create email templates",
				Description: "Agent can create new email templates.",
				Parameters:  json.RawMessage(`{"name":"*","generation":"dynamic"}`),
			},
			{
				ID:          "tpl_sendgrid_get_stats",
				ActionType:  "sendgrid.get_campaign_stats",
				Name:        "View campaign analytics",
				Description: "Agent can check campaign stats like opens, clicks, and bounces.",
				Parameters:  json.RawMessage(`{"singlesend_id":"*"}`),
			},
			{
				ID:          "tpl_sendgrid_list_segments",
				ActionType:  "sendgrid.list_segments",
				Name:        "List contact segments",
				Description: "Agent can list all contact segments.",
				Parameters:  json.RawMessage(`{}`),
			},
			{
				ID:          "tpl_sendgrid_list_senders",
				ActionType:  "sendgrid.list_senders",
				Name:        "List verified senders",
				Description: "Agent can list all verified sender identities to find sender IDs.",
				Parameters:  json.RawMessage(`{}`),
			},
			{
				ID:          "tpl_sendgrid_list_lists",
				ActionType:  "sendgrid.list_lists",
				Name:        "List contact lists",
				Description: "Agent can list all contact lists to find list IDs.",
				Parameters:  json.RawMessage(`{}`),
			},
			{
				ID:          "tpl_sendgrid_send_transactional_email",
				ActionType:  "sendgrid.send_transactional_email",
				Name:        "Send transactional email",
				Description: "Agent can send transactional emails (welcome, password reset, notifications) to any recipient from any verified sender.",
				Parameters:  json.RawMessage(`{"to":"*","to_name":"*","from":"*","from_name":"*","subject":"*","html_content":"*","plain_content":"*","template_id":"*","dynamic_template_data":"*","reply_to":"*"}`),
			},
			{
				ID:          "tpl_sendgrid_send_transactional_locked_sender",
				ActionType:  "sendgrid.send_transactional_email",
				Name:        "Send transactional email (locked sender)",
				Description: "Locks the sender address — agent can only send from the specified verified sender. Safer than the unrestricted template.",
				Parameters:  json.RawMessage(`{"to":"*","to_name":"*","from":"YOUR_VERIFIED_SENDER@example.com","from_name":"*","subject":"*","html_content":"*","plain_content":"*","template_id":"*","dynamic_template_data":"*"}`),
			},
			{
				ID:          "tpl_sendgrid_create_contact",
				ActionType:  "sendgrid.create_contact",
				Name:        "Create contact",
				Description: "Agent can add or update contacts in SendGrid without assigning them to a list.",
				Parameters:  json.RawMessage(`{"email":"*","first_name":"*","last_name":"*","phone_number":"*","city":"*","country":"*"}`),
			},
			{
				ID:          "tpl_sendgrid_get_bounces",
				ActionType:  "sendgrid.get_bounces",
				Name:        "View bounce list",
				Description: "Agent can retrieve the bounce list for deliverability monitoring.",
				Parameters:  json.RawMessage(`{"start_time":"*","end_time":"*","limit":"*","offset":"*"}`),
			},
			{
				ID:          "tpl_sendgrid_get_suppressions",
				ActionType:  "sendgrid.get_suppressions",
				Name:        "View suppression/unsubscribe list",
				Description: "Agent can retrieve the global unsubscribe list for compliance and deliverability auditing.",
				Parameters:  json.RawMessage(`{"limit":"*","offset":"*"}`),
			},
		},
	}
}
