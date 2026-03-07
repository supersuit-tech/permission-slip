package datadog

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// getIncidentAction implements connectors.Action for datadog.get_incident.
type getIncidentAction struct {
	conn *DatadogConnector
}

type getIncidentParams struct {
	IncidentID string `json:"incident_id"`
}

func (p *getIncidentParams) validate() error {
	if p.IncidentID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: incident_id"}
	}
	return nil
}

// Execute retrieves an existing incident from Datadog by ID.
func (a *getIncidentAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params getIncidentParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	var respBody json.RawMessage
	path := fmt.Sprintf("/api/v2/incidents/%s", params.IncidentID)
	if err := a.conn.do(ctx, req.Credentials, http.MethodGet, path, nil, &respBody); err != nil {
		return nil, err
	}

	return connectors.JSONResult(respBody)
}
