package pagerduty

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// escalateIncidentAction implements connectors.Action for pagerduty.escalate_incident.
type escalateIncidentAction struct {
	conn *PagerDutyConnector
}

type escalateIncidentParams struct {
	IncidentID         string `json:"incident_id"`
	EscalationLevel    int    `json:"escalation_level"`
	EscalationPolicyID string `json:"escalation_policy_id"`
}

func (p *escalateIncidentParams) validate() error {
	if p.IncidentID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: incident_id"}
	}
	if p.EscalationLevel == 0 {
		return &connectors.ValidationError{Message: "missing required parameter: escalation_level"}
	}
	return nil
}

// Execute escalates an incident in PagerDuty.
func (a *escalateIncidentAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params escalateIncidentParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	incident := map[string]any{
		"type":             "incident_reference",
		"escalation_level": params.EscalationLevel,
	}
	if params.EscalationPolicyID != "" {
		incident["escalation_policy"] = map[string]any{
			"id":   params.EscalationPolicyID,
			"type": "escalation_policy_reference",
		}
	}

	body := map[string]any{"incident": incident}

	var respBody json.RawMessage
	path := fmt.Sprintf("/incidents/%s", url.PathEscape(params.IncidentID))
	if err := a.conn.do(ctx, req.Credentials, http.MethodPut, path, body, &respBody); err != nil {
		return nil, err
	}

	return connectors.JSONResult(respBody)
}
