package monday

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip/connectors"
)

type listBoardsAction struct {
	conn *MondayConnector
}

type listBoardsParams struct {
	Limit int    `json:"limit"`
	Kind  string `json:"kind"`
}

func (a *listBoardsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params listBoardsParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}

	if err := validateBoardKind(params.Kind); err != nil {
		return nil, err
	}

	limit := params.Limit
	if limit <= 0 {
		limit = 20
	}

	query := `query ($limit: Int, $kind: BoardKind) {
		boards(limit: $limit, kind: $kind) {
			id
			name
			description
			state
			board_kind
			url
		}
	}`

	variables := map[string]any{
		"limit": limit,
	}
	if params.Kind != "" {
		variables["kind"] = params.Kind
	}

	var data struct {
		Boards []struct {
			ID          string `json:"id"`
			Name        string `json:"name"`
			Description string `json:"description"`
			State       string `json:"state"`
			BoardKind   string `json:"board_kind"`
			URL         string `json:"url"`
		} `json:"boards"`
	}

	if err := a.conn.query(ctx, req.Credentials, query, variables, &data); err != nil {
		return nil, err
	}

	return connectors.JSONResult(data.Boards)
}
