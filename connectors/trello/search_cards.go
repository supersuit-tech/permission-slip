package trello

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// searchCardsAction implements connectors.Action for trello.search_cards.
type searchCardsAction struct {
	conn *TrelloConnector
}

type searchCardsParams struct {
	Query   string `json:"query"`
	BoardID string `json:"board_id"`
	ListID  string `json:"list_id"`
	Members string `json:"members"`
	Due     string `json:"due"`
	Limit   int    `json:"limit"`
}

func (p *searchCardsParams) validate() error {
	if p.Query == "" {
		return &connectors.ValidationError{Message: "missing required parameter: query"}
	}
	if p.Limit < 0 || p.Limit > 1000 {
		return &connectors.ValidationError{Message: "limit must be between 0 and 1000"}
	}
	return nil
}

// Execute searches for cards across boards via GET /1/search.
func (a *searchCardsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params searchCardsParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	// Build the search query string. Trello's search supports inline
	// modifiers like @member, list:name, and due:day — we append these
	// to the user's query so they compose naturally.
	query := params.Query
	if params.Members != "" {
		query += " @" + params.Members
	}
	if params.ListID != "" {
		query += " list:" + params.ListID
	}
	if params.Due != "" {
		query += " due:" + params.Due
	}

	qp := map[string]string{
		"query":      query,
		"modelTypes": "cards",
		"card_fields": "id,name,desc,idList,shortUrl,url,due,dueComplete,labels,idMembers,closed",
	}

	limit := params.Limit
	if limit == 0 {
		limit = 10
	}
	qp["cards_limit"] = strconv.Itoa(limit)

	if params.BoardID != "" {
		qp["idBoards"] = params.BoardID
	}

	var resp struct {
		Cards []json.RawMessage `json:"cards"`
	}
	if err := a.conn.doGet(ctx, req.Credentials, "/search", qp, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]any{
		"cards": resp.Cards,
		"count": len(resp.Cards),
	})
}
