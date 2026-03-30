package trello

import (
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// ValidateParams validates action parameters at approval request time.
// This runs as a fallback when the action does not implement RequestValidator,
// giving the agent an immediate error instead of failing after user approval.
func (c *TrelloConnector) ValidateParams(actionType string, params json.RawMessage) error {
	return connectors.ValidateWithMap(actionType, params, paramValidators)
}

// paramValidators maps action types to their parameter validation functions.
var paramValidators = map[string]connectors.ParamValidatorFunc{
	"trello.add_comment": makeParamValidator[addCommentParams](),
	"trello.add_label": makeParamValidator[addLabelParams](),
	"trello.add_member": makeParamValidator[addMemberParams](),
	"trello.create_board": makeParamValidator[createBoardParams](),
	"trello.create_card": makeParamValidator[createCardParams](),
	"trello.create_checklist": makeParamValidator[createChecklistParams](),
	"trello.create_list": makeParamValidator[createListParams](),
	"trello.delete_card": makeParamValidator[deleteCardParams](),
	"trello.list_labels": makeParamValidator[listLabelsParams](),
	"trello.list_lists": makeParamValidator[listListsParams](),
	"trello.move_card": makeParamValidator[moveCardParams](),
	"trello.search_cards": makeParamValidator[searchCardsParams](),
	"trello.update_card": makeParamValidator[updateCardParams](),
}

func makeParamValidator[T any, PT interface {
	*T
	validate() error
}]() connectors.ParamValidatorFunc {
	return func(params json.RawMessage) error {
		p := PT(new(T))
		if err := json.Unmarshal(params, p); err != nil {
			return &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
		}
		return p.validate()
	}
}
