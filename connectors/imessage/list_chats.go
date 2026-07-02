package imessage

import (
	"context"
	"encoding/json"

	"github.com/supersuit-tech/permission-slip/connectors"
)

type listChatsAction struct {
	conn *IMessageConnector
}

type listChatsParams struct {
	Limit int `json:"limit"`
}

func (p *listChatsParams) validate() error {
	if p.Limit <= 0 {
		p.Limit = 20
	}
	if p.Limit > 100 {
		return &connectors.ValidationError{Message: "limit must be at most 100"}
	}
	return nil
}

func (a *listChatsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params listChatsParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: "invalid parameters: " + err.Error()}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	actionCtx, cancel := a.conn.actionTimeout(ctx)
	defer cancel()

	var result chatsListResult
	if err := a.conn.client.rpcCall(actionCtx, req.Credentials, "chats.list", map[string]any{
		"limit": params.Limit,
	}, &result); err != nil {
		return nil, err
	}
	chats := result.Chats
	if chats == nil {
		chats = []chat{}
	}
	return connectors.JSONResult(map[string]any{"chats": chats})
}
