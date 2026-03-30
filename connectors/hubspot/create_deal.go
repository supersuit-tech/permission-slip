package hubspot

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// createDealAction implements connectors.Action for hubspot.create_deal.
// It creates a new deal via POST /crm/v3/objects/deals, then optionally
// associates it with contacts.
type createDealAction struct {
	conn *HubSpotConnector
}

type createDealParams struct {
	DealName           string            `json:"dealname"`
	Pipeline           string            `json:"pipeline"`
	DealStage          string            `json:"dealstage"`
	Amount             string            `json:"amount"`
	CloseDate          string            `json:"closedate"`
	AssociatedContacts []string          `json:"associated_contacts"`
	Properties         map[string]string `json:"properties"`
}

func (p *createDealParams) validate() error {
	if p.DealName == "" {
		return &connectors.ValidationError{Message: "missing required parameter: dealname"}
	}
	if p.Pipeline == "" {
		return &connectors.ValidationError{Message: "missing required parameter: pipeline"}
	}
	if p.DealStage == "" {
		return &connectors.ValidationError{Message: "missing required parameter: dealstage"}
	}
	if len(p.AssociatedContacts) > maxAssociations {
		return &connectors.ValidationError{Message: fmt.Sprintf("associated_contacts exceeds maximum of %d", maxAssociations)}
	}
	for i, id := range p.AssociatedContacts {
		if !isValidHubSpotID(id) {
			return &connectors.ValidationError{Message: fmt.Sprintf("associated_contacts[%d]: must be a numeric HubSpot ID", i)}
		}
	}
	return nil
}

func (p *createDealParams) toAPIProperties() map[string]string {
	overrides := map[string]string{
		"dealname":  p.DealName,
		"pipeline":  p.Pipeline,
		"dealstage": p.DealStage,
	}
	if p.Amount != "" {
		overrides["amount"] = p.Amount
	}
	if p.CloseDate != "" {
		overrides["closedate"] = p.CloseDate
	}
	return mergeProperties(p.Properties, overrides)
}

func (a *createDealAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createDealParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := hubspotObjectRequest{Properties: params.toAPIProperties()}
	var resp hubspotObjectResponse
	if err := a.conn.do(ctx, req.Credentials, http.MethodPost, "/crm/v3/objects/deals", body, &resp); err != nil {
		return nil, err
	}

	// Associate with contacts if specified. Validate the API-returned ID
	// as defense-in-depth before interpolating it into subsequent URL paths.
	if len(params.AssociatedContacts) > 0 && !isValidHubSpotID(resp.ID) {
		return nil, fmt.Errorf("deal created but got unexpected non-numeric id %q from HubSpot", resp.ID)
	}
	for _, contactID := range params.AssociatedContacts {
		path := fmt.Sprintf("/crm/v3/objects/deals/%s/associations/contacts/%s/deal_to_contact", resp.ID, contactID)
		if err := a.conn.do(ctx, req.Credentials, http.MethodPut, path, nil, nil); err != nil {
			return nil, fmt.Errorf("deal created (id=%s) but association with contact %s failed: %w", resp.ID, contactID, err)
		}
	}

	return connectors.JSONResult(resp)
}
