package make

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// toggleScenarioAction implements connectors.Action for make.toggle_scenario.
type toggleScenarioAction struct {
	conn *MakeConnector
}

type toggleScenarioParams struct {
	ScenarioID int  `json:"scenario_id"`
	Enabled    bool `json:"enabled"`
}

func (p *toggleScenarioParams) validate() error {
	if p.ScenarioID <= 0 {
		return &connectors.ValidationError{Message: "scenario_id must be a positive integer"}
	}
	return nil
}

// Execute enables or disables a scenario.
func (a *toggleScenarioAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params toggleScenarioParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/scenarios/%d", params.ScenarioID)
	body := map[string]any{
		"scheduling": map[string]any{
			"isEnabled": params.Enabled,
		},
	}

	var resp json.RawMessage
	if err := a.conn.doRequest(ctx, "PATCH", path, req.Credentials, body, &resp); err != nil {
		return nil, err
	}

	// Wrap the raw API response with a clear confirmation of what changed,
	// so users don't have to parse the full scenario object to verify.
	action := "enabled"
	if !params.Enabled {
		action = "disabled"
	}
	result := map[string]any{
		"status":      action,
		"scenario_id": params.ScenarioID,
		"scenario":    resp,
	}

	return connectors.JSONResult(result)
}
