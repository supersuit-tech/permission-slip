package datadog

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// triggerRunbookAction implements connectors.Action for datadog.trigger_runbook.
type triggerRunbookAction struct {
	conn *DatadogConnector
}

type triggerRunbookParams struct {
	WorkflowID string          `json:"workflow_id"`
	Payload    json.RawMessage `json:"payload"`
}

func (p *triggerRunbookParams) validate() error {
	if p.WorkflowID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: workflow_id"}
	}
	return nil
}

// Execute triggers a Datadog Workflow automation.
func (a *triggerRunbookAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params triggerRunbookParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := map[string]any{
		"meta": map[string]any{
			"payload": params.Payload,
		},
	}

	var respBody json.RawMessage
	path := fmt.Sprintf("/api/v2/workflows/%s/instances", params.WorkflowID)
	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, path, body, &respBody); err != nil {
		return nil, err
	}

	return connectors.JSONResult(respBody)
}
