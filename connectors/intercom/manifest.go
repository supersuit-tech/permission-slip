package intercom

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
							"description": "Ticket title (e.g. 'Password reset request')",
							"x-ui": {
								"label": "Title",
								"placeholder": "e.g. Password reset request"
							}
						},
						"description": {
							"type": "string",
							"description": "Ticket description with details about the issue",
							"x-ui": {
								"label": "Description",
								"placeholder": "Describe the issue...",
								"widget": "textarea"
							}
						},
						"ticket_type_id": {
							"type": "string",
							"description": "Intercom ticket type ID — find via Settings > Tickets > Ticket types, or the API",
							"x-ui": {
								"label": "Ticket Type ID",
								"placeholder": "e.g. 1",
								"help_text": "Find via Settings > Tickets > Ticket types in Intercom"
							}
						},
						"contact_id": {
							"type": "string",
							"description": "Intercom contact ID of the requester (the user or lead who submitted the request)",
							"x-ui": {
								"label": "Contact ID",
								"placeholder": "e.g. 6329f3b5a2e985b564e5e5e1",
								"help_text": "Find via search_contacts or the contact page URL"
							}
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
							"description": "Custom ticket attributes defined in your ticket type (e.g. [{'name': 'severity', 'value': 'high'}])",
							"x-ui": {
								"label": "Custom Attributes"
							}
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
							"description": "Intercom ticket ID",
							"x-ui": {
								"label": "Ticket ID",
								"placeholder": "12345"
							}
						},
						"body": {
							"type": "string",
							"description": "Reply body text — HTML is supported (e.g. '<b>bold</b>', '<a href=\"...\">link</a>')",
							"x-ui": {
								"label": "Body",
								"placeholder": "Write your reply...",
								"widget": "textarea"
							}
						},
						"message_type": {
							"type": "string",
							"enum": ["comment", "note"],
							"description": "Message type — 'comment' for customer-visible replies, 'note' for internal-only notes",
							"default": "comment",
							"x-ui": {
								"label": "Message Type",
								"widget": "select"
							}
						},
						"admin_id": {
							"type": "string",
							"description": "Intercom admin ID of the replying agent (find via Settings > Teammates, or the API)",
							"x-ui": {
								"label": "Admin ID",
								"placeholder": "e.g. 1234567",
								"help_text": "Find via Settings > Teammates in Intercom"
							}
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
							"description": "Intercom ticket ID",
							"x-ui": {
								"label": "Ticket ID",
								"placeholder": "12345"
							}
						},
						"state": {
							"type": "string",
							"enum": ["submitted", "in_progress", "waiting_on_customer", "resolved"],
							"description": "New ticket state — not all transitions are valid (e.g. resolved tickets cannot go back to submitted)",
							"x-ui": {
								"label": "State",
								"widget": "select"
							}
						},
						"title": {
							"type": "string",
							"description": "New ticket title",
							"x-ui": {
								"label": "Title",
								"placeholder": "Updated ticket title"
							}
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
							"description": "Custom ticket attributes to update",
							"x-ui": {
								"label": "Custom Attributes"
							}
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
							"description": "Intercom ticket ID",
							"x-ui": {
								"label": "Ticket ID",
								"placeholder": "12345"
							}
						},
						"assignee_id": {
							"type": "string",
							"description": "Intercom admin or team ID to assign the ticket to (find via Settings > Teammates or Teams)",
							"x-ui": {
								"label": "Assignee ID",
								"placeholder": "e.g. 1234567",
								"help_text": "Admin or team ID — find via Settings > Teammates or Teams"
							}
						}
					}
				}`)),
			},
			{
				ActionType:  "intercom.search_tickets",
				Name:        "Search Tickets",
				Description: "Search tickets using a single-field filter. Use '=' for exact match, '~' for contains, 'IN' for multiple values. Optional created_at / updated_at bounds are combined with AND.",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["field", "operator", "value"],
					"properties": {
						"field": {
							"type": "string",
							"description": "Field to search on (e.g. 'state', 'title', 'ticket_type_id', 'created_at', 'updated_at')",
							"x-ui": {
								"label": "Field",
								"placeholder": "e.g. state"
							}
						},
						"operator": {
							"type": "string",
							"enum": ["=", "!=", ">", "<", "~", "IN", "NIN"],
							"description": "Search operator — '=' exact match, '!=' not equal, '>' / '<' for dates, '~' contains, 'IN' / 'NIN' for lists",
							"x-ui": {
								"label": "Operator",
								"widget": "select"
							}
						},
						"value": {
							"type": "string",
							"description": "Value to search for (e.g. 'submitted', 'billing issue')",
							"x-ui": {
								"label": "Value",
								"placeholder": "e.g. submitted"
							}
						},
						"created_at_after": {
							"type": "string",
							"format": "date-time",
							"description": "Only tickets with created_at strictly after this time (RFC 3339 or Unix timestamp)",
							"x-ui": {
								"label": "Created After",
								"help_text": "RFC 3339 format, e.g. 2026-01-01T00:00:00Z",
								"widget": "datetime",
								"datetime_range_pair": "created_at_before",
								"datetime_range_role": "lower"
							}
						},
						"created_at_before": {
							"type": "string",
							"format": "date-time",
							"description": "Only tickets with created_at strictly before this time (RFC 3339 or Unix timestamp)",
							"x-ui": {
								"label": "Created Before",
								"help_text": "RFC 3339 format, e.g. 2026-01-01T00:00:00Z",
								"widget": "datetime",
								"datetime_range_pair": "created_at_after",
								"datetime_range_role": "upper"
							}
						},
						"updated_at_after": {
							"type": "string",
							"format": "date-time",
							"description": "Only tickets with updated_at strictly after this time (RFC 3339 or Unix timestamp)",
							"x-ui": {
								"label": "Updated After",
								"help_text": "RFC 3339 format, e.g. 2026-01-01T00:00:00Z",
								"widget": "datetime",
								"datetime_range_pair": "updated_at_before",
								"datetime_range_role": "lower"
							}
						},
						"updated_at_before": {
							"type": "string",
							"format": "date-time",
							"description": "Only tickets with updated_at strictly before this time (RFC 3339 or Unix timestamp)",
							"x-ui": {
								"label": "Updated Before",
								"help_text": "RFC 3339 format, e.g. 2026-01-01T00:00:00Z",
								"widget": "datetime",
								"datetime_range_pair": "updated_at_after",
								"datetime_range_role": "upper"
							}
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
							"description": "Name of the tag to apply (e.g. 'vip', 'billing')",
							"x-ui": {
								"label": "Tag Name",
								"placeholder": "e.g. vip"
							}
						},
						"ticket_id": {
							"type": "string",
							"description": "Intercom ticket ID to tag",
							"x-ui": {
								"label": "Ticket ID",
								"placeholder": "12345"
							}
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
							"description": "Contact email address",
							"x-ui": {
								"label": "Email",
								"placeholder": "user@example.com"
							}
						},
						"phone": {
							"type": "string",
							"description": "Contact phone number",
							"x-ui": {
								"label": "Phone",
								"placeholder": "+1-555-000-0000"
							}
						},
						"name": {
							"type": "string",
							"description": "Contact full name",
							"x-ui": {
								"label": "Name",
								"placeholder": "Jane Doe"
							}
						},
						"role": {
							"type": "string",
							"enum": ["user", "lead"],
							"description": "Contact role — 'user' for identified users, 'lead' for anonymous leads (default: lead)",
							"x-ui": {
								"label": "Role",
								"widget": "select"
							}
						},
						"custom_attributes": {
							"type": "object",
							"description": "Custom attributes to set on the contact",
							"additionalProperties": true,
							"x-ui": {
								"label": "Custom Attributes",
								"help_text": "Key-value pairs of custom data attributes defined in your Intercom workspace"
							}
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
							"description": "Intercom contact ID to update",
							"x-ui": {
								"label": "Contact ID",
								"placeholder": "e.g. 6329f3b5a2e985b564e5e5e1",
								"help_text": "Find via search_contacts or the contact page URL"
							}
						},
						"email": {
							"type": "string",
							"description": "Updated email address",
							"x-ui": {
								"label": "Email",
								"placeholder": "user@example.com"
							}
						},
						"phone": {
							"type": "string",
							"description": "Updated phone number",
							"x-ui": {
								"label": "Phone",
								"placeholder": "+1-555-000-0000"
							}
						},
						"name": {
							"type": "string",
							"description": "Updated full name",
							"x-ui": {
								"label": "Name",
								"placeholder": "Jane Doe"
							}
						},
						"custom_attributes": {
							"type": "object",
							"description": "Custom attributes to update",
							"additionalProperties": true,
							"x-ui": {
								"label": "Custom Attributes",
								"help_text": "Key-value pairs of custom data attributes defined in your Intercom workspace"
							}
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
									"description": "Field to search on (e.g. 'email', 'name', 'phone')",
									"x-ui": {
										"label": "Field",
										"placeholder": "e.g. email"
									}
								},
								"operator": {
									"type": "string",
									"enum": ["=", "!=", "IN", "NIN", ">", "<", "~", "!~", "^", "$"],
									"description": "Search operator",
									"x-ui": {
										"label": "Operator",
										"widget": "select"
									}
								},
								"value": {
									"type": "string",
									"description": "Value to match against",
									"x-ui": {
										"label": "Value",
										"placeholder": "e.g. user@example.com"
									}
								}
							},
							"x-ui": {
								"label": "Query"
							}
						},
						"limit": {
							"type": "integer",
							"default": 20,
							"description": "Maximum number of results (default 20, max 150)",
							"x-ui": {
								"label": "Max results",
								"placeholder": "20"
							}
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
							"description": "Message body (HTML supported)",
							"x-ui": {
								"label": "Body",
								"placeholder": "Write your message...",
								"widget": "textarea"
							}
						},
						"message_type": {
							"type": "string",
							"enum": ["inapp", "email"],
							"default": "inapp",
							"description": "Delivery channel — 'inapp' for in-product messages, 'email' for email delivery",
							"x-ui": {
								"label": "Message Type",
								"widget": "select"
							}
						},
						"subject": {
							"type": "string",
							"description": "Email subject line (required when message_type is email)",
							"x-ui": {
								"label": "Subject",
								"placeholder": "Email subject line"
							}
						},
						"from_admin_id": {
							"type": "string",
							"description": "Intercom admin ID to send from (find via Settings > Teammates)",
							"x-ui": {
								"label": "From Admin ID",
								"placeholder": "e.g. 1234567",
								"help_text": "Find via Settings > Teammates in Intercom"
							}
						},
						"to_contact_id": {
							"type": "string",
							"description": "Intercom contact ID to send to",
							"x-ui": {
								"label": "To Contact ID",
								"placeholder": "e.g. 6329f3b5a2e985b564e5e5e1",
								"help_text": "Find via search_contacts or the contact page URL"
							}
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
							"description": "Filter by conversation state (omit for all states)",
							"x-ui": {
								"label": "State",
								"widget": "select"
							}
						},
						"limit": {
							"type": "integer",
							"default": 20,
							"description": "Maximum number of results (default 20, max 150)",
							"x-ui": {
								"label": "Max results",
								"placeholder": "20"
							}
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
							"description": "Article title",
							"x-ui": {
								"label": "Title",
								"placeholder": "e.g. How to reset your password"
							}
						},
						"body": {
							"type": "string",
							"description": "Article content (HTML supported)",
							"x-ui": {
								"label": "Body",
								"placeholder": "Write article content...",
								"widget": "textarea"
							}
						},
						"author_id": {
							"type": "integer",
							"description": "Intercom admin ID of the article author (integer, e.g. 1234567)",
							"x-ui": {
								"label": "Author ID",
								"placeholder": "e.g. 1234567",
								"help_text": "Integer admin ID — find via Settings > Teammates"
							}
						},
						"state": {
							"type": "string",
							"enum": ["draft", "published"],
							"default": "draft",
							"description": "Publication state — 'draft' is the safe default; 'published' makes it immediately visible",
							"x-ui": {
								"label": "State",
								"widget": "select"
							}
						},
						"parent_id": {
							"type": "integer",
							"description": "Collection ID to place the article in (optional)",
							"x-ui": {
								"label": "Parent Collection ID",
								"placeholder": "e.g. 123456",
								"help_text": "Collection ID — find via Help Center > Collections"
							}
						},
						"parent_type": {
							"type": "string",
							"description": "Parent type — must be 'collection' when parent_id is set",
							"x-ui": {
								"label": "Parent Type",
								"placeholder": "collection"
							}
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
			{
				ID:          "tpl_intercom_search_contacts",
				ActionType:  "intercom.search_contacts",
				Name:        "Search contacts by email",
				Description: "Look up an Intercom contact by email address. Returns matching contact IDs needed for other actions like creating tickets.",
				Parameters:  json.RawMessage(`{"query":{"field":"email","operator":"=","value":"*"}}`),
			},
			{
				ID:          "tpl_intercom_send_message_approval",
				ActionType:  "intercom.send_message",
				Name:        "Send message to customer (with approval)",
				Description: "Send a proactive in-app message to a contact. Agent fills in the content and recipient; each message requires approval before sending.",
				Parameters:  json.RawMessage(`{"body":"*","message_type":"inapp","from_admin_id":"*","to_contact_id":"*"}`),
			},
			{
				ID:          "tpl_intercom_list_open_conversations",
				ActionType:  "intercom.list_conversations",
				Name:        "List open conversations",
				Description: "List all open conversations in the Intercom workspace. State is locked to 'open' so the agent always sees the live queue.",
				Parameters:  json.RawMessage(`{"state":"open","limit":20}`),
			},
		},
	}
}
