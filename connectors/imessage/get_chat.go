package imessage

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip/connectors"
)

type getChatAction struct {
	conn *IMessageConnector
}

type getChatParams struct {
	ChatID int `json:"chat_id"`
}

func (p *getChatParams) validate() error {
	if p.ChatID <= 0 {
		return &connectors.ValidationError{Message: "missing or invalid required parameter: chat_id"}
	}
	return nil
}

func (a *getChatAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params getChatParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: "invalid parameters: " + err.Error()}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	actionCtx, cancel := a.conn.actionTimeout(ctx)
	defer cancel()

	lines, err := a.conn.client.runCLI(actionCtx, req.Credentials,
		"group",
		"--chat-id", fmt.Sprintf("%d", params.ChatID),
		"--json",
	)
	if err != nil {
		return nil, err
	}
	if len(lines) == 0 {
		return nil, &connectors.ExternalError{Message: fmt.Sprintf("chat %d not found", params.ChatID)}
	}

	var chatObj chat
	if err := json.Unmarshal(lines[0], &chatObj); err != nil {
		return nil, &connectors.ExternalError{Message: fmt.Sprintf("parse chat response: %v", err)}
	}
	return connectors.JSONResult(chatObj)
}
