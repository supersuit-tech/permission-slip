package trello

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

type listLabelsAction struct {
	conn *TrelloConnector
}

type listLabelsParams struct {
	BoardID string `json:"board_id"`
}

func (p *listLabelsParams) validate() error {
	return validateTrelloID(p.BoardID, "board_id")
}

func (a *listLabelsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params listLabelsParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	var labels []struct {
		ID      string `json:"id"`
		IDBoard string `json:"idBoard"`
		Name    string `json:"name"`
		Color   string `json:"color"`
	}

	if err := a.conn.doGet(ctx, req.Credentials, "/boards/"+params.BoardID+"/labels", nil, &labels); err != nil {
		return nil, err
	}

	return connectors.JSONResult(labels)
}
