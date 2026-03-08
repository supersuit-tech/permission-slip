package trello

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

type deleteCardAction struct {
	conn *TrelloConnector
}

type deleteCardParams struct {
	CardID string `json:"card_id"`
}

func (p *deleteCardParams) validate() error {
	return validateTrelloID(p.CardID, "card_id")
}

func (a *deleteCardAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params deleteCardParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	if err := a.conn.do(ctx, req.Credentials, http.MethodDelete, "/cards/"+params.CardID, nil, nil); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]string{
		"card_id": params.CardID,
		"status":  "deleted",
	})
}
