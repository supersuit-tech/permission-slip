package trello

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip/connectors"
)

type listListsAction struct {
	conn *TrelloConnector
}

type listListsParams struct {
	BoardID string `json:"board_id"`
	Filter  string `json:"filter"`
}

func (p *listListsParams) validate() error {
	return validateTrelloID(p.BoardID, "board_id")
}

func (a *listListsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params listListsParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	filter := params.Filter
	if filter == "" {
		filter = "open"
	}

	var lists []struct {
		ID     string `json:"id"`
		Name   string `json:"name"`
		Closed bool   `json:"closed"`
		IDBoard string `json:"idBoard"`
		Pos    float64 `json:"pos"`
	}

	if err := a.conn.doGet(ctx, req.Credentials, "/boards/"+params.BoardID+"/lists", map[string]string{
		"filter": filter,
	}, &lists); err != nil {
		return nil, err
	}

	return connectors.JSONResult(lists)
}
