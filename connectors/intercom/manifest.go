package intercom

import (
	_ "embed"
	"encoding/json"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// Manifest returns the connector's metadata manifest. Used by the server to
// auto-seed DB rows on startup.
//go:embed logo.svg
var logoSVG string

func (c *IntercomConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "intercom",
		Name:        "Intercom",
		Description: "Intercom integration for ticket management, customer communication, and support workflows",
		LogoSVG:     logoSVG,
		Actions: []connectors.ManifestAction{
			{
				ActionType:  "intercom.create_ticket",
				Name:        "Create Ticket",
				Description: "Create a new ticket in Intercom. Requires a ticket type and contact — find these IDs via the Intercom admin or API.",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["title", "ticket_type_id", "contact_id"],
					"properties": {
						"title": {
							"type": "string",
							"description": "Ticket title (e.g. 'Password reset request')"
						},
						"description": {
							"type": "string",
							"description": "Ticket description with details about the issue"
						},
						"ticket_type_id": {
							"type": "string",
							"description": "Intercom ticket type ID — find via Settings > Tickets > Ticket types, or the API"
						},
						"contact_id": {
							"type": "string",
							"description": "Intercom contact ID of the requester (the user or lead who submitted the request)"
						},
						"attributes": {
							"type": "array",
							"items": {
								"type": "object",
								"required": ["name", "value"],
								"properties": {
									"name": {"type": "string", "description": "Attribute name (must match a defined ticket attribute)"},
									"value": {"type": "string", "description": "Attribute value"}
								}
							},
							"description": "Custom ticket attributes defined in your ticket type (e.g. [{'name': 'severity', 'value': 'high'}])"
						}
					}
				}`)),
			},
			{
				ActionType:  "intercom.reply_ticket",
				Name:        "Reply to Ticket",
				Description: "Add a public reply or internal note to a ticket. Public comments are visible to the customer; notes are only visible to teammates.",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["ticket_id", "body", "admin_id"],
					"properties": {
						"ticket_id": {
							"type": "string",
							"description": "Intercom ticket ID"
						},
						"body": {
							"type": "string",
							"description": "Reply body text — HTML is supported (e.g. '<b>bold</b>', '<a href=\"...\">link</a>')"
						},
						"message_type": {
							"type": "string",
							"enum": ["comment", "note"],
							"description": "Message type — 'comment' for customer-visible replies, 'note' for internal-only notes",
							"default": "comment"
						},
						"admin_id": {
							"type": "string",
							"description": "Intercom admin ID of the replying agent (find via Settings > Teammates, or the API)"
						}
					}
				}`)),
			},
			{
				ActionType:  "intercom.update_ticket",
				Name:        "Update Ticket",
				Description: "Update ticket state, title, or custom attributes. At least one field must be provided.",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["ticket_id"],
					"properties": {
						"ticket_id": {
							"type": "string",
							"description": "Intercom ticket ID"
						},
						"state": {
							"type": "string",
							"enum": ["submitted", "in_progress", "waiting_on_customer", "resolved"],
							"description": "New ticket state — not all transitions are valid (e.g. resolved tickets cannot go back to submitted)"
						},
						"title": {
							"type": "string",
							"description": "New ticket title"
						},
						"attributes": {
							"type": "array",
							"items": {
								"type": "object",
								"required": ["name", "value"],
								"properties": {
									"name": {"type": "string", "description": "Attribute name"},
									"value": {"type": "string", "description": "Attribute value"}
								}
							},
							"description": "Custom ticket attributes to update"
						}
					}
				}`)),
			},
			{
				ActionType:  "intercom.assign_ticket",
				Name:        "Assign Ticket",
				Description: "Assign a ticket to an admin or team. The assignee must have access to the ticket's workspace.",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["ticket_id", "assignee_id"],
					"properties": {
						"ticket_id": {
							"type": "string",
							"description": "Intercom ticket ID"
						},
						"assignee_id": {
							"type": "string",
							"description": "Intercom admin or team ID to assign the ticket to (find via Settings > Teammates or Teams)"
						}
					}
				}`)),
			},
			{
				ActionType:  "intercom.search_tickets",
				Name:        "Search Tickets",
				Description: "Search tickets using a single-field filter. Use '=' for exact match, '~' for contains, 'IN' for multiple values.",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["field", "operator", "value"],
					"properties": {
						"field": {
							"type": "string",
							"description": "Field to search on (e.g. 'state', 'title', 'ticket_type_id', 'created_at', 'updated_at')"
						},
						"operator": {
							"type": "string",
							"enum": ["=", "!=", ">", "<", "~", "IN", "NIN"],
							"description": "Search operator — '=' exact match, '!=' not equal, '>' / '<' for dates, '~' contains, 'IN' / 'NIN' for lists"
						},
						"value": {
							"type": "string",
							"description": "Value to search for (e.g. 'submitted', 'billing issue')"
						}
					}
				}`)),
			},
			{
				ActionType:  "intercom.list_tags",
				Name:        "List Tags",
				Description: "List all available tags in the Intercom workspace. Use these tag names with the Tag Ticket action.",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"properties": {}
				}`)),
			},
			{
				ActionType:  "intercom.tag_ticket",
				Name:        "Tag Ticket",
				Description: "Apply a tag to a ticket. The tag will be created if it doesn't already exist.",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["tag_name", "ticket_id"],
					"properties": {
						"tag_name": {
							"type": "string",
							"description": "Name of the tag to apply (e.g. 'vip', 'billing')"
						},
						"ticket_id": {
							"type": "string",
							"description": "Intercom ticket ID to tag"
						}
					}
				}`)),
			},
			{
				ActionType:  "intercom.create_contact",
				Name:        "Create Contact",
				Description: "Create a new contact (user or lead) in Intercom.",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"properties": {
						"email": {
							"type": "string",
							"description": "Contact email address"
						},
						"phone": {
							"type": "string",
							"description": "Contact phone number"
						},
						"name": {
							"type": "string",
							"description": "Contact full name"
						},
						"role": {
							"type": "string",
							"enum": ["user", "lead"],
							"description": "Contact role — 'user' for identified users, 'lead' for anonymous leads (default: lead)"
						},
						"custom_attributes": {
							"type": "object",
							"description": "Custom attributes to set on the contact",
							"additionalProperties": true
						}
					}
				}`)),
			},
			{
				ActionType:  "intercom.update_contact",
				Name:        "Update Contact",
				Description: "Update attributes on an existing Intercom contact.",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["contact_id"],
					"properties": {
						"contact_id": {
							"type": "string",
							"description": "Intercom contact ID to update"
						},
						"email": {
							"type": "string",
							"description": "Updated email address"
						},
						"phone": {
							"type": "string",
							"description": "Updated phone number"
						},
						"name": {
							"type": "string",
							"description": "Updated full name"
						},
						"custom_attributes": {
							"type": "object",
							"description": "Custom attributes to update",
							"additionalProperties": true
						}
					}
				}`)),
			},
			{
				ActionType:  "intercom.search_contacts",
				Name:        "Search Contacts",
				Description: "Search contacts by a single field (email, name, phone, etc.).",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["query"],
					"properties": {
						"query": {
							"type": "object",
							"required": ["field", "operator", "value"],
							"properties": {
								"field": {
									"type": "string",
									"description": "Field to search on (e.g. 'email', 'name', 'phone')"
								},
								"operator": {
									"type": "string",
									"enum": ["=", "!=", "IN", "NIN", ">", "<", "~", "!~", "^", "$"],
									"description": "Search operator"
								},
								"value": {
									"type": "string",
									"description": "Value to match against"
								}
							}
						},
						"limit": {
							"type": "integer",
							"default": 20,
							"description": "Maximum number of results (default 20, max 150)"
						}
					}
				}`)),
			},
			{
				ActionType:  "intercom.send_message",
				Name:        "Send Message",
				Description: "Send a proactive outbound message to a contact (in-app or email). This is not a ticket reply — it creates a new outbound conversation.",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["body", "from_admin_id", "to_contact_id"],
					"properties": {
						"body": {
							"type": "string",
							"description": "Message body (HTML supported)"
						},
						"message_type": {
							"type": "string",
							"enum": ["inapp", "email"],
							"default": "inapp",
							"description": "Delivery channel — 'inapp' for in-product messages, 'email' for email delivery"
						},
						"subject": {
							"type": "string",
							"description": "Email subject line (required when message_type is email)"
						},
						"from_admin_id": {
							"type": "string",
							"description": "Intercom admin ID to send from (find via Settings > Teammates)"
						},
						"to_contact_id": {
							"type": "string",
							"description": "Intercom contact ID to send to"
						}
					}
				}`)),
			},
			{
				ActionType:  "intercom.list_conversations",
				Name:        "List Conversations",
				Description: "List conversations with optional state filter (open, closed, snoozed).",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"properties": {
						"state": {
							"type": "string",
							"enum": ["open", "closed", "snoozed"],
							"description": "Filter by conversation state (omit for all states)"
						},
						"limit": {
							"type": "integer",
							"default": 20,
							"description": "Maximum number of results (default 20, max 150)"
						}
					}
				}`)),
			},
			{
				ActionType:  "intercom.create_article",
				Name:        "Create Article",
				Description: "Create a help center article. Articles are created as drafts by default — set state to 'published' to make them publicly visible immediately.",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["title", "author_id"],
					"properties": {
						"title": {
							"type": "string",
							"description": "Article title"
						},
						"body": {
							"type": "string",
							"description": "Article content (HTML supported)"
						},
						"author_id": {
							"type": "string",
							"description": "Intercom admin ID of the article author"
						},
						"state": {
							"type": "string",
							"enum": ["draft", "published"],
							"default": "draft",
							"description": "Publication state — 'draft' is the safe default; 'published' makes it immediately visible"
						},
						"parent_id": {
							"type": "integer",
							"description": "Collection ID to place the article in (optional)"
						},
						"parent_type": {
							"type": "string",
							"description": "Parent type — must be 'collection' when parent_id is set"
						}
					}
				}`)),
			},
		},
		RequiredCredentials: []connectors.ManifestCredential{
			{
				Service:       "intercom_oauth",
				AuthType:      "oauth2",
				OAuthProvider: "intercom",
				OAuthScopes:   []string{},
			},
			{
				Service:         "intercom",
				AuthType:        "api_key",
				InstructionsURL: "https://developers.intercom.com/docs/build-an-integration/learn-more/authentication/",
			},
		},
		OAuthProviders: []connectors.ManifestOAuthProvider{
			{
				ID:           "intercom",
				AuthorizeURL: "https://app.intercom.com/oauth",
				TokenURL:     "https://api.intercom.io/auth/eagle/token",
				Scopes:       []string{},
			},
		},
		Templates: []connectors.ManifestTemplate{
			{
				ID:          "tpl_intercom_search_open",
				ActionType:  "intercom.search_tickets",
				Name:        "Search open tickets",
				Description: "Find all tickets in submitted state. Agent can see the queue of new tickets waiting for triage.",
				Parameters:  json.RawMessage(`{"field":"state","operator":"=","value":"submitted"}`),
			},
			{
				ID:          "tpl_intercom_reply_with_approval",
				ActionType:  "intercom.reply_ticket",
				Name:        "Reply to customer (with approval)",
				Description: "Send a public reply to a customer on any ticket. Agent chooses the ticket and writes the message; each reply requires approval before sending.",
				Parameters:  json.RawMessage(`{"ticket_id":"*","body":"*","admin_id":"*","message_type":"comment"}`),
			},
			{
				ID:          "tpl_intercom_add_note",
				ActionType:  "intercom.reply_ticket",
				Name:        "Add internal note",
				Description: "Add an internal note to any ticket. Notes are only visible to teammates, never to the customer. Agent can write freely without customer-facing risk.",
				Parameters:  json.RawMessage(`{"ticket_id":"*","body":"*","admin_id":"*","message_type":"note"}`),
			},
			{
				ID:          "tpl_intercom_resolve_ticket",
				ActionType:  "intercom.update_ticket",
				Name:        "Resolve ticket",
				Description: "Mark a ticket as resolved. Agent chooses which ticket to resolve; the state is locked to 'resolved' to prevent accidental state changes.",
				Parameters:  json.RawMessage(`{"ticket_id":"*","state":"resolved"}`),
			},
		},
	}
}
