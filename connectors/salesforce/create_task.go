package salesforce

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/supersuit-tech/permission-slip/connectors"
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
	if p.WhatID != "" {
		if err := validateRecordID(p.WhatID, "what_id"); err != nil {
			return err
		}
	}
	if p.WhoID != "" {
		if err := validateRecordID(p.WhoID, "who_id"); err != nil {
			return err
		}
	}
	if p.DueDate != "" {
		if _, err := time.Parse("2006-01-02", p.DueDate); err != nil {
			return &connectors.ValidationError{
				Message: fmt.Sprintf("due_date must be in YYYY-MM-DD format, got %q", p.DueDate),
			}
		}
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

	// Apply sensible defaults.
	if params.Status == "" {
		params.Status = "Not Started"
	}
	if params.Priority == "" {
		params.Priority = "Normal"
	}

	fields := map[string]string{
		"Subject":  params.Subject,
		"Status":   params.Status,
		"Priority": params.Priority,
	}
	if params.WhatID != "" {
		fields["WhatId"] = params.WhatID
	}
	if params.WhoID != "" {
		fields["WhoId"] = params.WhoID
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

	result := map[string]any{
		"id":      resp.ID,
		"success": resp.Success,
	}
	if url := recordURL(req.Credentials, resp.ID); url != "" {
		result["record_url"] = url
	}
	return connectors.JSONResult(result)
}
