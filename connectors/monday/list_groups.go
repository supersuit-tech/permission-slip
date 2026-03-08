package monday

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

type listGroupsAction struct {
	conn *MondayConnector
}

type listGroupsParams struct {
	BoardID string `json:"board_id"`
}

func (p *listGroupsParams) validate() error {
	if p.BoardID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: board_id"}
	}
	if !isValidMondayID(p.BoardID) {
		return &connectors.ValidationError{Message: "board_id must be a numeric string"}
	}
	return nil
}

func (a *listGroupsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params listGroupsParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	query := `query ($ids: [ID!]) {
		boards(ids: $ids) {
			groups {
				id
				title
				color
				position
			}
		}
	}`

	variables := map[string]any{
		"ids": []string{params.BoardID},
	}

	var data struct {
		Boards []struct {
			Groups []struct {
				ID       string `json:"id"`
				Title    string `json:"title"`
				Color    string `json:"color"`
				Position string `json:"position"`
			} `json:"groups"`
		} `json:"boards"`
	}

	if err := a.conn.query(ctx, req.Credentials, query, variables, &data); err != nil {
		return nil, err
	}

	if len(data.Boards) == 0 {
		return connectors.JSONResult([]struct{}{})
	}

	return connectors.JSONResult(data.Boards[0].Groups)
}
