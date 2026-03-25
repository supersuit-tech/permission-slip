package plaid

import (
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// ValidateParams validates action parameters at approval request time.
// This runs as a fallback when the action does not implement RequestValidator,
// giving the agent an immediate error instead of failing after user approval.
func (c *PlaidConnector) ValidateParams(actionType string, params json.RawMessage) error {
	return connectors.ValidateWithMap(actionType, params, paramValidators)
}

// paramValidators maps action types to their parameter validation functions.
var paramValidators = map[string]connectors.ParamValidatorFunc{
	"plaid.create_link_token": makeParamValidator[createLinkTokenParams](),
	"plaid.get_accounts": makeParamValidator[accessTokenParams](),
	"plaid.get_balances": makeParamValidator[accessTokenParams](),
	"plaid.get_identity": makeParamValidator[accessTokenParams](),
	"plaid.get_institution": makeParamValidator[getInstitutionParams](),
	"plaid.list_transactions": makeParamValidator[listTransactionsParams](),
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
