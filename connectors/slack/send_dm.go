package slack

import (
	"context"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// sendDMAction implements connectors.Action for slack.send_dm.
// It opens (or reuses) a DM channel with a user and sends a message
// via conversations.open + chat.postMessage.
type sendDMAction struct {
	conn *SlackConnector
}

// sendDMParams is the user-facing parameter schema.
type sendDMParams struct {
	UserID  string `json:"user_id"`
	Message string `json:"message"`
}

func (p *sendDMParams) validate() error {
	if err := validateUserID(p.UserID); err != nil {
		return err
	}
	if p.Message == "" {
		return &connectors.ValidationError{Message: "missing required parameter: message"}
	}
	return nil
}

// conversationsOpenRequest is the Slack API request body for conversations.open.
type conversationsOpenRequest struct {
	Users string `json:"users"`
}

type conversationsOpenResponse struct {
	slackResponse
	Channel struct {
		ID string `json:"id"`
	} `json:"channel"`
}

// Execute opens a DM channel with the user and sends a message.
func (a *sendDMAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params sendDMParams
	if err := parseAndValidate(req.Parameters, &params); err != nil {
		return nil, err
	}

	// Step 1: Open (or reuse) a DM channel with the user.
	openBody := conversationsOpenRequest{Users: params.UserID}
	var openResp conversationsOpenResponse
	if err := a.conn.doPost(ctx, "conversations.open", req.Credentials, openBody, &openResp); err != nil {
		return nil, err
	}
	if !openResp.OK {
		return nil, mapSlackError(openResp.Error)
	}

	dmChannelID := openResp.Channel.ID

	// Step 2: Post the message to the DM channel.
	msgBody := sendMessageRequest{
		Channel: dmChannelID,
		Text:    params.Message,
	}
	var msgResp sendMessageResponse
	if err := a.conn.doPost(ctx, "chat.postMessage", req.Credentials, msgBody, &msgResp); err != nil {
		return nil, err
	}
	if !msgResp.OK {
		return nil, mapSlackError(msgResp.Error)
	}

	return connectors.JSONResult(map[string]string{
		"ts":      msgResp.TS,
		"channel": msgResp.Channel,
	})
}
