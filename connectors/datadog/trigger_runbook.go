package datadog

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip/connectors"
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

	meta := map[string]any{}
	if len(params.Payload) > 0 {
		meta["payload"] = params.Payload
	}

	body := map[string]any{
		"meta": meta,
	}

	var respBody json.RawMessage
	path := fmt.Sprintf("/api/v2/workflows/%s/instances", url.PathEscape(params.WorkflowID))
	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, path, body, &respBody); err != nil {
		return nil, err
	}

	return connectors.JSONResult(respBody)
}
