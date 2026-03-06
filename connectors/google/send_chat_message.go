package google

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// sendChatMessageAction implements connectors.Action for google.send_chat_message.
// It sends a message via the Google Chat API POST /v1/{parent}/messages.
type sendChatMessageAction struct {
	conn *GoogleConnector
}

// sendChatMessageParams is the user-facing parameter schema.
type sendChatMessageParams struct {
	SpaceName string `json:"space_name"`
	Text      string `json:"text"`
}

func (p *sendChatMessageParams) validate() error {
	if p.SpaceName == "" {
		return &connectors.ValidationError{Message: "missing required parameter: space_name"}
	}
	if p.Text == "" {
		return &connectors.ValidationError{Message: "missing required parameter: text"}
	}
	return nil
}

// chatMessageRequest is the Google Chat API request body for messages.create.
type chatMessageRequest struct {
	Text string `json:"text"`
}

// chatMessageResponse is the Google Chat API response from messages.create.
type chatMessageResponse struct {
	Name       string `json:"name"`
	CreateTime string `json:"createTime"`
	Space      struct {
		Name string `json:"name"`
	} `json:"space"`
	Thread struct {
		Name string `json:"name"`
	} `json:"thread"`
}

// Execute sends a message to a Google Chat space.
func (a *sendChatMessageAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params sendChatMessageParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := chatMessageRequest{Text: params.Text}
	var resp chatMessageResponse

	url := a.conn.chatBaseURL + "/v1/" + params.SpaceName + "/messages"
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodPost, url, body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]string{
		"name":        resp.Name,
		"space":       resp.Space.Name,
		"thread":      resp.Thread.Name,
		"create_time": resp.CreateTime,
	})
}
