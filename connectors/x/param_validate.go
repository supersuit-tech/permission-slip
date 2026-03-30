package x

import (
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// ValidateParams validates action parameters at approval request time.
// This runs as a fallback when the action does not implement RequestValidator,
// giving the agent an immediate error instead of failing after user approval.
func (c *XConnector) ValidateParams(actionType string, params json.RawMessage) error {
	return connectors.ValidateWithMap(actionType, params, paramValidators)
}

// paramValidators maps action types to their parameter validation functions.
var paramValidators = map[string]connectors.ParamValidatorFunc{
	"x.delete_tweet": makeParamValidator[deleteTweetParams](),
	"x.follow_user": makeParamValidator[userFollowParams](),
	"x.get_followers": makeParamValidator[userListParams](),
	"x.get_following": makeParamValidator[userListParams](),
	"x.get_user_tweets": makeParamValidator[getUserTweetsParams](),
	"x.like_tweet": makeParamValidator[userTweetParams](),
	"x.post_tweet": makeParamValidator[postTweetParams](),
	"x.retweet": makeParamValidator[userTweetParams](),
	"x.search_tweets": makeParamValidator[searchTweetsParams](),
	"x.send_dm": makeParamValidator[sendDMParams](),
	"x.unfollow_user": makeParamValidator[userFollowParams](),
	"x.unlike_tweet": makeParamValidator[userTweetParams](),
	"x.unretweet": makeParamValidator[userTweetParams](),
	"x.upload_media": makeParamValidator[uploadMediaParams](),
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
