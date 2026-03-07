package make

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// listExecutionsAction implements connectors.Action for make.list_executions.
type listExecutionsAction struct {
	conn *MakeConnector
}

type listExecutionsParams struct {
	ScenarioID int `json:"scenario_id"`
	Limit      int `json:"limit"`
	Offset     int `json:"offset"`
}

func (p *listExecutionsParams) validate() error {
	if p.ScenarioID <= 0 {
		return &connectors.ValidationError{Message: "scenario_id must be a positive integer"}
	}
	if p.Limit < 0 || p.Limit > 100 {
		return &connectors.ValidationError{Message: "limit must be between 0 and 100"}
	}
	if p.Offset < 0 {
		return &connectors.ValidationError{Message: "offset must be non-negative"}
	}
	return nil
}

// Execute lists execution history for a scenario.
func (a *listExecutionsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params listExecutionsParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	limit := params.Limit
	if limit == 0 {
		limit = 20
	}

	path := fmt.Sprintf("/scenarios/%d/logs?pg[limit]=%d&pg[offset]=%d", params.ScenarioID, limit, params.Offset)

	var resp json.RawMessage
	if err := a.conn.doRequest(ctx, "GET", path, req.Credentials, nil, &resp); err != nil {
		return nil, err
	}

	return &connectors.ActionResult{Data: resp}, nil
}
