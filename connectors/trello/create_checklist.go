package trello

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// createChecklistAction implements connectors.Action for trello.create_checklist.
type createChecklistAction struct {
	conn *TrelloConnector
}

type createChecklistParams struct {
	CardID string   `json:"card_id"`
	Name   string   `json:"name"`
	Items  []string `json:"items"`
}

func (p *createChecklistParams) validate() error {
	if err := validateTrelloID(p.CardID, "card_id"); err != nil {
		return err
	}
	if p.Name == "" {
		return &connectors.ValidationError{Message: "missing required parameter: name"}
	}
	return nil
}

// Execute creates a checklist on a card via POST /1/checklists, then adds
// items via POST /1/checklists/{id}/checkItems for each item.
func (a *createChecklistAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createChecklistParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	// Create the checklist.
	checklistBody := map[string]string{
		"idCard": params.CardID,
		"name":   params.Name,
	}

	var checklistResp struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, "/checklists", checklistBody, &checklistResp); err != nil {
		return nil, err
	}

	// Add items to the checklist. Initialize to empty slice so JSON
	// serialization produces [] instead of null when no items are provided.
	addedItems := make([]map[string]string, 0, len(params.Items))
	for _, item := range params.Items {
		itemBody := map[string]string{
			"name": item,
		}
		var itemResp struct {
			ID   string `json:"id"`
			Name string `json:"name"`
		}
		path := fmt.Sprintf("/checklists/%s/checkItems", url.PathEscape(checklistResp.ID))
		if err := a.conn.do(ctx, req.Credentials, http.MethodPost, path, itemBody, &itemResp); err != nil {
			return nil, err
		}
		addedItems = append(addedItems, map[string]string{
			"id":   itemResp.ID,
			"name": itemResp.Name,
		})
	}

	return connectors.JSONResult(map[string]any{
		"id":    checklistResp.ID,
		"name":  checklistResp.Name,
		"items": addedItems,
	})
}
