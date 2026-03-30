package make

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// listScenariosAction implements connectors.Action for make.list_scenarios.
type listScenariosAction struct {
	conn *MakeConnector
}

type listScenariosParams struct {
	TeamID int `json:"team_id"`
	Limit  int `json:"limit"`
	Offset int `json:"offset"`
}

func (p *listScenariosParams) validate() error {
	if p.TeamID <= 0 {
		return &connectors.ValidationError{Message: "team_id must be a positive integer"}
	}
	if p.Limit < 0 || p.Limit > 100 {
		return &connectors.ValidationError{Message: "limit must be between 0 and 100"}
	}
	if p.Offset < 0 {
		return &connectors.ValidationError{Message: "offset must be non-negative"}
	}
	return nil
}

// Execute lists scenarios for a team.
func (a *listScenariosAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params listScenariosParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	limit := params.Limit
	if limit == 0 {
		limit = 50
	}

	q := url.Values{}
	q.Set("teamId", fmt.Sprintf("%d", params.TeamID))
	q.Set("pg[limit]", fmt.Sprintf("%d", limit))
	q.Set("pg[offset]", fmt.Sprintf("%d", params.Offset))
	path := "/scenarios?" + q.Encode()

	var resp json.RawMessage
	if err := a.conn.doRequest(ctx, "GET", path, req.Credentials, nil, &resp); err != nil {
		return nil, err
	}

	return &connectors.ActionResult{Data: resp}, nil
}
