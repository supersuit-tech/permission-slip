package monday

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// moveItemToGroupAction implements connectors.Action for monday.move_item_to_group.
type moveItemToGroupAction struct {
	conn *MondayConnector
}

type moveItemToGroupParams struct {
	ItemID  string `json:"item_id"`
	GroupID string `json:"group_id"`
}

func (p *moveItemToGroupParams) validate() error {
	if p.ItemID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: item_id"}
	}
	if !isValidMondayID(p.ItemID) {
		return &connectors.ValidationError{Message: "item_id must be a numeric string"}
	}
	if p.GroupID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: group_id"}
	}
	return nil
}

func (a *moveItemToGroupAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params moveItemToGroupParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	query := `mutation ($item_id: ID!, $group_id: String!) {
		move_item_to_group(item_id: $item_id, group_id: $group_id) {
			id
			name
		}
	}`

	variables := map[string]any{
		"item_id":  params.ItemID,
		"group_id": params.GroupID,
	}

	var data struct {
		MoveItemToGroup struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		} `json:"move_item_to_group"`
	}
	if err := a.conn.query(ctx, req.Credentials, query, variables, &data); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]string{
		"id":   data.MoveItemToGroup.ID,
		"name": data.MoveItemToGroup.Name,
	})
}
