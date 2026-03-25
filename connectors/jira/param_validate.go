package jira

import (
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// ValidateParams validates action parameters at approval request time.
// This runs as a fallback when the action does not implement RequestValidator,
// giving the agent an immediate error instead of failing after user approval.
func (c *JiraConnector) ValidateParams(actionType string, params json.RawMessage) error {
	return connectors.ValidateWithMap(actionType, params, paramValidators)
}

// paramValidators maps action types to their parameter validation functions.
var paramValidators = map[string]connectors.ParamValidatorFunc{
	"jira.add_comment": makeParamValidator[addCommentParams](),
	"jira.assign_issue": makeParamValidator[assignIssueParams](),
	"jira.create_issue": makeParamValidator[createIssueParams](),
	"jira.create_sprint": makeParamValidator[createSprintParams](),
	"jira.delete_issue": makeParamValidator[deleteIssueParams](),
	"jira.get_issue": makeParamValidator[getIssueParams](),
	"jira.list_sprints": makeParamValidator[listSprintsParams](),
	"jira.move_to_sprint": makeParamValidator[moveToSprintParams](),
	"jira.search": makeParamValidator[searchParams](),
	"jira.transition_issue": makeParamValidator[transitionIssueParams](),
	"jira.update_issue": makeParamValidator[updateIssueParams](),
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
