package monday

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// createItemAction implements connectors.Action for monday.create_item.
type createItemAction struct {
	conn *MondayConnector
}

type createItemParams struct {
	BoardID      string         `json:"board_id"`
	ItemName     string         `json:"item_name"`
	ColumnValues map[string]any `json:"column_values,omitempty"`
	GroupID      string         `json:"group_id,omitempty"`
}

func (p *createItemParams) validate() error {
	if p.BoardID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: board_id"}
	}
	if !isValidMondayID(p.BoardID) {
		return &connectors.ValidationError{Message: "board_id must be a numeric string"}
	}
	if p.ItemName == "" {
		return &connectors.ValidationError{Message: "missing required parameter: item_name"}
	}
	return nil
}

func (a *createItemAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createItemParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	// Build mutation. column_values must be a stringified JSON object.
	query := `mutation ($board_id: ID!, $item_name: String!, $column_values: JSON, $group_id: String) {
		create_item(board_id: $board_id, item_name: $item_name, column_values: $column_values, group_id: $group_id) {
			id
			name
		}
	}`

	variables := map[string]any{
		"board_id":  params.BoardID,
		"item_name": params.ItemName,
	}
	if params.ColumnValues != nil {
		cv, err := stringifyColumnValues(params.ColumnValues)
		if err != nil {
			return nil, err
		}
		variables["column_values"] = cv
	}
	if params.GroupID != "" {
		variables["group_id"] = params.GroupID
	}

	var data struct {
		CreateItem struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"create_item"`
	}
	if err := a.conn.query(ctx, req.Credentials, query, variables, &data); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]string{
		"id":   data.CreateItem.ID,
		"name": data.CreateItem.Name,
	})
}
