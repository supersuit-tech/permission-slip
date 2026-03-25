package salesforce

import (
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// ValidateParams validates action parameters at approval request time.
// This runs as a fallback when the action does not implement RequestValidator,
// giving the agent an immediate error instead of failing after user approval.
func (c *SalesforceConnector) ValidateParams(actionType string, params json.RawMessage) error {
	return connectors.ValidateWithMap(actionType, params, paramValidators)
}

// paramValidators maps action types to their parameter validation functions.
var paramValidators = map[string]connectors.ParamValidatorFunc{
	"salesforce.add_note": makeParamValidator[addNoteParams](),
	"salesforce.convert_lead": makeParamValidator[convertLeadParams](),
	"salesforce.create_lead": makeParamValidator[createLeadParams](),
	"salesforce.create_opportunity": makeParamValidator[createOpportunityParams](),
	"salesforce.create_record": makeParamValidator[createRecordParams](),
	"salesforce.create_task": makeParamValidator[createTaskParams](),
	"salesforce.delete_record": makeParamValidator[deleteRecordParams](),
	"salesforce.describe_object": makeParamValidator[describeObjectParams](),
	"salesforce.query": makeParamValidator[queryParams](),
	"salesforce.run_report": makeParamValidator[runReportParams](),
	"salesforce.update_opportunity": makeParamValidator[updateOpportunityParams](),
	"salesforce.update_record": makeParamValidator[updateRecordParams](),
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
