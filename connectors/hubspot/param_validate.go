package hubspot

import (
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// ValidateParams validates action parameters at approval request time.
// This runs as a fallback when the action does not implement RequestValidator,
// giving the agent an immediate error instead of failing after user approval.
func (c *HubSpotConnector) ValidateParams(actionType string, params json.RawMessage) error {
	return connectors.ValidateWithMap(actionType, params, paramValidators)
}

// paramValidators maps action types to their parameter validation functions.
var paramValidators = map[string]connectors.ParamValidatorFunc{
	"hubspot.add_note": makeParamValidator[addNoteParams](),
	"hubspot.create_company": makeParamValidator[createCompanyParams](),
	"hubspot.create_contact": makeParamValidator[createContactParams](),
	"hubspot.create_deal": makeParamValidator[createDealParams](),
	"hubspot.create_email_campaign": makeParamValidator[createEmailCampaignParams](),
	"hubspot.create_ticket": makeParamValidator[createTicketParams](),
	"hubspot.delete_contact": makeParamValidator[deleteContactParams](),
	"hubspot.delete_deal": makeParamValidator[deleteDealParams](),
	"hubspot.enroll_in_workflow": makeParamValidator[enrollInWorkflowParams](),
	"hubspot.get_analytics": makeParamValidator[getAnalyticsParams](),
	"hubspot.get_contact": makeParamValidator[getContactParams](),
	"hubspot.list_deals": makeParamValidator[listDealsParams](),
	"hubspot.search": makeParamValidator[searchParams](),
	"hubspot.update_company": makeParamValidator[updateCompanyParams](),
	"hubspot.update_contact": makeParamValidator[updateContactParams](),
	"hubspot.update_deal_stage": makeParamValidator[updateDealStageParams](),
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
