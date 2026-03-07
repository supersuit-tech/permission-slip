package make

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// runScenarioAction implements connectors.Action for make.run_scenario.
type runScenarioAction struct {
	conn *MakeConnector
}

type runScenarioParams struct {
	ScenarioID int              `json:"scenario_id"`
	Data       *json.RawMessage `json:"data,omitempty"`
	Responsive bool             `json:"responsive"`
}

func (p *runScenarioParams) validate() error {
	if p.ScenarioID <= 0 {
		return &connectors.ValidationError{Message: "scenario_id must be a positive integer"}
	}
	return nil
}

// Execute runs a Make scenario.
func (a *runScenarioAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params runScenarioParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/scenarios/%d/run", params.ScenarioID)

	body := map[string]any{
		"responsive": params.Responsive,
	}
	if params.Data != nil {
		var data any
		if err := json.Unmarshal(*params.Data, &data); err != nil {
			return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid data JSON: %v", err)}
		}
		body["data"] = data
	}

	var resp json.RawMessage
	if err := a.conn.doRequest(ctx, "POST", path, req.Credentials, body, &resp); err != nil {
		return nil, err
	}

	return &connectors.ActionResult{Data: resp}, nil
}
