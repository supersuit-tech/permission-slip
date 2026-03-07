package trello

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// moveCardAction implements connectors.Action for trello.move_card.
// This is a thin wrapper over PUT /1/cards/{card_id} that exists as a
// separate action for explicit permission gating, since moving a card
// between lists represents a workflow state change.
type moveCardAction struct {
	conn *TrelloConnector
}

type moveCardParams struct {
	CardID string `json:"card_id"`
	ListID string `json:"list_id"`
	Pos    string `json:"pos"`
}

func (p *moveCardParams) validate() error {
	if err := validateTrelloID(p.CardID, "card_id"); err != nil {
		return err
	}
	if err := validateTrelloID(p.ListID, "list_id"); err != nil {
		return err
	}
	return nil
}

// Execute moves a card to a different list via PUT /1/cards/{card_id}.
func (a *moveCardAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params moveCardParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := map[string]any{
		"idList": params.ListID,
	}
	if params.Pos != "" {
		body["pos"] = params.Pos
	}

	var resp struct {
		ID       string `json:"id"`
		Name     string `json:"name"`
		IDList   string `json:"idList"`
		ShortURL string `json:"shortUrl"`
		URL      string `json:"url"`
	}
	path := fmt.Sprintf("/cards/%s", url.PathEscape(params.CardID))
	if err := a.conn.do(ctx, req.Credentials, http.MethodPut, path, body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
