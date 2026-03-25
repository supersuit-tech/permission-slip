package monday

import (
	_ "embed"
	"encoding/json"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// Manifest returns the connector's metadata manifest including action
// schemas, required credentials, and configuration templates.
//
//go:embed logo.svg
var logoSVG string

func (c *MondayConnector) Manifest() *connectors.ConnectorManifest {
	return &connectors.ConnectorManifest{
		ID:          "monday",
		Name:        "Monday.com",
		Description: "Monday.com integration for project management",
		LogoSVG:     logoSVG,
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
							"description": "The board ID to create the item on",
							"x-ui": {
								"label": "Board ID",
								"placeholder": "1234567890",
								"help_text": "Find in board URL or use monday.list_boards"
							}
						},
						"item_name": {
							"type": "string",
							"description": "Name of the new item",
							"x-ui": {
								"label": "Item Name",
								"placeholder": "New item name"
							}
						},
						"column_values": {
							"type": "object",
							"description": "JSON object mapping column IDs to values, e.g. {\"status\": {\"label\": \"Working on it\"}, \"date\": {\"date\": \"2024-01-15\"}}",
							"x-ui": {
								"label": "Column Values",
								"help_text": "JSON mapping column IDs to values. Example: {\"status\": {\"label\": \"Working on it\"}}. Use monday.get_board for column IDs."
							}
						},
						"group_id": {
							"type": "string",
							"description": "Group ID to create the item in (use the group's unique ID, not its display name)",
							"x-ui": {
								"label": "Group ID",
								"placeholder": "new_group",
								"help_text": "Use monday.list_groups to discover IDs"
							}
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
							"description": "The board ID containing the item",
							"x-ui": {
								"label": "Board ID",
								"placeholder": "1234567890",
								"help_text": "Find in board URL or use monday.list_boards"
							}
						},
						"item_id": {
							"type": "string",
							"description": "The item ID to update",
							"x-ui": {
								"label": "Item ID",
								"placeholder": "1234567890",
								"help_text": "Visible in item URL"
							}
						},
						"column_values": {
							"type": "object",
							"description": "JSON object mapping column IDs to new values",
							"x-ui": {
								"label": "Column Values",
								"help_text": "JSON mapping column IDs to values. Example: {\"status\": {\"label\": \"Working on it\"}}. Use monday.get_board for column IDs."
							}
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
							"description": "The item ID to add the update to",
							"x-ui": {
								"label": "Item ID",
								"placeholder": "1234567890",
								"help_text": "Visible in item URL"
							}
						},
						"body": {
							"type": "string",
							"description": "Update text content (supports HTML)",
							"x-ui": {
								"label": "Body",
								"placeholder": "Enter update text...",
								"widget": "textarea"
							}
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
							"description": "The parent item ID to create the subitem under",
							"x-ui": {
								"label": "Parent Item ID",
								"placeholder": "1234567890",
								"help_text": "Visible in item URL"
							}
						},
						"item_name": {
							"type": "string",
							"description": "Name of the new subitem",
							"x-ui": {
								"label": "Subitem Name",
								"placeholder": "New subitem name"
							}
						},
						"column_values": {
							"type": "object",
							"description": "JSON object mapping column IDs to values",
							"x-ui": {
								"label": "Column Values",
								"help_text": "JSON mapping column IDs to values. Example: {\"status\": {\"label\": \"Working on it\"}}. Use monday.get_board for column IDs."
							}
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
							"description": "The item ID to move",
							"x-ui": {
								"label": "Item ID",
								"placeholder": "1234567890",
								"help_text": "Visible in item URL"
							}
						},
						"group_id": {
							"type": "string",
							"description": "The target group ID (e.g. 'done', 'in_progress')",
							"x-ui": {
								"label": "Group ID",
								"placeholder": "new_group",
								"help_text": "Use monday.list_groups to discover IDs"
							}
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
							"description": "The board ID to search on",
							"x-ui": {
								"label": "Board ID",
								"placeholder": "1234567890",
								"help_text": "Find in board URL or use monday.list_boards"
							}
						},
						"query": {
							"type": "string",
							"description": "Text search query",
							"x-ui": {
								"label": "Search Query",
								"placeholder": "Search text..."
							}
						},
						"column_id": {
							"type": "string",
							"description": "Column ID to filter by (use with column_value)",
							"x-ui": {
								"label": "Column ID",
								"placeholder": "status",
								"help_text": "Internal column ID — use monday.get_board"
							}
						},
						"column_value": {
							"type": "string",
							"description": "Column value to filter by (use with column_id)",
							"x-ui": {
								"label": "Column Value",
								"placeholder": "Working on it"
							}
						},
						"limit": {
							"type": "integer",
							"default": 20,
							"description": "Maximum number of items to return (default 20)",
							"x-ui": {
								"label": "Max Results"
							}
						}
					}
				}`)),
			},
			{
				ActionType:  "monday.list_boards",
				Name:        "List Boards",
				Description: "List boards accessible to the authenticated user",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"properties": {
						"limit": {
							"type": "integer",
							"default": 20,
							"description": "Maximum number of boards to return (default 20)",
							"x-ui": {
								"label": "Max Results"
							}
						},
						"kind": {
							"type": "string",
							"enum": ["public", "private", "share"],
							"description": "Board kind filter",
							"x-ui": {
								"label": "Board Kind",
								"widget": "select"
							}
						}
					}
				}`)),
			},
			{
				ActionType:  "monday.get_board",
				Name:        "Get Board",
				Description: "Get board details including columns and groups",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["board_id"],
					"properties": {
						"board_id": {
							"type": "string",
							"description": "The board ID to retrieve",
							"x-ui": {
								"label": "Board ID",
								"placeholder": "1234567890",
								"help_text": "Find in board URL or use monday.list_boards"
							}
						}
					}
				}`)),
			},
			{
				ActionType:  "monday.create_board",
				Name:        "Create Board",
				Description: "Create a new Monday.com board",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["name"],
					"properties": {
						"name": {
							"type": "string",
							"description": "Board name",
							"x-ui": {
								"label": "Board Name",
								"placeholder": "My New Board"
							}
						},
						"kind": {
							"type": "string",
							"enum": ["public", "private", "share"],
							"description": "Board kind (default: public)",
							"x-ui": {
								"label": "Board Kind",
								"widget": "select"
							}
						},
						"folder_id": {
							"type": "string",
							"description": "Folder ID to create the board in",
							"x-ui": {
								"label": "Folder ID",
								"placeholder": "1234567890",
								"help_text": "Find the folder ID in the Monday.com URL when viewing the folder"
							}
						},
						"workspace_id": {
							"type": "string",
							"description": "Workspace ID to create the board in",
							"x-ui": {
								"label": "Workspace ID",
								"placeholder": "1234567890",
								"help_text": "Find the workspace ID in the Monday.com URL when viewing the workspace"
							}
						}
					}
				}`)),
			},
			{
				ActionType:  "monday.delete_item",
				Name:        "Delete Item",
				Description: "Permanently delete an item from a board",
				RiskLevel:   "high",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["item_id"],
					"properties": {
						"item_id": {
							"type": "string",
							"description": "The item ID to delete",
							"x-ui": {
								"label": "Item ID",
								"placeholder": "1234567890",
								"help_text": "Visible in item URL"
							}
						}
					}
				}`)),
			},
			{
				ActionType:  "monday.list_groups",
				Name:        "List Groups",
				Description: "List groups on a board (needed to identify group IDs for move_item_to_group)",
				RiskLevel:   "low",
				ParametersSchema: json.RawMessage(connectors.TrimIndent(`{
					"type": "object",
					"required": ["board_id"],
					"properties": {
						"board_id": {
							"type": "string",
							"description": "The board ID to list groups for",
							"x-ui": {
								"label": "Board ID",
								"placeholder": "1234567890",
								"help_text": "Find in board URL or use monday.list_boards"
							}
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
			{
				ID:          "tpl_monday_list_boards",
				ActionType:  "monday.list_boards",
				Name:        "List boards",
				Description: "Agent can list all boards accessible to the user.",
				Parameters:  json.RawMessage(`{"limit":20}`),
			},
			{
				ID:          "tpl_monday_get_board",
				ActionType:  "monday.get_board",
				Name:        "Get board details",
				Description: "Agent can retrieve full details (columns, groups) for any board.",
				Parameters:  json.RawMessage(`{"board_id":"*"}`),
			},
			{
				ID:          "tpl_monday_create_board",
				ActionType:  "monday.create_board",
				Name:        "Create boards",
				Description: "Agent can create new Monday.com boards.",
				Parameters:  json.RawMessage(`{"name":"*","kind":"public"}`),
			},
			{
				ID:          "tpl_monday_delete_item",
				ActionType:  "monday.delete_item",
				Name:        "Delete any item",
				Description: "Agent can permanently delete any item.",
				Parameters:  json.RawMessage(`{"item_id":"*"}`),
			},
			{
				ID:          "tpl_monday_list_groups",
				ActionType:  "monday.list_groups",
				Name:        "List groups on any board",
				Description: "Agent can list all groups on any board to find group IDs.",
				Parameters:  json.RawMessage(`{"board_id":"*"}`),
			},
		},
	}
}
