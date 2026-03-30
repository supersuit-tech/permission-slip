package hubspot

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// updateDealStageAction implements connectors.Action for hubspot.update_deal_stage.
// It moves a deal to a different pipeline stage via PATCH /crm/v3/objects/deals/{dealId}.
type updateDealStageAction struct {
	conn *HubSpotConnector
}

type updateDealStageParams struct {
	DealID        string `json:"deal_id"`
	PipelineStage string `json:"pipeline_stage"`
	CloseDate     string `json:"close_date"`
}

func (p *updateDealStageParams) validate() error {
	if p.DealID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: deal_id"}
	}
	if !isValidHubSpotID(p.DealID) {
		return &connectors.ValidationError{Message: "deal_id must be a numeric HubSpot ID"}
	}
	if p.PipelineStage == "" {
		return &connectors.ValidationError{Message: "missing required parameter: pipeline_stage"}
	}
	return nil
}

func (a *updateDealStageAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params updateDealStageParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	props := map[string]string{
		"dealstage": params.PipelineStage,
	}
	if params.CloseDate != "" {
		props["closedate"] = params.CloseDate
	}

	body := hubspotObjectRequest{Properties: props}
	var resp hubspotObjectResponse
	path := fmt.Sprintf("/crm/v3/objects/deals/%s", params.DealID)
	if err := a.conn.do(ctx, req.Credentials, http.MethodPatch, path, body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
