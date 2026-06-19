package imessage

import (
	"context"
	"encoding/json"

	"github.com/supersuit-tech/permission-slip/connectors"
)

type readHistoryAction struct {
	conn *IMessageConnector
}

type readHistoryParams struct {
	ChatID      int    `json:"chat_id"`
	Limit       int    `json:"limit"`
	SinceGUID   string `json:"since_guid"`
	SinceRowID  int    `json:"since_rowid"`
	Attachments bool   `json:"attachments"`
	Start       string `json:"start"`
	End         string `json:"end"`
}

func (p *readHistoryParams) validate() error {
	if p.ChatID <= 0 {
		return &connectors.ValidationError{Message: "missing or invalid required parameter: chat_id"}
	}
	if p.Limit <= 0 {
		p.Limit = 50
	}
	if p.Limit > 200 {
		return &connectors.ValidationError{Message: "limit must be at most 200"}
	}
	return nil
}

func (a *readHistoryAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params readHistoryParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: "invalid parameters: " + err.Error()}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	actionCtx, cancel := a.conn.actionTimeout(ctx)
	defer cancel()

	rpcParams := map[string]any{
		"chat_id": params.ChatID,
		"limit":   params.Limit,
	}
	if params.Attachments {
		rpcParams["attachments"] = true
	}
	if params.Start != "" {
		rpcParams["start"] = params.Start
	}
	if params.End != "" {
		rpcParams["end"] = params.End
	}

	var result messagesHistoryResult
	if err := a.conn.client.rpcCall(actionCtx, req.Credentials, "messages.history", rpcParams, &result); err != nil {
		return nil, err
	}

	messages := filterMessagesSince(result.Messages, params.SinceGUID, params.SinceRowID)
	return connectors.JSONResult(map[string]any{
		"messages": messages,
		"count":    len(messages),
	})
}

// filterMessagesSince returns only messages newer than the given cursor.
func filterMessagesSince(messages []message, sinceGUID string, sinceRowID int) []message {
	if sinceGUID == "" && sinceRowID <= 0 {
		return messages
	}

	cutoffID := sinceRowID
	if sinceGUID != "" {
		for _, m := range messages {
			if m.GUID == sinceGUID && m.ID > cutoffID {
				cutoffID = m.ID
			}
		}
	}

	out := make([]message, 0, len(messages))
	for _, m := range messages {
		if cutoffID > 0 && m.ID <= cutoffID {
			continue
		}
		if sinceGUID != "" && m.GUID == sinceGUID {
			continue
		}
		out = append(out, m)
	}
	return out
}
