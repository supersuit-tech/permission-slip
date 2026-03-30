package twilio

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// getMessageAction implements connectors.Action for twilio.get_message.
// It retrieves message details via GET /Accounts/{sid}/Messages/{MessageSid}.json.
type getMessageAction struct {
	conn *TwilioConnector
}

type getMessageParams struct {
	MessageSID string `json:"message_sid"`
}

func (p *getMessageParams) validate() error {
	if p.MessageSID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: message_sid"}
	}
	if len(p.MessageSID) < 2 || (p.MessageSID[:2] != "SM" && p.MessageSID[:2] != "MM") {
		return &connectors.ValidationError{
			Message: fmt.Sprintf("message_sid must start with \"SM\" or \"MM\", got %q", p.MessageSID),
		}
	}
	return nil
}

// Execute retrieves a message's status and details.
func (a *getMessageAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params getMessageParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	reqURL := a.conn.accountURL(req.Credentials) + "/Messages/" + url.PathEscape(params.MessageSID) + ".json"

	var resp struct {
		SID       string `json:"sid"`
		Status    string `json:"status"`
		To        string `json:"to"`
		From      string `json:"from"`
		Body      string `json:"body"`
		DateSent  string `json:"date_sent"`
		Direction string `json:"direction"`
	}

	if err := a.conn.doGet(ctx, req.Credentials, reqURL, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
