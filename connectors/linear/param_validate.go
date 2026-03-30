package linear

import (
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// ValidateParams validates action parameters at approval request time.
// This runs as a fallback when the action does not implement RequestValidator,
// giving the agent an immediate error instead of failing after user approval.
func (c *LinearConnector) ValidateParams(actionType string, params json.RawMessage) error {
	return connectors.ValidateWithMap(actionType, params, paramValidators)
}

// paramValidators maps action types to their parameter validation functions.
var paramValidators = map[string]connectors.ParamValidatorFunc{
	"linear.add_comment": makeParamValidator[addCommentParams](),
	"linear.add_label": makeParamValidator[addLabelParams](),
	"linear.assign_issue": makeParamValidator[assignIssueParams](),
	"linear.change_state": makeParamValidator[changeStateParams](),
	"linear.create_issue": makeParamValidator[createIssueParams](),
	"linear.create_project": makeParamValidator[createProjectParams](),
	"linear.get_issue": makeParamValidator[getIssueParams](),
	"linear.list_cycles": makeParamValidator[listCyclesParams](),
	"linear.search_issues": makeParamValidator[searchIssuesParams](),
	"linear.update_issue": makeParamValidator[updateIssueParams](),
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
