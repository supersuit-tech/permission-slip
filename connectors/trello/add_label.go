package trello

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip/connectors"
)

type addLabelAction struct {
	conn *TrelloConnector
}

type addLabelParams struct {
	CardID  string `json:"card_id"`
	LabelID string `json:"label_id"`
}

func (p *addLabelParams) validate() error {
	if err := validateTrelloID(p.CardID, "card_id"); err != nil {
		return err
	}
	if err := validateTrelloID(p.LabelID, "label_id"); err != nil {
		return err
	}
	return nil
}

func (a *addLabelAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params addLabelParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := map[string]any{
		"value": params.LabelID,
	}

	// Trello returns the updated list of all label IDs on the card.
	var labelIDs []string
	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, "/cards/"+params.CardID+"/idLabels", body, &labelIDs); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]any{
		"card_id":   params.CardID,
		"label_id":  params.LabelID,
		"status":    "added",
		"id_labels": labelIDs,
	})
}
