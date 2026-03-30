package pagerduty

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// resolveIncidentAction implements connectors.Action for pagerduty.resolve_incident.
type resolveIncidentAction struct {
	conn *PagerDutyConnector
}

type resolveIncidentParams struct {
	IncidentID string `json:"incident_id"`
}

func (p *resolveIncidentParams) validate() error {
	if p.IncidentID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: incident_id"}
	}
	return nil
}

// Execute resolves an incident in PagerDuty.
func (a *resolveIncidentAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params resolveIncidentParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := map[string]any{
		"incident": map[string]any{
			"type":   "incident_reference",
			"status": "resolved",
		},
	}

	var respBody json.RawMessage
	path := fmt.Sprintf("/incidents/%s", url.PathEscape(params.IncidentID))
	if err := a.conn.do(ctx, req.Credentials, http.MethodPut, path, body, &respBody); err != nil {
		return nil, err
	}

	return connectors.JSONResult(respBody)
}
