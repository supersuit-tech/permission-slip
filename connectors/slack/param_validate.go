package slack

import (
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// ValidateParams validates action parameters at approval request time.
// This runs as a fallback when the action does not implement RequestValidator,
// giving the agent an immediate error instead of failing after user approval.
func (c *SlackConnector) ValidateParams(actionType string, params json.RawMessage) error {
	return connectors.ValidateWithMap(actionType, params, paramValidators)
}

// paramValidators maps action types to their parameter validation functions.
// This intentionally includes ALL actions, even those that already implement
// RequestValidator. The approval handler prefers RequestValidator when present,
// so these entries are only reached if an action's RequestValidator is removed.
// Keeping the full table avoids silent validation gaps during refactoring.
var paramValidators = map[string]connectors.ParamValidatorFunc{
	"slack.add_reaction": makeParamValidator[addReactionParams](),
	"slack.create_channel": makeParamValidator[createChannelParams](),
	"slack.delete_message": makeParamValidator[deleteMessageParams](),
	"slack.invite_to_channel": makeParamValidator[inviteToChannelParams](),
	"slack.list_channels": makeParamValidator[listChannelsParams](),
	"slack.list_users": makeParamValidator[listUsersParams](),
	"slack.read_channel_messages": makeParamValidator[readChannelMessagesParams](),
	"slack.read_thread": makeParamValidator[readThreadParams](),
	"slack.schedule_message": makeParamValidator[scheduleMessageParams](),
	"slack.search_messages": makeParamValidator[searchMessagesParams](),
	"slack.send_dm": makeParamValidator[sendDMParams](),
	"slack.send_message": makeParamValidator[sendMessageParams](),
	"slack.set_topic": makeParamValidator[setTopicParams](),
	"slack.update_message": makeParamValidator[updateMessageParams](),
	"slack.upload_file": makeParamValidator[uploadFileParams](),
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
