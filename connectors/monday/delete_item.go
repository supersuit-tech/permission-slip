package monday

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

type deleteItemAction struct {
	conn *MondayConnector
}

type deleteItemParams struct {
	ItemID string `json:"item_id"`
}

func (p *deleteItemParams) validate() error {
	if p.ItemID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: item_id"}
	}
	if !isValidMondayID(p.ItemID) {
		return &connectors.ValidationError{Message: "item_id must be a numeric string"}
	}
	return nil
}

func (a *deleteItemAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params deleteItemParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	query := `mutation ($item_id: ID!) {
		delete_item(item_id: $item_id) {
			id
		}
	}`

	variables := map[string]any{
		"item_id": params.ItemID,
	}

	var data struct {
		DeleteItem struct {
			ID string `json:"id"`
		} `json:"delete_item"`
	}

	if err := a.conn.query(ctx, req.Credentials, query, variables, &data); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]string{
		"item_id": data.DeleteItem.ID,
		"status":  "deleted",
	})
}
