package pagerduty

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// acknowledgeAlertAction implements connectors.Action for pagerduty.acknowledge_alert.
type acknowledgeAlertAction struct {
	conn *PagerDutyConnector
}

type acknowledgeAlertParams struct {
	IncidentID string `json:"incident_id"`
}

func (p *acknowledgeAlertParams) validate() error {
	if p.IncidentID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: incident_id"}
	}
	return nil
}

// Execute acknowledges an incident in PagerDuty.
func (a *acknowledgeAlertAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params acknowledgeAlertParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := map[string]any{
		"incident": map[string]any{
			"type":   "incident_reference",
			"status": "acknowledged",
		},
	}

	var respBody json.RawMessage
	path := fmt.Sprintf("/incidents/%s", params.IncidentID)
	if err := a.conn.do(ctx, req.Credentials, http.MethodPut, path, body, &respBody); err != nil {
		return nil, err
	}

	return connectors.JSONResult(respBody)
}
