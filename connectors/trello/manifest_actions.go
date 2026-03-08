package trello

import (
	"encoding/json"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// trelloActions returns the JSON Schema definitions for all Trello actions.
// Each entry describes the parameters an agent may supply for that action,
// validated against the schema before execution.
func trelloActions() []connectors.ManifestAction {
	return []connectors.ManifestAction{
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
		{
			ActionType:  "trello.list_boards",
			Name:        "List Boards",
			Description: "List boards accessible to the authenticated user",
			RiskLevel:   "low",
			ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
				"type": "object",
				"properties": {
					"filter": {
						"type": "string",
						"enum": ["open", "closed", "starred", "members", "all"],
						"description": "Filter boards (default: open)"
					}
				}
			}`)),
		},
		{
			ActionType:  "trello.create_board",
			Name:        "Create Board",
			Description: "Create a new Trello board",
			RiskLevel:   "low",
			ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
				"type": "object",
				"required": ["name"],
				"properties": {
					"name": {
						"type": "string",
						"description": "Board name"
					},
					"desc": {
						"type": "string",
						"description": "Board description"
					},
					"defaultLists": {
						"type": "boolean",
						"description": "Whether to create default lists (To Do, Doing, Done). Defaults to true"
					},
					"idOrganization": {
						"type": "string",
						"description": "Organization or workspace ID to create the board in"
					},
					"prefs_permissionLevel": {
						"type": "string",
						"description": "Permission level: \"private\", \"org\", or \"public\""
					}
				}
			}`)),
		},
		{
			ActionType:  "trello.list_lists",
			Name:        "List Lists",
			Description: "List lists on a board",
			RiskLevel:   "low",
			ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
				"type": "object",
				"required": ["board_id"],
				"properties": {
					"board_id": {
						"type": "string",
						"description": "Board ID to list lists for (24-character hex string)"
					},
					"filter": {
						"type": "string",
						"enum": ["open", "closed", "all"],
						"description": "Filter lists (default: open)"
					}
				}
			}`)),
		},
		{
			ActionType:  "trello.create_list",
			Name:        "Create List",
			Description: "Create a new list on a board",
			RiskLevel:   "low",
			ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
				"type": "object",
				"required": ["board_id", "name"],
				"properties": {
					"board_id": {
						"type": "string",
						"description": "Board ID to create the list on (24-character hex string)"
					},
					"name": {
						"type": "string",
						"description": "List name"
					},
					"pos": {
						"type": "string",
						"description": "Position: \"top\", \"bottom\", or a positive number"
					}
				}
			}`)),
		},
		{
			ActionType:  "trello.delete_card",
			Name:        "Delete Card",
			Description: "Permanently delete a card",
			RiskLevel:   "high",
			ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
				"type": "object",
				"required": ["card_id"],
				"properties": {
					"card_id": {
						"type": "string",
						"description": "ID of the card to delete (24-character hex string)"
					}
				}
			}`)),
		},
		{
			ActionType:  "trello.list_labels",
			Name:        "List Labels",
			Description: "List available labels on a board",
			RiskLevel:   "low",
			ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
				"type": "object",
				"required": ["board_id"],
				"properties": {
					"board_id": {
						"type": "string",
						"description": "Board ID to list labels for (24-character hex string)"
					}
				}
			}`)),
		},
		{
			ActionType:  "trello.add_label",
			Name:        "Add Label",
			Description: "Add a label to a card",
			RiskLevel:   "low",
			ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
				"type": "object",
				"required": ["card_id", "label_id"],
				"properties": {
					"card_id": {
						"type": "string",
						"description": "ID of the card to add the label to (24-character hex string)"
					},
					"label_id": {
						"type": "string",
						"description": "ID of the label to add (24-character hex string)"
					}
				}
			}`)),
		},
		{
			ActionType:  "trello.add_member",
			Name:        "Add Member",
			Description: "Add a member to a card",
			RiskLevel:   "low",
			ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
				"type": "object",
				"required": ["card_id", "member_id"],
				"properties": {
					"card_id": {
						"type": "string",
						"description": "ID of the card (24-character hex string)"
					},
					"member_id": {
						"type": "string",
						"description": "ID of the member to add (24-character hex string)"
					}
				}
			}`)),
		},
	}
}
