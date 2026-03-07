package trello

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// createCardAction implements connectors.Action for trello.create_card.
type createCardAction struct {
	conn *TrelloConnector
}

type createCardParams struct {
	ListID   string `json:"list_id"`
	Name     string `json:"name"`
	Desc     string `json:"desc"`
	Pos      string `json:"pos"`
	Due      string `json:"due"`
	IDMembers string `json:"idMembers"`
	IDLabels  string `json:"idLabels"`
}

func (p *createCardParams) validate() error {
	if p.ListID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: list_id"}
	}
	if p.Name == "" {
		return &connectors.ValidationError{Message: "missing required parameter: name"}
	}
	return nil
}

// Execute creates a new card in a Trello list via POST /1/cards.
func (a *createCardAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createCardParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := map[string]any{
		"idList": params.ListID,
		"name":   params.Name,
	}
	if params.Desc != "" {
		body["desc"] = params.Desc
	}
	if params.Pos != "" {
		body["pos"] = params.Pos
	}
	if params.Due != "" {
		body["due"] = params.Due
	}
	if params.IDMembers != "" {
		body["idMembers"] = params.IDMembers
	}
	if params.IDLabels != "" {
		body["idLabels"] = params.IDLabels
	}

	var resp struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		ShortURL string `json:"shortUrl"`
		URL      string `json:"url"`
	}
	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, "/cards", body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
