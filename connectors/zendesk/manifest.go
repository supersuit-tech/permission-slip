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
							"description": "Ticket subject line"
						},
						"description": {
							"type": "string",
							"description": "Ticket body/description"
						},
						"priority": {
							"type": "string",
							"enum": ["urgent", "high", "normal", "low"],
							"description": "Ticket priority"
						},
						"type": {
							"type": "string",
							"enum": ["problem", "incident", "question", "task"],
							"description": "Ticket type"
						},
						"tags": {
							"type": "array",
							"items": {"type": "string"},
							"description": "Tags to apply to the ticket"
						},
						"requester_id": {
							"type": "integer",
							"description": "Zendesk user ID of the requester"
						}
					}
				}`)),
			},
			{
				ActionType:  "zendesk.reply_ticket",
				Name:        "Reply to Ticket",
				Description: "Add a public reply or internal note to a ticket",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["ticket_id", "body"],
					"properties": {
						"ticket_id": {
							"type": "integer",
							"description": "Zendesk ticket ID"
						},
						"body": {
							"type": "string",
							"description": "Comment body text"
						},
						"public": {
							"type": "boolean",
							"description": "Whether the reply is public (customer-visible) or an internal note (default: false)",
							"default": false
						}
					}
				}`)),
			},
			{
				ActionType:  "zendesk.update_ticket",
				Name:        "Update Ticket Status",
				Description: "Update ticket status, priority, or type",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["ticket_id"],
					"properties": {
						"ticket_id": {
							"type": "integer",
							"description": "Zendesk ticket ID"
						},
						"status": {
							"type": "string",
							"enum": ["new", "open", "pending", "hold", "solved", "closed"],
							"description": "New ticket status"
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
				Description: "Assign a ticket to an agent or group",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["ticket_id"],
					"properties": {
						"ticket_id": {
							"type": "integer",
							"description": "Zendesk ticket ID"
						},
						"assignee_id": {
							"type": "integer",
							"description": "Zendesk user ID of the agent to assign"
						},
						"group_id": {
							"type": "integer",
							"description": "Zendesk group ID to assign"
						}
					}
				}`)),
			},
			{
				ActionType:  "zendesk.merge_tickets",
				Name:        "Merge Tickets",
				Description: "Merge duplicate tickets into a target ticket (destructive, irreversible)",
				RiskLevel:   "high",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["target_id", "source_ids"],
					"properties": {
						"target_id": {
							"type": "integer",
							"description": "Zendesk ticket ID to merge into (target)"
						},
						"source_ids": {
							"type": "array",
							"items": {"type": "integer"},
							"description": "Zendesk ticket IDs to merge from (sources, max 5)"
						}
					}
				}`)),
			},
			{
				ActionType:  "zendesk.search_tickets",
				Name:        "Search Tickets",
				Description: "Search tickets using Zendesk search query syntax",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["query"],
					"properties": {
						"query": {
							"type": "string",
							"description": "Zendesk search query (e.g. 'status:open assignee:me priority:high')"
						},
						"sort_by": {
							"type": "string",
							"enum": ["updated_at", "created_at", "priority", "status", "relevance"],
							"description": "Field to sort results by"
						},
						"sort_order": {
							"type": "string",
							"enum": ["asc", "desc"],
							"description": "Sort direction"
						}
					}
				}`)),
			},
			{
				ActionType:  "zendesk.list_tags",
				Name:        "List Ticket Tags",
				Description: "List all tags on a ticket",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["ticket_id"],
					"properties": {
						"ticket_id": {
							"type": "integer",
							"description": "Zendesk ticket ID"
						}
					}
				}`)),
			},
			{
				ActionType:  "zendesk.update_tags",
				Name:        "Update Ticket Tags",
				Description: "Replace all tags on a ticket",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["ticket_id", "tags"],
					"properties": {
						"ticket_id": {
							"type": "integer",
							"description": "Zendesk ticket ID"
						},
						"tags": {
							"type": "array",
							"items": {"type": "string"},
							"description": "New set of tags (replaces existing tags)"
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
				Description: "Search for open, unassigned tickets to triage.",
				Parameters:  json.RawMessage(`{"query":"status:open assignee:none"}`),
			},
			{
				ID:          "tpl_zendesk_reply_with_approval",
				ActionType:  "zendesk.reply_ticket",
				Name:        "Reply to customer (with approval)",
				Description: "Send a public reply to a customer on a ticket. Requires approval before sending.",
				Parameters:  json.RawMessage(`{"ticket_id":"*","body":"*","public":true}`),
			},
			{
				ID:          "tpl_zendesk_add_internal_note",
				ActionType:  "zendesk.reply_ticket",
				Name:        "Add internal note",
				Description: "Add an internal note to a ticket (not visible to customer).",
				Parameters:  json.RawMessage(`{"ticket_id":"*","body":"*","public":false}`),
			},
		},
	}
}
