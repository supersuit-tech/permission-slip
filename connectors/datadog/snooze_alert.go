package datadog

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// snoozeAlertAction implements connectors.Action for datadog.snooze_alert.
type snoozeAlertAction struct {
	conn *DatadogConnector
}

type snoozeAlertParams struct {
	MonitorID int64  `json:"monitor_id"`
	End       int64  `json:"end"`
	Scope     string `json:"scope"`
}

func (p *snoozeAlertParams) validate() error {
	if p.MonitorID <= 0 {
		return &connectors.ValidationError{Message: "missing required parameter: monitor_id"}
	}
	return nil
}

// Execute mutes a Datadog monitor for a specified duration.
func (a *snoozeAlertAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params snoozeAlertParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := map[string]any{}
	if params.End != 0 {
		body["end"] = params.End
	}
	if params.Scope != "" {
		body["scope"] = params.Scope
	}

	var respBody json.RawMessage
	path := fmt.Sprintf("/api/v1/monitor/%d/mute", params.MonitorID)
	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, path, body, &respBody); err != nil {
		return nil, err
	}

	return connectors.JSONResult(respBody)
}
