package monday

import (
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// ValidateParams validates action parameters at approval request time.
// This runs as a fallback when the action does not implement RequestValidator,
// giving the agent an immediate error instead of failing after user approval.
func (c *MondayConnector) ValidateParams(actionType string, params json.RawMessage) error {
	return connectors.ValidateWithMap(actionType, params, paramValidators)
}

// paramValidators maps action types to their parameter validation functions.
var paramValidators = map[string]connectors.ParamValidatorFunc{
	"monday.add_update": makeParamValidator[addUpdateParams](),
	"monday.create_board": makeParamValidator[createBoardParams](),
	"monday.create_item": makeParamValidator[createItemParams](),
	"monday.create_subitem": makeParamValidator[createSubitemParams](),
	"monday.delete_item": makeParamValidator[deleteItemParams](),
	"monday.get_board": makeParamValidator[getBoardParams](),
	"monday.list_groups": makeParamValidator[listGroupsParams](),
	"monday.move_item_to_group": makeParamValidator[moveItemToGroupParams](),
	"monday.search_items": makeParamValidator[searchItemsParams](),
	"monday.update_item": makeParamValidator[updateItemParams](),
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
