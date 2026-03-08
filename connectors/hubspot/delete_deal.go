package hubspot

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// deleteDealAction implements connectors.Action for hubspot.delete_deal.
// It archives (soft-deletes) a deal via DELETE /crm/v3/objects/deals/{deal_id}.
// HubSpot's "delete" is actually an archive — the record can be restored.
type deleteDealAction struct {
	conn *HubSpotConnector
}

type deleteDealParams struct {
	DealID string `json:"deal_id"`
}

func (p *deleteDealParams) validate() error {
	if p.DealID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: deal_id"}
	}
	if !isValidHubSpotID(p.DealID) {
		return &connectors.ValidationError{Message: "deal_id must be a numeric HubSpot ID"}
	}
	return nil
}

func (a *deleteDealAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params deleteDealParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/crm/v3/objects/deals/%s", params.DealID)
	if err := a.conn.do(ctx, req.Credentials, http.MethodDelete, path, nil, nil); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]any{
		"deal_id":  params.DealID,
		"archived": true,
	})
}
