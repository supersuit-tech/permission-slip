package monday

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// updateItemAction implements connectors.Action for monday.update_item.
type updateItemAction struct {
	conn *MondayConnector
}

type updateItemParams struct {
	BoardID      string         `json:"board_id"`
	ItemID       string         `json:"item_id"`
	ColumnValues map[string]any `json:"column_values"`
}

func (p *updateItemParams) validate() error {
	if p.BoardID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: board_id"}
	}
	if !isValidMondayID(p.BoardID) {
		return &connectors.ValidationError{Message: "board_id must be a numeric string"}
	}
	if p.ItemID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: item_id"}
	}
	if !isValidMondayID(p.ItemID) {
		return &connectors.ValidationError{Message: "item_id must be a numeric string"}
	}
	if p.ColumnValues == nil {
		return &connectors.ValidationError{Message: "missing required parameter: column_values"}
	}
	return nil
}

func (a *updateItemAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params updateItemParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	query := `mutation ($board_id: ID!, $item_id: ID!, $column_values: JSON!) {
		change_multiple_column_values(board_id: $board_id, item_id: $item_id, column_values: $column_values) {
			id
			name
		}
	}`

	cv, err := stringifyColumnValues(params.ColumnValues)
	if err != nil {
		return nil, err
	}

	variables := map[string]any{
		"board_id":      params.BoardID,
		"item_id":       params.ItemID,
		"column_values": cv,
	}

	var data struct {
		ChangeMultipleColumnValues struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"change_multiple_column_values"`
	}
	if err := a.conn.query(ctx, req.Credentials, query, variables, &data); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]string{
		"id":   data.ChangeMultipleColumnValues.ID,
		"name": data.ChangeMultipleColumnValues.Name,
	})
}
