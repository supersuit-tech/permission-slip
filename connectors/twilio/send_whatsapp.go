package twilio

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// sendWhatsAppAction implements connectors.Action for twilio.send_whatsapp.
// It sends a WhatsApp message via POST /Accounts/{sid}/Messages.json
// with the "whatsapp:" prefix on phone numbers.
type sendWhatsAppAction struct {
	conn *TwilioConnector
}

type sendWhatsAppParams struct {
	To   string `json:"to"`
	From string `json:"from"`
	Body string `json:"body"`
}

func (p *sendWhatsAppParams) validate() error {
	if p.To == "" {
		return &connectors.ValidationError{Message: "missing required parameter: to"}
	}
	if p.From == "" {
		return &connectors.ValidationError{Message: "missing required parameter: from"}
	}
	if p.Body == "" {
		return &connectors.ValidationError{Message: "missing required parameter: body"}
	}
	return nil
}

// Execute sends a WhatsApp message and returns the message metadata.
func (a *sendWhatsAppAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params sendWhatsAppParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	form := url.Values{
		"To":   {"whatsapp:" + params.To},
		"From": {"whatsapp:" + params.From},
		"Body": {params.Body},
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
