package twilio

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// getCallAction implements connectors.Action for twilio.get_call.
// It retrieves call details via GET /Accounts/{sid}/Calls/{CallSid}.json.
type getCallAction struct {
	conn *TwilioConnector
}

type getCallParams struct {
	CallSID string `json:"call_sid"`
}

func (p *getCallParams) validate() error {
	if p.CallSID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: call_sid"}
	}
	return nil
}

// Execute retrieves a call's status and details.
func (a *getCallAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params getCallParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	sid, _ := req.Credentials.Get(credKeyAccountSID)
	reqURL := a.conn.baseURL + "/Accounts/" + url.PathEscape(sid) + "/Calls/" + url.PathEscape(params.CallSID) + ".json"

	var resp struct {
		SID       string `json:"sid"`
		Status    string `json:"status"`
		To        string `json:"to"`
		From      string `json:"from"`
		Duration  string `json:"duration"`
		Direction string `json:"direction"`
		StartTime string `json:"start_time"`
		EndTime   string `json:"end_time"`
	}

	if err := a.conn.doGet(ctx, req.Credentials, reqURL, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
