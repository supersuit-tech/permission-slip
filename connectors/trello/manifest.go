package trello

import (
	"encoding/json"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// Manifest returns the connector's metadata manifest. Used by the server to
// auto-seed DB rows on startup, replacing manual seed.go files.
func (c *TrelloConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "trello",
		Name:        "Trello",
		Description: "Trello integration for project management and kanban boards",
		Actions: []connectors.ManifestAction{
			{
				ActionType:  "trello.create_card",
				Name:        "Create Card",
				Description: "Create a new card in a Trello list",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["list_id", "name"],
					"properties": {
						"list_id": {
							"type": "string",
							"description": "ID of the list to create the card in (24-character hex string, e.g. 507f1f77bcf86cd799439011)"
						},
						"name": {
							"type": "string",
							"description": "Card title"
						},
						"desc": {
							"type": "string",
							"description": "Card description (supports Markdown)"
						},
						"pos": {
							"type": "string",
							"description": "Position of the card: \"top\", \"bottom\", or a positive number"
						},
						"due": {
							"type": "string",
							"description": "Due date in ISO 8601 format (e.g. 2026-12-31T00:00:00.000Z)"
						},
						"idMembers": {
							"type": "string",
							"description": "Comma-separated member IDs to assign"
						},
						"idLabels": {
							"type": "string",
							"description": "Comma-separated label IDs to apply"
						}
					}
				}`)),
			},
			{
				ActionType:  "trello.update_card",
				Name:        "Update Card",
				Description: "Update fields on an existing Trello card",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["card_id"],
					"properties": {
						"card_id": {
							"type": "string",
							"description": "ID of the card to update (24-character hex string)"
						},
						"name": {
							"type": "string",
							"description": "New card title"
						},
						"desc": {
							"type": "string",
							"description": "New card description (supports Markdown)"
						},
						"due": {
							"type": "string",
							"description": "Due date in ISO 8601 format"
						},
						"dueComplete": {
							"type": "boolean",
							"description": "Whether the due date is complete"
						},
						"idList": {
							"type": "string",
							"description": "ID of the list to move the card to"
						},
						"pos": {
							"type": "string",
							"description": "Position: \"top\", \"bottom\", or a positive number"
						},
						"idMembers": {
							"type": "string",
							"description": "Comma-separated member IDs"
						},
						"idLabels": {
							"type": "string",
							"description": "Comma-separated label IDs"
						},
						"closed": {
							"type": "boolean",
							"description": "Whether to archive the card"
						}
					}
				}`)),
			},
			{
				ActionType:  "trello.add_comment",
				Name:        "Add Comment",
				Description: "Add a comment to a Trello card",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["card_id", "text"],
					"properties": {
						"card_id": {
							"type": "string",
							"description": "ID of the card to comment on (24-character hex string)"
						},
						"text": {
							"type": "string",
							"description": "Comment text"
						}
					}
				}`)),
			},
			{
				ActionType:  "trello.move_card",
				Name:        "Move Card",
				Description: "Move a card to a different list — separate action for permission gating since moving represents a workflow state change",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["card_id", "list_id"],
					"properties": {
						"card_id": {
							"type": "string",
							"description": "ID of the card to move (24-character hex string)"
						},
						"list_id": {
							"type": "string",
							"description": "ID of the destination list (24-character hex string)"
						},
						"pos": {
							"type": "string",
							"description": "Position in the destination list: \"top\", \"bottom\", or a positive number"
						}
					}
				}`)),
			},
			{
				ActionType:  "trello.create_checklist",
				Name:        "Create Checklist",
				Description: "Create a checklist on a card with optional initial items",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["card_id", "name"],
					"properties": {
						"card_id": {
							"type": "string",
							"description": "ID of the card to add the checklist to (24-character hex string)"
						},
						"name": {
							"type": "string",
							"description": "Checklist name"
						},
						"items": {
							"type": "array",
							"items": {
								"type": "string"
							},
							"description": "List of checklist item names to add"
						}
					}
				}`)),
			},
			{
				ActionType:  "trello.search_cards",
				Name:        "Search Cards",
				Description: "Search cards across Trello boards",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["query"],
					"properties": {
						"query": {
							"type": "string",
							"description": "Search query string (supports Trello search operators like @member, #label, is:open)"
						},
						"board_id": {
							"type": "string",
							"description": "Filter results to a specific board (24-character hex string)"
						},
						"list_id": {
							"type": "string",
							"description": "Filter results to a specific list name or ID (appended as list: modifier)"
						},
						"members": {
							"type": "string",
							"description": "Filter by member username (appended as @member modifier)"
						},
						"due": {
							"type": "string",
							"description": "Filter by due date: \"day\", \"week\", \"month\", \"overdue\", \"notdue\", or \"incomplete\""
						},
						"limit": {
							"type": "integer",
							"default": 10,
							"description": "Max cards to return (1-1000)"
						}
					}
				}`)),
			},
		},
		RequiredCredentials: []connectors.ManifestCredential{
			{
				Service:         "trello",
				AuthType:        "api_key",
				InstructionsURL: "https://developer.atlassian.com/cloud/trello/guides/rest-api/api-introduction/#authentication-and-authorization",
			},
		},
		Templates: []connectors.ManifestTemplate{
			{
				ID:          "tpl_trello_create_card",
				ActionType:  "trello.create_card",
				Name:        "Create cards in a list",
				Description: "Locks the list and lets the agent choose card name and details.",
				Parameters:  json.RawMessage(`{"list_id":"your-list-id","name":"*","desc":"*","pos":"*","due":"*","idMembers":"*","idLabels":"*"}`),
			},
			{
				ID:          "tpl_trello_create_card_any",
				ActionType:  "trello.create_card",
				Name:        "Create cards freely",
				Description: "Agent can create cards in any list with any details.",
				Parameters:  json.RawMessage(`{"list_id":"*","name":"*","desc":"*","pos":"*","due":"*","idMembers":"*","idLabels":"*"}`),
			},
			{
				ID:          "tpl_trello_update_card",
				ActionType:  "trello.update_card",
				Name:        "Update any card",
				Description: "Agent can update any field on any card.",
				Parameters:  json.RawMessage(`{"card_id":"*","name":"*","desc":"*","due":"*","dueComplete":"*","idList":"*","pos":"*","idMembers":"*","idLabels":"*","closed":"*"}`),
			},
			{
				ID:          "tpl_trello_add_comment",
				ActionType:  "trello.add_comment",
				Name:        "Comment on any card",
				Description: "Agent can add comments to any card.",
				Parameters:  json.RawMessage(`{"card_id":"*","text":"*"}`),
			},
			{
				ID:          "tpl_trello_move_card_to_list",
				ActionType:  "trello.move_card",
				Name:        "Move cards to a specific list",
				Description: "Locks the destination list (e.g., Done). Agent picks the card.",
				Parameters:  json.RawMessage(`{"card_id":"*","list_id":"your-done-list-id","pos":"*"}`),
			},
			{
				ID:          "tpl_trello_move_card_any",
				ActionType:  "trello.move_card",
				Name:        "Move cards freely",
				Description: "Agent can move any card to any list.",
				Parameters:  json.RawMessage(`{"card_id":"*","list_id":"*","pos":"*"}`),
			},
			{
				ID:          "tpl_trello_create_checklist",
				ActionType:  "trello.create_checklist",
				Name:        "Create checklists",
				Description: "Agent can create checklists on any card.",
				Parameters:  json.RawMessage(`{"card_id":"*","name":"*","items":"*"}`),
			},
			{
				ID:          "tpl_trello_search_cards",
				ActionType:  "trello.search_cards",
				Name:        "Search cards",
				Description: "Agent can search for cards across boards.",
				Parameters:  json.RawMessage(`{"query":"*","board_id":"*","list_id":"*","members":"*","due":"*","limit":"*"}`),
			},
		},
	}
}
