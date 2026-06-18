package imessage

import (
	"encoding/json"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// ValidateParams validates action parameters at approval request time.
func (c *IMessageConnector) ValidateParams(actionType string, params json.RawMessage) error {
	return connectors.ValidateWithMap(actionType, params, paramValidators)
}

var paramValidators = map[string]connectors.ParamValidatorFunc{
	"imessage.list_chats":   makeParamValidator[listChatsParams](),
	"imessage.get_chat":     makeParamValidator[getChatParams](),
	"imessage.read_history": makeParamValidator[readHistoryParams](),
	"imessage.search":       makeParamValidator[searchParams](),
	"imessage.send_message": makeParamValidator[sendMessageParams](),
}

func makeParamValidator[T any, PT interface {
	*T
	validate() error
}]() connectors.ParamValidatorFunc {
	return func(params json.RawMessage) error {
		p := PT(new(T))
		if err := json.Unmarshal(params, p); err != nil {
			return &connectors.ValidationError{Message: "invalid parameters: " + err.Error()}
		}
		return p.validate()
	}
}

var _ connectors.ParamValidator = (*IMessageConnector)(nil)
