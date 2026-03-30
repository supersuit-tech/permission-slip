package twilio

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// initiateCallAction implements connectors.Action for twilio.initiate_call.
// It initiates an outbound voice call via POST /Accounts/{sid}/Calls.json.
type initiateCallAction struct {
	conn *TwilioConnector
}

type initiateCallParams struct {
	To    string `json:"to"`
	From  string `json:"from"`
	Twiml string `json:"twiml"`
}

func (p *initiateCallParams) validate() error {
	if p.To == "" {
		return &connectors.ValidationError{Message: "missing required parameter: to"}
	}
	if err := validateE164("to", p.To); err != nil {
		return err
	}
	if p.From == "" {
		return &connectors.ValidationError{Message: "missing required parameter: from"}
	}
	if err := validateE164("from", p.From); err != nil {
		return err
	}
	if p.Twiml == "" {
		return &connectors.ValidationError{Message: "missing required parameter: twiml"}
	}
	return nil
}

// Execute initiates an outbound call and returns the call metadata.
func (a *initiateCallAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params initiateCallParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	form := url.Values{
		"To":    {params.To},
		"From":  {params.From},
		"Twiml": {params.Twiml},
	}

	var resp messageResponse
	if err := a.conn.doForm(ctx, req.Credentials, "/Calls.json", form, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
