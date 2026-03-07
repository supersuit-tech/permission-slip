package zendesk

import (
	"encoding/json"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// Manifest returns the connector's metadata manifest. Used by the server to
// auto-seed DB rows on startup.
func (c *ZendeskConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "zendesk",
		Name:        "Zendesk",
		Description: "Zendesk Support integration for ticket management, customer communication, and support workflows",
		Actions: []connectors.ManifestAction{
			{
				ActionType:  "zendesk.create_ticket",
				Name:        "Create Ticket",
				Description: "Create a new support ticket in Zendesk",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["subject"],
					"properties": {
						"subject": {
							"type": "string",
							"description": "Ticket subject line (e.g. 'Password reset not working')",
							"maxLength": 150
						},
						"description": {
							"type": "string",
							"description": "Ticket body/description with details about the issue"
						},
						"priority": {
							"type": "string",
							"enum": ["urgent", "high", "normal", "low"],
							"description": "Ticket priority (default: no priority set)"
						},
						"type": {
							"type": "string",
							"enum": ["problem", "incident", "question", "task"],
							"description": "Ticket type — 'problem' for bugs, 'incident' for service disruptions, 'question' for inquiries, 'task' for to-dos"
						},
						"tags": {
							"type": "array",
							"items": {"type": "string"},
							"description": "Tags to apply to the ticket (e.g. ['billing', 'vip'])",
							"maxItems": 100
						},
						"requester_id": {
							"type": "integer",
							"description": "Zendesk user ID of the requester — omit to use the authenticated agent",
							"minimum": 1
						}
					}
				}`)),
			},
			{
				ActionType:  "zendesk.reply_ticket",
				Name:        "Reply to Ticket",
				Description: "Add a public reply or internal note to a ticket. Public replies are visible to the customer; internal notes are only visible to agents.",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["ticket_id", "body"],
					"properties": {
						"ticket_id": {
							"type": "integer",
							"description": "Zendesk ticket ID (e.g. 12345)",
							"minimum": 1
						},
						"body": {
							"type": "string",
							"description": "Comment body text — supports plain text"
						},
						"public": {
							"type": "boolean",
							"description": "If true, reply is visible to the customer. If false, adds an internal note visible only to agents.",
							"default": false
						}
					}
				}`)),
			},
			{
				ActionType:  "zendesk.update_ticket",
				Name:        "Update Ticket Status",
				Description: "Update ticket status, priority, or type. At least one field must be provided.",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["ticket_id"],
					"properties": {
						"ticket_id": {
							"type": "integer",
							"description": "Zendesk ticket ID (e.g. 12345)",
							"minimum": 1
						},
						"status": {
							"type": "string",
							"enum": ["new", "open", "pending", "hold", "solved", "closed"],
							"description": "New ticket status — note: 'closed' is permanent and cannot be undone"
						},
						"priority": {
							"type": "string",
							"enum": ["urgent", "high", "normal", "low"],
							"description": "New ticket priority"
						},
						"type": {
							"type": "string",
							"enum": ["problem", "incident", "question", "task"],
							"description": "New ticket type"
						}
					}
				}`)),
			},
			{
				ActionType:  "zendesk.assign_ticket",
				Name:        "Assign Ticket",
				Description: "Assign a ticket to an agent or group. Provide assignee_id, group_id, or both.",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["ticket_id"],
					"properties": {
						"ticket_id": {
							"type": "integer",
							"description": "Zendesk ticket ID (e.g. 12345)",
							"minimum": 1
						},
						"assignee_id": {
							"type": "integer",
							"description": "Zendesk user ID of the agent to assign (find via Zendesk admin or API)",
							"minimum": 1
						},
						"group_id": {
							"type": "integer",
							"description": "Zendesk group ID to assign (find via Admin > People > Groups)",
							"minimum": 1
						}
					}
				}`)),
			},
			{
				ActionType:  "zendesk.merge_tickets",
				Name:        "Merge Tickets",
				Description: "Merge duplicate tickets into a target ticket. This is destructive and irreversible — source tickets will be closed with a merge comment. The merge runs asynchronously; the response contains a job status URL to track completion.",
				RiskLevel:   "high",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["target_id", "source_ids"],
					"properties": {
						"target_id": {
							"type": "integer",
							"description": "Zendesk ticket ID to merge into (this ticket remains open)",
							"minimum": 1
						},
						"source_ids": {
							"type": "array",
							"items": {"type": "integer", "minimum": 1},
							"description": "Zendesk ticket IDs to merge from (these tickets will be closed)",
							"minItems": 1,
							"maxItems": 5
						}
					}
				}`)),
			},
			{
				ActionType:  "zendesk.search_tickets",
				Name:        "Search Tickets",
				Description: "Search tickets using Zendesk search query syntax. The query is automatically scoped to tickets (type:ticket is prepended). Returns the first page of results by default.",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["query"],
					"properties": {
						"query": {
							"type": "string",
							"description": "Zendesk search query (e.g. 'status:open assignee:me', 'priority:high created>2024-01-01', 'subject:refund')"
						},
						"sort_by": {
							"type": "string",
							"enum": ["updated_at", "created_at", "priority", "status", "relevance"],
							"description": "Field to sort results by (default: relevance)"
						},
						"sort_order": {
							"type": "string",
							"enum": ["asc", "desc"],
							"description": "Sort direction (default: desc)"
						},
						"page": {
							"type": "integer",
							"description": "Page number for paginated results (default: 1)",
							"minimum": 1,
							"default": 1
						},
						"per_page": {
							"type": "integer",
							"description": "Number of results per page (default: 25, max: 100)",
							"minimum": 1,
							"maximum": 100,
							"default": 25
						}
					}
				}`)),
			},
			{
				ActionType:  "zendesk.list_tags",
				Name:        "List Ticket Tags",
				Description: "List all tags currently applied to a specific ticket",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["ticket_id"],
					"properties": {
						"ticket_id": {
							"type": "integer",
							"description": "Zendesk ticket ID (e.g. 12345)",
							"minimum": 1
						}
					}
				}`)),
			},
			{
				ActionType:  "zendesk.update_tags",
				Name:        "Update Ticket Tags",
				Description: "Replace all tags on a ticket with a new set. To add tags without removing existing ones, first list the current tags and include them in the new set.",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["ticket_id", "tags"],
					"properties": {
						"ticket_id": {
							"type": "integer",
							"description": "Zendesk ticket ID (e.g. 12345)",
							"minimum": 1
						},
						"tags": {
							"type": "array",
							"items": {"type": "string"},
							"description": "New set of tags — replaces all existing tags (e.g. ['billing', 'escalated'])",
							"minItems": 1,
							"maxItems": 100
						}
					}
				}`)),
			},
		},
		RequiredCredentials: []connectors.ManifestCredential{
			{
				Service:         "zendesk",
				AuthType:        "custom",
				InstructionsURL: "https://developer.zendesk.com/api-reference/introduction/security-and-auth/",
			},
		},
		Templates: []connectors.ManifestTemplate{
			{
				ID:          "tpl_zendesk_triage_tickets",
				ActionType:  "zendesk.search_tickets",
				Name:        "Triage open tickets",
				Description: "Search for open, unassigned tickets to triage. Agent can modify the search query but results are always scoped to tickets.",
				Parameters:  json.RawMessage(`{"query":"status:open assignee:none"}`),
			},
			{
				ID:          "tpl_zendesk_reply_with_approval",
				ActionType:  "zendesk.reply_ticket",
				Name:        "Reply to customer (with approval)",
				Description: "Send a public reply to a customer on any ticket. Agent chooses the ticket and writes the message; each reply requires approval before sending.",
				Parameters:  json.RawMessage(`{"ticket_id":"*","body":"*","public":true}`),
			},
			{
				ID:          "tpl_zendesk_add_internal_note",
				ActionType:  "zendesk.reply_ticket",
				Name:        "Add internal note",
				Description: "Add an internal note to any ticket. Notes are only visible to agents, never to the customer. Agent can write freely without approval risk.",
				Parameters:  json.RawMessage(`{"ticket_id":"*","body":"*","public":false}`),
			},
			{
				ID:          "tpl_zendesk_close_ticket",
				ActionType:  "zendesk.update_ticket",
				Name:        "Close resolved tickets",
				Description: "Mark a ticket as solved. Agent chooses which ticket to close; the status is locked to 'solved' to prevent accidental closures to other states.",
				Parameters:  json.RawMessage(`{"ticket_id":"*","status":"solved"}`),
			},
			{
				ID:          "tpl_zendesk_escalate_priority",
				ActionType:  "zendesk.update_ticket",
				Name:        "Escalate ticket priority",
				Description: "Raise a ticket's priority to urgent. Agent chooses the ticket; priority is locked to 'urgent' to ensure only true escalations use this template.",
				Parameters:  json.RawMessage(`{"ticket_id":"*","priority":"urgent"}`),
			},
		},
	}
}
