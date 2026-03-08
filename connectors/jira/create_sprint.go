package jira

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// createSprintAction implements connectors.Action for jira.create_sprint.
// It creates a sprint via POST /rest/agile/1.0/sprint.
type createSprintAction struct {
	conn *JiraConnector
}

type createSprintParams struct {
	Name          string `json:"name"`
	BoardID       int    `json:"board_id"`
	Goal          string `json:"goal"`
	StartDate     string `json:"start_date"`
	EndDate       string `json:"end_date"`
}

func (p *createSprintParams) validate() error {
	p.Name = strings.TrimSpace(p.Name)
	if p.Name == "" {
		return &connectors.ValidationError{Message: "missing required parameter: name"}
	}
	if p.BoardID <= 0 {
		return &connectors.ValidationError{Message: "missing required parameter: board_id (must be a positive integer)"}
	}
	return nil
}

func (a *createSprintAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createSprintParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := map[string]interface{}{
		"name":            params.Name,
		"originBoardId":   params.BoardID,
	}
	if params.Goal != "" {
		body["goal"] = params.Goal
	}
	if params.StartDate != "" {
		body["startDate"] = params.StartDate
	}
	if params.EndDate != "" {
		body["endDate"] = params.EndDate
	}

	var resp json.RawMessage
	if err := a.conn.doAgile(ctx, req.Credentials, http.MethodPost, "/sprint", body, &resp); err != nil {
		return nil, err
	}

	return &connectors.ActionResult{Data: resp}, nil
}
