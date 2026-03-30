package monday

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// addUpdateAction implements connectors.Action for monday.add_update.
type addUpdateAction struct {
	conn *MondayConnector
}

type addUpdateParams struct {
	ItemID string `json:"item_id"`
	Body   string `json:"body"`
}

func (p *addUpdateParams) validate() error {
	if p.ItemID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: item_id"}
	}
	if !isValidMondayID(p.ItemID) {
		return &connectors.ValidationError{Message: "item_id must be a numeric string"}
	}
	if p.Body == "" {
		return &connectors.ValidationError{Message: "missing required parameter: body"}
	}
	return nil
}

func (a *addUpdateAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params addUpdateParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	query := `mutation ($item_id: ID!, $body: String!) {
		create_update(item_id: $item_id, body: $body) {
			id
			body
		}
	}`

	variables := map[string]any{
		"item_id": params.ItemID,
		"body":    params.Body,
	}

	var data struct {
		CreateUpdate struct {
			ID   string `json:"id"`
			Body string `json:"body"`
		} `json:"create_update"`
	}
	if err := a.conn.query(ctx, req.Credentials, query, variables, &data); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]string{
		"id":   data.CreateUpdate.ID,
		"body": data.CreateUpdate.Body,
	})
}
