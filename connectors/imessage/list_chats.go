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
	Limit      int  `json:"limit"`
	UnreadOnly bool `json:"unread_only"`
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

	fetchLimit := params.Limit
	if params.UnreadOnly {
		// Over-fetch when filtering client-side so we can still return up to limit
		// unread chats even when many recent chats are already read.
		fetchLimit = params.Limit * 5
		if fetchLimit > 100 {
			fetchLimit = 100
		}
	}

	rpcParams := map[string]any{
		"limit": fetchLimit,
	}
	if params.UnreadOnly {
		rpcParams["unread_only"] = true
	}

	var result chatsListResult
	if err := a.conn.client.rpcCall(actionCtx, req.Credentials, "chats.list", rpcParams, &result); err != nil {
		return nil, err
	}
	chats := result.Chats
	if chats == nil {
		chats = []chat{}
	}
	if params.UnreadOnly {
		chats = filterUnreadChats(chats, params.Limit)
	}
	return connectors.JSONResult(map[string]any{"chats": chats})
}

// filterUnreadChats keeps chats with unread_count > 0, capped at limit.
// When imsg omits unread_count (older builds), the zero value is treated as no unreads.
func filterUnreadChats(chats []chat, limit int) []chat {
	out := make([]chat, 0, limit)
	for _, c := range chats {
		if c.UnreadCount <= 0 {
			continue
		}
		out = append(out, c)
		if len(out) >= limit {
			break
		}
	}
	return out
}
