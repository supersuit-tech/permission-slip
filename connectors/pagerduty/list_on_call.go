package pagerduty

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// listOnCallAction implements connectors.Action for pagerduty.list_on_call.
type listOnCallAction struct {
	conn *PagerDutyConnector
}

type listOnCallParams struct {
	ScheduleIDs         []string `json:"schedule_ids"`
	EscalationPolicyIDs []string `json:"escalation_policy_ids"`
	Since               string   `json:"since"`
	Until               string   `json:"until"`
}

// Execute lists current on-call entries from PagerDuty.
func (a *listOnCallAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params listOnCallParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}

	q := url.Values{}
	for _, id := range params.ScheduleIDs {
		q.Add("schedule_ids[]", id)
	}
	for _, id := range params.EscalationPolicyIDs {
		q.Add("escalation_policy_ids[]", id)
	}
	if params.Since != "" {
		q.Set("since", params.Since)
	}
	if params.Until != "" {
		q.Set("until", params.Until)
	}

	path := "/oncalls"
	if encoded := q.Encode(); encoded != "" {
		path += "?" + encoded
	}

	// Replace %5B%5D with [] for PagerDuty API compatibility.
	path = strings.ReplaceAll(path, "%5B%5D", "[]")

	var respBody json.RawMessage
	if err := a.conn.do(ctx, req.Credentials, http.MethodGet, path, nil, &respBody); err != nil {
		return nil, err
	}

	return connectors.JSONResult(respBody)
}
