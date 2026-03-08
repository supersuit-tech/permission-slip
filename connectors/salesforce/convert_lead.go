package salesforce

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// convertLeadAction implements connectors.Action for salesforce.convert_lead.
type convertLeadAction struct {
	conn *SalesforceConnector
}

type convertLeadParams struct {
	LeadID                 string `json:"lead_id"`
	ConvertedStatus        string `json:"converted_status"`
	AccountID              string `json:"account_id,omitempty"`
	ContactID              string `json:"contact_id,omitempty"`
	OpportunityName        string `json:"opportunity_name,omitempty"`
	DoNotCreateOpportunity bool   `json:"do_not_create_opportunity,omitempty"`
	OwnerID                string `json:"owner_id,omitempty"`
}

func (p *convertLeadParams) validate() error {
	if p.LeadID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: lead_id"}
	}
	if err := validateRecordID(p.LeadID, "lead_id"); err != nil {
		return err
	}
	if p.ConvertedStatus == "" {
		return &connectors.ValidationError{Message: "missing required parameter: converted_status"}
	}
	if p.AccountID != "" {
		if err := validateRecordID(p.AccountID, "account_id"); err != nil {
			return err
		}
	}
	if p.ContactID != "" {
		if err := validateRecordID(p.ContactID, "contact_id"); err != nil {
			return err
		}
	}
	if p.OwnerID != "" {
		if err := validateRecordID(p.OwnerID, "owner_id"); err != nil {
			return err
		}
	}
	return nil
}

// sfConvertLeadRequest is the request body for the Salesforce Lead convert endpoint.
type sfConvertLeadRequest struct {
	LeadID                 string `json:"leadId"`
	ConvertedStatus        string `json:"convertedStatus"`
	AccountID              string `json:"accountId,omitempty"`
	ContactID              string `json:"contactId,omitempty"`
	OpportunityName        string `json:"opportunityName,omitempty"`
	DoNotCreateOpportunity bool   `json:"doNotCreateOpportunity,omitempty"`
	OwnerID                string `json:"ownerId,omitempty"`
}

// sfConvertLeadResponse is the response body from the Salesforce Lead convert endpoint.
type sfConvertLeadResponse struct {
	AccountID     string `json:"accountId"`
	ContactID     string `json:"contactId"`
	LeadID        string `json:"leadId"`
	OpportunityID string `json:"opportunityId"`
	Success       bool   `json:"success"`
}

func (a *convertLeadAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params convertLeadParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	baseURL, err := a.conn.apiBaseURL(req.Credentials)
	if err != nil {
		return nil, err
	}

	// The convert endpoint is under the Lead sObject.
	apiURL := baseURL + "/sobjects/Lead/" + url.PathEscape(params.LeadID) + "/convert"

	body := sfConvertLeadRequest{
		LeadID:                 params.LeadID,
		ConvertedStatus:        params.ConvertedStatus,
		AccountID:              params.AccountID,
		ContactID:              params.ContactID,
		OpportunityName:        params.OpportunityName,
		DoNotCreateOpportunity: params.DoNotCreateOpportunity,
		OwnerID:                params.OwnerID,
	}

	var resp sfConvertLeadResponse
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodPost, apiURL, body, &resp); err != nil {
		return nil, err
	}

	result := map[string]any{
		"lead_id": resp.LeadID,
		"success": resp.Success,
	}
	// Only include IDs that were actually created/used (non-empty).
	if resp.AccountID != "" {
		result["account_id"] = resp.AccountID
		if u := recordURL(req.Credentials, resp.AccountID); u != "" {
			result["account_url"] = u
		}
	}
	if resp.ContactID != "" {
		result["contact_id"] = resp.ContactID
		if u := recordURL(req.Credentials, resp.ContactID); u != "" {
			result["contact_url"] = u
		}
	}
	if resp.OpportunityID != "" {
		result["opportunity_id"] = resp.OpportunityID
		if u := recordURL(req.Credentials, resp.OpportunityID); u != "" {
			result["opportunity_url"] = u
		}
	}
	return connectors.JSONResult(result)
}
