package trello

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip/connectors"
)

type listBoardsAction struct {
	conn *TrelloConnector
}

type listBoardsParams struct {
	Filter string `json:"filter"`
}

func (a *listBoardsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params listBoardsParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}

	filter := params.Filter
	if filter == "" {
		filter = "open"
	}

	var boards []struct {
		ID        string `json:"id"`
		Name      string `json:"name"`
		Desc      string `json:"desc"`
		Closed    bool   `json:"closed"`
		ShortURL  string `json:"shortUrl"`
		URL       string `json:"url"`
	}

	if err := a.conn.doGet(ctx, req.Credentials, "/members/me/boards", map[string]string{
		"filter": filter,
		"fields": "id,name,desc,closed,shortUrl,url",
	}, &boards); err != nil {
		return nil, err
	}

	return connectors.JSONResult(boards)
}
