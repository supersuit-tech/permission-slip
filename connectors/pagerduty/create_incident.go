package pagerduty

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// createIncidentAction implements connectors.Action for pagerduty.create_incident.
type createIncidentAction struct {
	conn *PagerDutyConnector
}

type createIncidentParams struct {
	ServiceID          string `json:"service_id"`
	Title              string `json:"title"`
	Body               string `json:"body"`
	Urgency            string `json:"urgency"`
	EscalationPolicyID string `json:"escalation_policy_id"`
}

func (p *createIncidentParams) validate() error {
	if p.ServiceID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: service_id"}
	}
	if p.Title == "" {
		return &connectors.ValidationError{Message: "missing required parameter: title"}
	}
	if p.Urgency != "" && p.Urgency != "high" && p.Urgency != "low" {
		return &connectors.ValidationError{Message: fmt.Sprintf("invalid urgency: %s (must be 'high' or 'low')", p.Urgency)}
	}
	return nil
}

// Execute creates a new incident in PagerDuty.
func (a *createIncidentAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createIncidentParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	incident := map[string]any{
		"type":  "incident",
		"title": params.Title,
		"service": map[string]any{
			"id":   params.ServiceID,
			"type": "service_reference",
		},
	}

	if params.Body != "" {
		incident["body"] = map[string]any{
			"type":    "incident_body",
			"details": params.Body,
		}
	}
	if params.Urgency != "" {
		incident["urgency"] = params.Urgency
	}
	if params.EscalationPolicyID != "" {
		incident["escalation_policy"] = map[string]any{
			"id":   params.EscalationPolicyID,
			"type": "escalation_policy_reference",
		}
	}

	body := map[string]any{"incident": incident}

	var respBody json.RawMessage
	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, "/incidents", body, &respBody); err != nil {
		return nil, err
	}

	return connectors.JSONResult(respBody)
}
