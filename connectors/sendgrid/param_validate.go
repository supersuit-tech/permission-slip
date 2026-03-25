package sendgrid

import (
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// ValidateParams validates action parameters at approval request time.
// This runs as a fallback when the action does not implement RequestValidator,
// giving the agent an immediate error instead of failing after user approval.
func (c *SendGridConnector) ValidateParams(actionType string, params json.RawMessage) error {
	return connectors.ValidateWithMap(actionType, params, paramValidators)
}

// paramValidators maps action types to their parameter validation functions.
var paramValidators = map[string]connectors.ParamValidatorFunc{
	"sendgrid.add_to_list": makeParamValidator[addToListParams](),
	"sendgrid.create_contact": makeParamValidator[createContactParams](),
	"sendgrid.create_template": makeParamValidator[createTemplateParams](),
	"sendgrid.get_bounces": makeParamValidator[getBouncesParams](),
	"sendgrid.get_campaign_stats": makeParamValidator[getCampaignStatsParams](),
	"sendgrid.get_suppressions": makeParamValidator[getSuppressionsParams](),
	"sendgrid.remove_from_list": makeParamValidator[removeFromListParams](),
	"sendgrid.schedule_campaign": makeParamValidator[scheduleCampaignParams](),
	"sendgrid.send_transactional_email": makeParamValidator[sendTransactionalEmailParams](),
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
