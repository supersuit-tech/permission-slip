package intercom

import (
	"encoding/json"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// Manifest returns the connector's metadata manifest. Used by the server to
// auto-seed DB rows on startup.
func (c *IntercomConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "intercom",
		Name:        "Intercom",
		Description: "Intercom integration for ticket management, customer communication, and support workflows",
		Actions: []connectors.ManifestAction{
			{
				ActionType:  "intercom.create_ticket",
				Name:        "Create Ticket",
				Description: "Create a new ticket in Intercom",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["title", "ticket_type_id", "contact_id"],
					"properties": {
						"title": {
							"type": "string",
							"description": "Ticket title"
						},
						"description": {
							"type": "string",
							"description": "Ticket description"
						},
						"ticket_type_id": {
							"type": "string",
							"description": "Intercom ticket type ID"
						},
						"contact_id": {
							"type": "string",
							"description": "Intercom contact ID of the requester"
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
							"description": "Custom ticket attributes"
						}
					}
				}`)),
			},
			{
				ActionType:  "intercom.reply_ticket",
				Name:        "Reply to Ticket",
				Description: "Add a public reply or internal note to a ticket",
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
							"description": "Reply body text (HTML supported)"
						},
						"message_type": {
							"type": "string",
							"enum": ["comment", "note"],
							"description": "Whether the reply is a public comment or internal note (default: comment)",
							"default": "comment"
						},
						"admin_id": {
							"type": "string",
							"description": "Intercom admin ID of the replying agent"
						}
					}
				}`)),
			},
			{
				ActionType:  "intercom.update_ticket",
				Name:        "Update Ticket",
				Description: "Update ticket state, title, or attributes",
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
							"description": "New ticket state"
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
				Description: "Assign a ticket to an admin or team",
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
							"description": "Intercom admin or team ID to assign the ticket to"
						}
					}
				}`)),
			},
			{
				ActionType:  "intercom.search_tickets",
				Name:        "Search Tickets",
				Description: "Search tickets using Intercom search query",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["field", "operator", "value"],
					"properties": {
						"field": {
							"type": "string",
							"description": "Field to search on (e.g. 'state', 'title', 'ticket_type_id')"
						},
						"operator": {
							"type": "string",
							"enum": ["=", "!=", ">", "<", "~", "IN", "NIN"],
							"description": "Search operator"
						},
						"value": {
							"type": "string",
							"description": "Value to search for"
						}
					}
				}`)),
			},
			{
				ActionType:  "intercom.list_tags",
				Name:        "List Tags",
				Description: "List all available tags in Intercom",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"properties": {}
				}`)),
			},
			{
				ActionType:  "intercom.tag_ticket",
				Name:        "Tag Ticket",
				Description: "Apply a tag to a ticket",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["tag_name", "ticket_id"],
					"properties": {
						"tag_name": {
							"type": "string",
							"description": "Name of the tag to apply"
						},
						"ticket_id": {
							"type": "string",
							"description": "Intercom ticket ID to tag"
						}
					}
				}`)),
			},
		},
		RequiredCredentials: []connectors.ManifestCredential{
			{
				Service:         "intercom",
				AuthType:        "api_key",
				InstructionsURL: "https://developers.intercom.com/docs/build-an-integration/learn-more/authentication/",
			},
		},
		Templates: []connectors.ManifestTemplate{
			{
				ID:          "tpl_intercom_search_open",
				ActionType:  "intercom.search_tickets",
				Name:        "Search open tickets",
				Description: "Find all tickets in submitted or in_progress state.",
				Parameters:  json.RawMessage(`{"field":"state","operator":"=","value":"submitted"}`),
			},
			{
				ID:          "tpl_intercom_reply_with_approval",
				ActionType:  "intercom.reply_ticket",
				Name:        "Reply to customer (with approval)",
				Description: "Send a public reply to a customer. Requires approval before sending.",
				Parameters:  json.RawMessage(`{"ticket_id":"*","body":"*","admin_id":"*","message_type":"comment"}`),
			},
			{
				ID:          "tpl_intercom_add_note",
				ActionType:  "intercom.reply_ticket",
				Name:        "Add internal note",
				Description: "Add an internal note to a ticket (not visible to customer).",
				Parameters:  json.RawMessage(`{"ticket_id":"*","body":"*","admin_id":"*","message_type":"note"}`),
			},
		},
	}
}
