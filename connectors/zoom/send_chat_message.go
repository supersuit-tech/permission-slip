package zoom

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// sendChatMessageAction implements connectors.Action for zoom.send_chat_message.
// It sends a message in Zoom Team Chat via POST /chat/users/me/messages.
type sendChatMessageAction struct {
	conn *ZoomConnector
}

type sendChatMessageParams struct {
	Message   string `json:"message"`
	ToJID     string `json:"to_jid,omitempty"`
	ToChannel string `json:"to_channel,omitempty"`
}

func (p *sendChatMessageParams) validate() error {
	p.Message = strings.TrimSpace(p.Message)
	if p.Message == "" {
		return &connectors.ValidationError{Message: "missing required parameter: message"}
	}
	p.ToJID = strings.TrimSpace(p.ToJID)
	p.ToChannel = strings.TrimSpace(p.ToChannel)
	if p.ToJID == "" && p.ToChannel == "" {
		return &connectors.ValidationError{Message: "either to_jid or to_channel is required"}
	}
	if p.ToJID != "" && p.ToChannel != "" {
		return &connectors.ValidationError{Message: "provide either to_jid or to_channel, not both"}
	}
	return nil
}

type sendChatMessageRequest struct {
	Message   string `json:"message"`
	ToJID     string `json:"to_jid,omitempty"`
	ToChannel string `json:"to_channel,omitempty"`
}

type sendChatMessageResponse struct {
	ID string `json:"id"`
}

func (a *sendChatMessageAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params sendChatMessageParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := sendChatMessageRequest{
		Message:   params.Message,
		ToJID:     params.ToJID,
		ToChannel: params.ToChannel,
	}

	var resp sendChatMessageResponse
	reqURL := fmt.Sprintf("%s/chat/users/me/messages", a.conn.baseURL)
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodPost, reqURL, body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]string{
		"id": resp.ID,
	})
}
