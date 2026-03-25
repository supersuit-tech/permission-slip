package asana

import (
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// ValidateParams validates action parameters at approval request time.
// This runs as a fallback when the action does not implement RequestValidator,
// giving the agent an immediate error instead of failing after user approval.
func (c *AsanaConnector) ValidateParams(actionType string, params json.RawMessage) error {
	return connectors.ValidateWithMap(actionType, params, paramValidators)
}

// paramValidators maps action types to their parameter validation functions.
var paramValidators = map[string]connectors.ParamValidatorFunc{
	"asana.add_comment": makeParamValidator[addCommentParams](),
	"asana.complete_task": makeParamValidator[completeTaskParams](),
	"asana.create_project": makeParamValidator[createProjectParams](),
	"asana.create_section": makeParamValidator[createSectionParams](),
	"asana.create_subtask": makeParamValidator[createSubtaskParams](),
	"asana.create_task": makeParamValidator[createTaskParams](),
	"asana.delete_task": makeParamValidator[deleteTaskParams](),
	"asana.list_projects": makeParamValidator[listProjectsParams](),
	"asana.list_sections": makeParamValidator[listSectionsParams](),
	"asana.search_tasks": makeParamValidator[searchTasksParams](),
	"asana.update_task": makeParamValidator[updateTaskParams](),
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
