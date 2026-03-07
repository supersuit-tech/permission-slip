package trello

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// updateCardAction implements connectors.Action for trello.update_card.
type updateCardAction struct {
	conn *TrelloConnector
}

type updateCardParams struct {
	CardID      string `json:"card_id"`
	Name        string `json:"name"`
	Desc        string `json:"desc"`
	Due         string `json:"due"`
	DueComplete *bool  `json:"dueComplete"`
	IDList      string `json:"idList"`
	Pos         string `json:"pos"`
	IDMembers   string `json:"idMembers"`
	IDLabels    string `json:"idLabels"`
	Closed      *bool  `json:"closed"`
}

func (p *updateCardParams) validate() error {
	if err := validateTrelloID(p.CardID, "card_id"); err != nil {
		return err
	}
	return nil
}

// Execute updates fields on an existing Trello card via PUT /1/cards/{card_id}.
func (a *updateCardAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params updateCardParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := map[string]any{}
	if params.Name != "" {
		body["name"] = params.Name
	}
	if params.Desc != "" {
		body["desc"] = params.Desc
	}
	if params.Due != "" {
		body["due"] = params.Due
	}
	if params.DueComplete != nil {
		body["dueComplete"] = *params.DueComplete
	}
	if params.IDList != "" {
		body["idList"] = params.IDList
	}
	if params.Pos != "" {
		body["pos"] = params.Pos
	}
	if params.IDMembers != "" {
		body["idMembers"] = params.IDMembers
	}
	if params.IDLabels != "" {
		body["idLabels"] = params.IDLabels
	}
	if params.Closed != nil {
		body["closed"] = *params.Closed
	}

	var resp struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		ShortURL string `json:"shortUrl"`
		URL      string `json:"url"`
	}
	path := fmt.Sprintf("/cards/%s", params.CardID)
	if err := a.conn.do(ctx, req.Credentials, http.MethodPut, path, body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
