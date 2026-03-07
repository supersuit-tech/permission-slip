package twilio

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// sendSMSAction implements connectors.Action for twilio.send_sms.
// It sends an SMS (or MMS) message via POST /Accounts/{sid}/Messages.json.
type sendSMSAction struct {
	conn *TwilioConnector
}

type sendSMSParams struct {
	To       string `json:"to"`
	From     string `json:"from"`
	Body     string `json:"body"`
	MediaURL string `json:"media_url"`
}

func (p *sendSMSParams) validate() error {
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
	if p.Body == "" {
		return &connectors.ValidationError{Message: "missing required parameter: body"}
	}
	if len(p.Body) > maxSMSBodyLen {
		return &connectors.ValidationError{
			Message: fmt.Sprintf("body exceeds maximum length of %d characters (got %d)", maxSMSBodyLen, len(p.Body)),
		}
	}
	return nil
}

// Execute sends an SMS message and returns the message metadata.
func (a *sendSMSAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params sendSMSParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	form := url.Values{
		"To":   {params.To},
		"From": {params.From},
		"Body": {params.Body},
	}
	if params.MediaURL != "" {
		form.Set("MediaUrl", params.MediaURL)
	}

	var resp struct {
		SID    string `json:"sid"`
		Status string `json:"status"`
		To     string `json:"to"`
		From   string `json:"from"`
	}

	if err := a.conn.doForm(ctx, req.Credentials, "/Messages.json", form, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
