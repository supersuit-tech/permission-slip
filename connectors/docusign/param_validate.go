package docusign

import (
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// ValidateParams validates action parameters at approval request time.
// This runs as a fallback when the action does not implement RequestValidator,
// giving the agent an immediate error instead of failing after user approval.
func (c *DocuSignConnector) ValidateParams(actionType string, params json.RawMessage) error {
	return connectors.ValidateWithMap(actionType, params, paramValidators)
}

// paramValidators maps action types to their parameter validation functions.
var paramValidators = map[string]connectors.ParamValidatorFunc{
	"docusign.check_status": makeParamValidator[checkStatusParams](),
	"docusign.create_envelope": makeParamValidator[createEnvelopeParams](),
	"docusign.download_signed": makeParamValidator[downloadSignedParams](),
	"docusign.list_templates": makeParamValidator[listTemplatesParams](),
	"docusign.send_envelope": makeParamValidator[sendEnvelopeParams](),
	"docusign.update_recipients": makeParamValidator[updateRecipientsParams](),
	"docusign.void_envelope": makeParamValidator[voidEnvelopeParams](),
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
