package linkedin

import (
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// ValidateParams validates action parameters at approval request time.
// This runs as a fallback when the action does not implement RequestValidator,
// giving the agent an immediate error instead of failing after user approval.
func (c *LinkedInConnector) ValidateParams(actionType string, params json.RawMessage) error {
	return connectors.ValidateWithMap(actionType, params, paramValidators)
}

// paramValidators maps action types to their parameter validation functions.
var paramValidators = map[string]connectors.ParamValidatorFunc{
	"linkedin.add_comment": makeParamValidator[addCommentParams](),
	"linkedin.create_company_post": makeParamValidator[createCompanyPostParams](),
	"linkedin.create_post": makeParamValidator[createPostParams](),
	"linkedin.delete_post": makeParamValidator[deletePostParams](),
	"linkedin.get_company": makeParamValidator[getCompanyParams](),
	"linkedin.get_post_analytics": makeParamValidator[getPostAnalyticsParams](),
	"linkedin.list_connections": makeParamValidator[listConnectionsParams](),
	"linkedin.search_companies": makeParamValidator[searchCompaniesParams](),
	"linkedin.search_people": makeParamValidator[searchPeopleParams](),
	"linkedin.send_message": makeParamValidator[sendMessageParams](),
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
