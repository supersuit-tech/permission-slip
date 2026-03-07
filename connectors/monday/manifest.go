package monday

import (
	"encoding/json"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// Manifest returns the connector's metadata manifest including action
// schemas, required credentials, and configuration templates.
func (c *MondayConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "monday",
		Name:        "Monday.com",
		Description: "Monday.com integration for project management",
		Actions: []connectors.ManifestAction{
			{
				ActionType:  "monday.create_item",
				Name:        "Create Item",
				Description: "Create a new item on a Monday.com board",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["board_id", "item_name"],
					"properties": {
						"board_id": {
							"type": "string",
							"description": "The board ID to create the item on"
						},
						"item_name": {
							"type": "string",
							"description": "Name of the new item"
						},
						"column_values": {
							"type": "object",
							"description": "JSON object mapping column IDs to values, e.g. {\"status\": {\"label\": \"Working on it\"}, \"date\": {\"date\": \"2024-01-15\"}}"
						},
						"group_id": {
							"type": "string",
							"description": "Group ID to create the item in (use the group's unique ID, not its display name)"
						}
					}
				}`)),
			},
			{
				ActionType:  "monday.update_item",
				Name:        "Update Item",
				Description: "Update column values on an existing Monday.com item",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["board_id", "item_id", "column_values"],
					"properties": {
						"board_id": {
							"type": "string",
							"description": "The board ID containing the item"
						},
						"item_id": {
							"type": "string",
							"description": "The item ID to update"
						},
						"column_values": {
							"type": "object",
							"description": "JSON object mapping column IDs to new values"
						}
					}
				}`)),
			},
			{
				ActionType:  "monday.add_update",
				Name:        "Add Update",
				Description: "Add an update (comment) to a Monday.com item",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["item_id", "body"],
					"properties": {
						"item_id": {
							"type": "string",
							"description": "The item ID to add the update to"
						},
						"body": {
							"type": "string",
							"description": "Update text content (supports HTML)"
						}
					}
				}`)),
			},
			{
				ActionType:  "monday.create_subitem",
				Name:        "Create Subitem",
				Description: "Create a subitem under an existing Monday.com item",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["parent_item_id", "item_name"],
					"properties": {
						"parent_item_id": {
							"type": "string",
							"description": "The parent item ID to create the subitem under"
						},
						"item_name": {
							"type": "string",
							"description": "Name of the new subitem"
						},
						"column_values": {
							"type": "object",
							"description": "JSON object mapping column IDs to values"
						}
					}
				}`)),
			},
			{
				ActionType:  "monday.move_item_to_group",
				Name:        "Move Item to Group",
				Description: "Move an item to a different group on its board",
				RiskLevel:   "medium",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["item_id", "group_id"],
					"properties": {
						"item_id": {
							"type": "string",
							"description": "The item ID to move"
						},
						"group_id": {
							"type": "string",
							"description": "The target group ID (e.g. 'done', 'in_progress')"
						}
					}
				}`)),
			},
			{
				ActionType:  "monday.search_items",
				Name:        "Search Items",
				Description: "Search and filter items on a Monday.com board",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["board_id"],
					"properties": {
						"board_id": {
							"type": "string",
							"description": "The board ID to search on"
						},
						"query": {
							"type": "string",
							"description": "Text search query"
						},
						"column_id": {
							"type": "string",
							"description": "Column ID to filter by (use with column_value)"
						},
						"column_value": {
							"type": "string",
							"description": "Column value to filter by (use with column_id)"
						},
						"limit": {
							"type": "integer",
							"default": 20,
							"description": "Maximum number of items to return (default 20)"
						}
					}
				}`)),
			},
		},
		RequiredCredentials: []connectors.ManifestCredential{
			{
				Service:         "monday",
				AuthType:        "api_key",
				InstructionsURL: "https://developer.monday.com/apps/docs/manage-access-tokens",
			},
		},
		Templates: []connectors.ManifestTemplate{
			{
				ID:          "tpl_monday_create_item_on_board",
				ActionType:  "monday.create_item",
				Name:        "Create items on a specific board",
				Description: "Replace the board_id with your board's numeric ID. Agent can create items with any name and column values.",
				Parameters:  json.RawMessage(`{"board_id":"1234567890","item_name":"*","column_values":"*","group_id":"*"}`),
			},
			{
				ID:          "tpl_monday_create_item_any",
				ActionType:  "monday.create_item",
				Name:        "Create items on any board",
				Description: "Agent can create items on any board with any values.",
				Parameters:  json.RawMessage(`{"board_id":"*","item_name":"*","column_values":"*","group_id":"*"}`),
			},
			{
				ID:          "tpl_monday_update_item",
				ActionType:  "monday.update_item",
				Name:        "Update items on a specific board",
				Description: "Replace the board_id with your board's numeric ID. Agent can update column values on any item in that board.",
				Parameters:  json.RawMessage(`{"board_id":"1234567890","item_id":"*","column_values":"*"}`),
			},
			{
				ID:          "tpl_monday_add_update",
				ActionType:  "monday.add_update",
				Name:        "Add updates to items",
				Description: "Agent can add comments and updates to any item.",
				Parameters:  json.RawMessage(`{"item_id":"*","body":"*"}`),
			},
			{
				ID:          "tpl_monday_create_subitem",
				ActionType:  "monday.create_subitem",
				Name:        "Create subitems",
				Description: "Agent can create subitems under any item.",
				Parameters:  json.RawMessage(`{"parent_item_id":"*","item_name":"*","column_values":"*"}`),
			},
			{
				ID:          "tpl_monday_move_to_group",
				ActionType:  "monday.move_item_to_group",
				Name:        "Move items between groups",
				Description: "Agent can move items to any group (e.g. status changes like moving to 'Done').",
				Parameters:  json.RawMessage(`{"item_id":"*","group_id":"*"}`),
			},
			{
				ID:          "tpl_monday_search_items",
				ActionType:  "monday.search_items",
				Name:        "Search items on any board",
				Description: "Agent can search and filter items. Use query for text search or column_id+column_value for filtering.",
				Parameters:  json.RawMessage(`{"board_id":"*","query":"*","column_id":"*","column_value":"*","limit":20}`),
			},
		},
	}
}
