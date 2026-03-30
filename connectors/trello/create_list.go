package trello

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip/connectors"
)

type createListAction struct {
	conn *TrelloConnector
}

type createListParams struct {
	BoardID string `json:"board_id"`
	Name    string `json:"name"`
	Pos     string `json:"pos"`
}

func (p *createListParams) validate() error {
	if err := validateTrelloID(p.BoardID, "board_id"); err != nil {
		return err
	}
	if p.Name == "" {
		return &connectors.ValidationError{Message: "missing required parameter: name"}
	}
	return nil
}

func (a *createListAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createListParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := map[string]any{
		"idBoard": params.BoardID,
		"name":    params.Name,
	}
	if params.Pos != "" {
		body["pos"] = params.Pos
	}

	var resp struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Closed  bool   `json:"closed"`
		IDBoard string `json:"idBoard"`
	}
	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, "/lists", body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
