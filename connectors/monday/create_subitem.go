package monday

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// createSubitemAction implements connectors.Action for monday.create_subitem.
type createSubitemAction struct {
	conn *MondayConnector
}

type createSubitemParams struct {
	ParentItemID string         `json:"parent_item_id"`
	ItemName     string         `json:"item_name"`
	ColumnValues map[string]any `json:"column_values,omitempty"`
}

func (p *createSubitemParams) validate() error {
	if p.ParentItemID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: parent_item_id"}
	}
	if p.ItemName == "" {
		return &connectors.ValidationError{Message: "missing required parameter: item_name"}
	}
	return nil
}

func (a *createSubitemAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createSubitemParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	query := `mutation ($parent_item_id: ID!, $item_name: String!, $column_values: JSON) {
		create_subitem(parent_item_id: $parent_item_id, item_name: $item_name, column_values: $column_values) {
			id
			name
		}
	}`

	variables := map[string]any{
		"parent_item_id": params.ParentItemID,
		"item_name":      params.ItemName,
	}
	if params.ColumnValues != nil {
		cv, err := json.Marshal(params.ColumnValues)
		if err != nil {
			return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid column_values: %v", err)}
		}
		variables["column_values"] = string(cv)
	}

	var data struct {
		CreateSubitem struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"create_subitem"`
	}
	if err := a.conn.query(ctx, req.Credentials, query, variables, &data); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]string{
		"id":   data.CreateSubitem.ID,
		"name": data.CreateSubitem.Name,
	})
}
