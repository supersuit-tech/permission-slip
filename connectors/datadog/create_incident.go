package datadog

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// createIncidentAction implements connectors.Action for datadog.create_incident.
type createIncidentAction struct {
	conn *DatadogConnector
}

type createIncidentParams struct {
	Title                string `json:"title"`
	Severity             string `json:"severity"`
	CustomerImpactScope  string `json:"customer_impact_scope"`
	CustomerImpacted     bool   `json:"customer_impacted"`
}

func (p *createIncidentParams) validate() error {
	if p.Title == "" {
		return &connectors.ValidationError{Message: "missing required parameter: title"}
	}
	validSeverities := map[string]bool{
		"SEV-1": true, "SEV-2": true, "SEV-3": true,
		"SEV-4": true, "SEV-5": true, "UNKNOWN": true, "": true,
	}
	if !validSeverities[p.Severity] {
		return &connectors.ValidationError{Message: fmt.Sprintf("invalid severity: %s", p.Severity)}
	}
	return nil
}

// Execute creates a new incident in Datadog via the v2 API.
func (a *createIncidentAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createIncidentParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	severity := params.Severity
	if severity == "" {
		severity = "UNKNOWN"
	}

	body := map[string]any{
		"data": map[string]any{
			"type": "incidents",
			"attributes": map[string]any{
				"title":                  params.Title,
				"customer_impacted":      params.CustomerImpacted,
				"customer_impact_scope":  params.CustomerImpactScope,
				"fields": map[string]any{
					"severity": map[string]any{
						"type":  "dropdown",
						"value": severity,
					},
				},
			},
		},
	}

	var respBody json.RawMessage
	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, "/api/v2/incidents", body, &respBody); err != nil {
		return nil, err
	}

	return connectors.JSONResult(respBody)
}
