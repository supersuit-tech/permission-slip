package make

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// getScenarioAction implements connectors.Action for make.get_scenario.
type getScenarioAction struct {
	conn *MakeConnector
}

type getScenarioParams struct {
	ScenarioID int `json:"scenario_id"`
}

func (p *getScenarioParams) validate() error {
	if p.ScenarioID <= 0 {
		return &connectors.ValidationError{Message: "scenario_id must be a positive integer"}
	}
	return nil
}

// Execute retrieves scenario details.
func (a *getScenarioAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params getScenarioParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/scenarios/%d", params.ScenarioID)

	var resp json.RawMessage
	if err := a.conn.doRequest(ctx, "GET", path, req.Credentials, nil, &resp); err != nil {
		return nil, err
	}

	return &connectors.ActionResult{Data: resp}, nil
}
