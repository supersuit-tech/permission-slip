package trello

import (
	"encoding/json"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// trelloTemplates returns the pre-built permission templates for common Trello
// use cases. Templates let workspace admins grant scoped access to agents
// without exposing full parameter freedom.
func intPtr(n int) *int {
	return &n
}

func trelloTemplates() []connectors.ManifestTemplate {
	saRead30d := &connectors.ManifestStandingApproval{DurationDays: intPtr(30)}
	return []connectors.ManifestTemplate{
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
			ID:               "tpl_trello_search_cards",
			ActionType:       "trello.search_cards",
			Name:             "Search cards",
			Description:      "Agent can search for cards across boards.",
			Parameters:       json.RawMessage(`{"query":"*","board_id":"*","list_id":"*","members":"*","due":"*","limit":"*"}`),
			StandingApproval: saRead30d,
		},
		{
			ID:               "tpl_trello_list_boards",
			ActionType:       "trello.list_boards",
			Name:             "List boards",
			Description:      "Agent can list all boards accessible to the user.",
			Parameters:       json.RawMessage(`{"filter":"open"}`),
			StandingApproval: saRead30d,
		},
		{
			ID:          "tpl_trello_create_board",
			ActionType:  "trello.create_board",
			Name:        "Create boards",
			Description: "Agent can create new Trello boards.",
			Parameters:  json.RawMessage(`{"name":"*","desc":"*","defaultLists":true}`),
		},
		{
			ID:               "tpl_trello_list_lists",
			ActionType:       "trello.list_lists",
			Name:             "List lists on any board",
			Description:      "Agent can list all lists on any board.",
			Parameters:       json.RawMessage(`{"board_id":"*","filter":"open"}`),
			StandingApproval: saRead30d,
		},
		{
			ID:          "tpl_trello_create_list",
			ActionType:  "trello.create_list",
			Name:        "Create lists on boards",
			Description: "Agent can create new lists on any board.",
			Parameters:  json.RawMessage(`{"board_id":"*","name":"*","pos":"*"}`),
		},
		{
			ID:          "tpl_trello_delete_card",
			ActionType:  "trello.delete_card",
			Name:        "Delete any card",
			Description: "Agent can permanently delete any card.",
			Parameters:  json.RawMessage(`{"card_id":"*"}`),
		},
		{
			ID:               "tpl_trello_list_labels",
			ActionType:       "trello.list_labels",
			Name:             "List labels on a board",
			Description:      "Agent can list all labels available on any board.",
			Parameters:       json.RawMessage(`{"board_id":"*"}`),
			StandingApproval: saRead30d,
		},
		{
			ID:          "tpl_trello_add_label",
			ActionType:  "trello.add_label",
			Name:        "Add labels to cards",
			Description: "Agent can add any label to any card.",
			Parameters:  json.RawMessage(`{"card_id":"*","label_id":"*"}`),
		},
		{
			ID:          "tpl_trello_add_member",
			ActionType:  "trello.add_member",
			Name:        "Add members to cards",
			Description: "Agent can add any member to any card.",
			Parameters:  json.RawMessage(`{"card_id":"*","member_id":"*"}`),
		},
	}
}
