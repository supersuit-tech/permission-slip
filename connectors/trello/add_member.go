package trello

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

type addMemberAction struct {
	conn *TrelloConnector
}

type addMemberParams struct {
	CardID   string `json:"card_id"`
	MemberID string `json:"member_id"`
}

func (p *addMemberParams) validate() error {
	if err := validateTrelloID(p.CardID, "card_id"); err != nil {
		return err
	}
	if err := validateTrelloID(p.MemberID, "member_id"); err != nil {
		return err
	}
	return nil
}

func (a *addMemberAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params addMemberParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := map[string]any{
		"value": params.MemberID,
	}

	// Trello returns the updated list of all member IDs on the card.
	var memberIDs []string
	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, "/cards/"+params.CardID+"/idMembers", body, &memberIDs); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]any{
		"card_id":    params.CardID,
		"member_id":  params.MemberID,
		"status":     "added",
		"id_members": memberIDs,
	})
}
