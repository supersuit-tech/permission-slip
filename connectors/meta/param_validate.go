package meta

import (
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// ValidateParams validates action parameters at approval request time.
// This runs as a fallback when the action does not implement RequestValidator,
// giving the agent an immediate error instead of failing after user approval.
func (c *MetaConnector) ValidateParams(actionType string, params json.RawMessage) error {
	return connectors.ValidateWithMap(actionType, params, paramValidators)
}

// paramValidators maps action types to their parameter validation functions.
var paramValidators = map[string]connectors.ParamValidatorFunc{
	"meta.create_ad": makeParamValidator[createAdParams](),
	"meta.create_ad_campaign": makeParamValidator[createAdCampaignParams](),
	"meta.create_instagram_post": makeParamValidator[createInstagramPostParams](),
	"meta.create_instagram_story": makeParamValidator[createInstagramStoryParams](),
	"meta.create_page_post": makeParamValidator[createPagePostParams](),
	"meta.delete_page_post": makeParamValidator[deletePagePostParams](),
	"meta.get_instagram_insights": makeParamValidator[getInstagramInsightsParams](),
	"meta.get_page_insights": makeParamValidator[getPageInsightsParams](),
	"meta.list_instagram_posts": makeParamValidator[listInstagramPostsParams](),
	"meta.list_page_posts": makeParamValidator[listPagePostsParams](),
	"meta.reply_instagram_comment": makeParamValidator[replyInstagramCommentParams](),
	"meta.reply_page_comment": makeParamValidator[replyPageCommentParams](),
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
