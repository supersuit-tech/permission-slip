package trello

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// addCommentAction implements connectors.Action for trello.add_comment.
type addCommentAction struct {
	conn *TrelloConnector
}

type addCommentParams struct {
	CardID string `json:"card_id"`
	Text   string `json:"text"`
}

func (p *addCommentParams) validate() error {
	if p.CardID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: card_id"}
	}
	if p.Text == "" {
		return &connectors.ValidationError{Message: "missing required parameter: text"}
	}
	return nil
}

// Execute adds a comment to a Trello card via POST /1/cards/{card_id}/actions/comments.
func (a *addCommentAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params addCommentParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := map[string]string{
		"text": params.Text,
	}

	var resp struct {
		ID   string `json:"id"`
		Type string `json:"type"`
		Data struct {
			Text string `json:"text"`
		} `json:"data"`
	}
	path := fmt.Sprintf("/cards/%s/actions/comments", params.CardID)
	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, path, body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]string{
		"id":   resp.ID,
		"text": resp.Data.Text,
	})
}
