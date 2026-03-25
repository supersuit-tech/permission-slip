package zendesk

import (
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// ValidateParams validates action parameters at approval request time.
// This runs as a fallback when the action does not implement RequestValidator,
// giving the agent an immediate error instead of failing after user approval.
func (c *ZendeskConnector) ValidateParams(actionType string, params json.RawMessage) error {
	return connectors.ValidateWithMap(actionType, params, paramValidators)
}

// paramValidators maps action types to their parameter validation functions.
var paramValidators = map[string]connectors.ParamValidatorFunc{
	"zendesk.assign_ticket": makeParamValidator[assignTicketParams](),
	"zendesk.create_ticket": makeParamValidator[createTicketParams](),
	"zendesk.create_user": makeParamValidator[createUserParams](),
	"zendesk.get_satisfaction_ratings": makeParamValidator[getSatisfactionRatingsParams](),
	"zendesk.get_user": makeParamValidator[getUserParams](),
	"zendesk.list_tags": makeParamValidator[listTagsParams](),
	"zendesk.merge_tickets": makeParamValidator[mergeTicketsParams](),
	"zendesk.reply_ticket": makeParamValidator[replyTicketParams](),
	"zendesk.search_tickets": makeParamValidator[searchTicketsParams](),
	"zendesk.update_tags": makeParamValidator[updateTagsParams](),
	"zendesk.update_ticket": makeParamValidator[updateTicketParams](),
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
