package salesforce

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// createTaskAction implements connectors.Action for salesforce.create_task.
// It provides a friendlier interface than create_record for creating Task objects.
type createTaskAction struct {
	conn *SalesforceConnector
}

type createTaskParams struct {
	Subject     string `json:"subject"`
	WhatID      string `json:"what_id"`
	WhoID       string `json:"who_id"`
	Status      string `json:"status"`
	Priority    string `json:"priority"`
	DueDate     string `json:"due_date"`
	Description string `json:"description"`
}

func (p *createTaskParams) validate() error {
	if p.Subject == "" {
		return &connectors.ValidationError{Message: "missing required parameter: subject"}
	}
	return nil
}

func (a *createTaskAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createTaskParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	// Default status if not provided.
	if params.Status == "" {
		params.Status = "Not Started"
	}

	fields := map[string]string{
		"Subject": params.Subject,
		"Status":  params.Status,
	}
	if params.WhatID != "" {
		fields["WhatId"] = params.WhatID
	}
	if params.WhoID != "" {
		fields["WhoId"] = params.WhoID
	}
	if params.Priority != "" {
		fields["Priority"] = params.Priority
	}
	if params.DueDate != "" {
		fields["ActivityDate"] = params.DueDate
	}
	if params.Description != "" {
		fields["Description"] = params.Description
	}

	baseURL, err := a.conn.apiBaseURL(req.Credentials)
	if err != nil {
		return nil, err
	}

	apiURL := baseURL + "/sobjects/Task/"

	var resp sfCreateResponse
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodPost, apiURL, fields, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]any{
		"id":      resp.ID,
		"success": resp.Success,
	})
}
