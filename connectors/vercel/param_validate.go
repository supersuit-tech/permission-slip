package vercel

import (
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// ValidateParams validates action parameters at approval request time.
// This runs as a fallback when the action does not implement RequestValidator,
// giving the agent an immediate error instead of failing after user approval.
func (c *VercelConnector) ValidateParams(actionType string, params json.RawMessage) error {
	return connectors.ValidateWithMap(actionType, params, paramValidators)
}

// paramValidators maps action types to their parameter validation functions.
var paramValidators = map[string]connectors.ParamValidatorFunc{
	"vercel.delete_env_var": makeParamValidator[deleteEnvVarParams](),
	"vercel.get_deployment": makeParamValidator[getDeploymentParams](),
	"vercel.list_env_vars": makeParamValidator[listEnvVarsParams](),
	"vercel.promote_deployment": makeParamValidator[promoteDeploymentParams](),
	"vercel.rollback_deployment": makeParamValidator[rollbackDeploymentParams](),
	"vercel.set_env_var": makeParamValidator[setEnvVarParams](),
	"vercel.trigger_deployment": makeParamValidator[triggerDeploymentParams](),
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
