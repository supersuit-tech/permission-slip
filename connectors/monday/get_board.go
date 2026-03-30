package monday

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip/connectors"
)

type getBoardAction struct {
	conn *MondayConnector
}

type getBoardParams struct {
	BoardID string `json:"board_id"`
}

func (p *getBoardParams) validate() error {
	if p.BoardID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: board_id"}
	}
	if !isValidMondayID(p.BoardID) {
		return &connectors.ValidationError{Message: "board_id must be a numeric string"}
	}
	return nil
}

func (a *getBoardAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params getBoardParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	query := `query ($ids: [ID!]) {
		boards(ids: $ids) {
			id
			name
			description
			state
			board_kind
			url
			columns {
				id
				title
				type
			}
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
			ID          string `json:"id"`
			Name        string `json:"name"`
			Description string `json:"description"`
			State       string `json:"state"`
			BoardKind   string `json:"board_kind"`
			URL         string `json:"url"`
			Columns     []struct {
				ID    string `json:"id"`
				Title string `json:"title"`
				Type  string `json:"type"`
			} `json:"columns"`
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
		return nil, &connectors.ExternalError{
			StatusCode: 404,
			Message:    fmt.Sprintf("board %s not found", params.BoardID),
		}
	}

	return connectors.JSONResult(data.Boards[0])
}
