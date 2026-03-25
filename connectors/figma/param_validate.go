package figma

import (
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// ValidateParams validates action parameters at approval request time.
// This runs as a fallback when the action does not implement RequestValidator,
// giving the agent an immediate error instead of failing after user approval.
func (c *FigmaConnector) ValidateParams(actionType string, params json.RawMessage) error {
	return connectors.ValidateWithMap(actionType, params, paramValidators)
}

// paramValidators maps action types to their parameter validation functions.
var paramValidators = map[string]connectors.ParamValidatorFunc{
	"figma.export_images": makeParamValidator[exportImagesParams](),
	"figma.get_components": makeParamValidator[getComponentsParams](),
	"figma.get_file": makeParamValidator[getFileParams](),
	"figma.get_styles": makeParamValidator[getStylesParams](),
	"figma.get_variables": makeParamValidator[getVariablesParams](),
	"figma.get_versions": makeParamValidator[getVersionsParams](),
	"figma.list_comments": makeParamValidator[listCommentsParams](),
	"figma.list_files": makeParamValidator[listFilesParams](),
	"figma.list_projects": makeParamValidator[listProjectsParams](),
	"figma.post_comment": makeParamValidator[postCommentParams](),
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
