package discord

import (
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// ValidateParams validates action parameters at approval request time.
// This runs as a fallback when the action does not implement RequestValidator,
// giving the agent an immediate error instead of failing after user approval.
func (c *DiscordConnector) ValidateParams(actionType string, params json.RawMessage) error {
	return connectors.ValidateWithMap(actionType, params, paramValidators)
}

// paramValidators maps action types to their parameter validation functions.
var paramValidators = map[string]connectors.ParamValidatorFunc{
	"discord.ban_user": makeParamValidator[banUserParams](),
	"discord.create_channel": makeParamValidator[createChannelParams](),
	"discord.create_event": makeParamValidator[createEventParams](),
	"discord.create_thread": makeParamValidator[createThreadParams](),
	"discord.kick_user": makeParamValidator[kickUserParams](),
	"discord.list_members": makeParamValidator[listMembersParams](),
	"discord.manage_roles": makeParamValidator[manageRolesParams](),
	"discord.send_message": makeParamValidator[sendMessageParams](),
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
