package twilio

import (
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// ValidateParams validates action parameters at approval request time.
// This runs as a fallback when the action does not implement RequestValidator,
// giving the agent an immediate error instead of failing after user approval.
func (c *TwilioConnector) ValidateParams(actionType string, params json.RawMessage) error {
	return connectors.ValidateWithMap(actionType, params, paramValidators)
}

// paramValidators maps action types to their parameter validation functions.
var paramValidators = map[string]connectors.ParamValidatorFunc{
	"twilio.get_call": makeParamValidator[getCallParams](),
	"twilio.get_message": makeParamValidator[getMessageParams](),
	"twilio.initiate_call": makeParamValidator[initiateCallParams](),
	"twilio.lookup_phone": makeParamValidator[lookupPhoneParams](),
	"twilio.send_sms": makeParamValidator[sendSMSParams](),
	"twilio.send_whatsapp": makeParamValidator[sendWhatsAppParams](),
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
