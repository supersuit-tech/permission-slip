package meta

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// createAdCampaignAction implements connectors.Action for meta.create_ad_campaign.
// It creates a Facebook/Instagram ad campaign via POST /act_{ad_account_id}/campaigns.
type createAdCampaignAction struct {
	conn *MetaConnector
}

type createAdCampaignParams struct {
	AdAccountID string `json:"ad_account_id"`
	Name        string `json:"name"`
	Objective   string `json:"objective"`
	Status      string `json:"status,omitempty"`
	BudgetType  string `json:"budget_type,omitempty"`
	DailyBudget int64  `json:"daily_budget,omitempty"`
	LifetimeBudget int64 `json:"lifetime_budget,omitempty"`
}

var validCampaignObjectives = map[string]bool{
	"OUTCOME_AWARENESS":     true,
	"OUTCOME_ENGAGEMENT":    true,
	"OUTCOME_LEADS":         true,
	"OUTCOME_SALES":         true,
	"OUTCOME_TRAFFIC":       true,
	"OUTCOME_APP_PROMOTION": true,
}

var validCampaignStatuses = map[string]bool{
	"ACTIVE":   true,
	"PAUSED":   true,
	"ARCHIVED": true,
}

var validBudgetTypes = map[string]bool{
	"DAILY":    true,
	"LIFETIME": true,
}

func (p *createAdCampaignParams) validate() error {
	if p.AdAccountID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: ad_account_id"}
	}
	if !isValidGraphID(p.AdAccountID) {
		return &connectors.ValidationError{Message: "ad_account_id contains invalid characters"}
	}
	if p.Name == "" {
		return &connectors.ValidationError{Message: "missing required parameter: name"}
	}
	if p.Objective == "" {
		return &connectors.ValidationError{Message: "missing required parameter: objective"}
	}
	if !validCampaignObjectives[p.Objective] {
		return &connectors.ValidationError{
			Message: "invalid objective — must be one of: OUTCOME_AWARENESS, OUTCOME_ENGAGEMENT, OUTCOME_LEADS, OUTCOME_SALES, OUTCOME_TRAFFIC, OUTCOME_APP_PROMOTION",
		}
	}
	if p.Status != "" && !validCampaignStatuses[p.Status] {
		return &connectors.ValidationError{Message: "invalid status — must be one of: ACTIVE, PAUSED, ARCHIVED"}
	}
	if p.BudgetType != "" && !validBudgetTypes[p.BudgetType] {
		return &connectors.ValidationError{Message: "invalid budget_type — must be one of: DAILY, LIFETIME"}
	}
	if p.DailyBudget < 0 {
		return &connectors.ValidationError{Message: "daily_budget must be non-negative"}
	}
	if p.LifetimeBudget < 0 {
		return &connectors.ValidationError{Message: "lifetime_budget must be non-negative"}
	}
	// Prevent conflicting budget types: daily_budget and lifetime_budget are mutually exclusive.
	if p.DailyBudget > 0 && p.LifetimeBudget > 0 {
		return &connectors.ValidationError{Message: "provide either daily_budget or lifetime_budget, not both"}
	}
	return nil
}

type createAdCampaignRequest struct {
	Name                string   `json:"name"`
	Objective           string   `json:"objective"`
	Status              string   `json:"status,omitempty"`
	DailyBudget         int64    `json:"daily_budget,omitempty"`
	LifetimeBudget      int64    `json:"lifetime_budget,omitempty"`
	SpecialAdCategories []string `json:"special_ad_categories"`
}

type createAdCampaignResponse struct {
	ID string `json:"id"`
}

func (a *createAdCampaignAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createAdCampaignParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	status := params.Status
	if status == "" {
		status = "PAUSED"
	}

	body := createAdCampaignRequest{
		Name:                params.Name,
		Objective:           params.Objective,
		Status:              status,
		DailyBudget:         params.DailyBudget,
		LifetimeBudget:      params.LifetimeBudget,
		SpecialAdCategories: []string{},
	}

	var resp createAdCampaignResponse
	reqURL := fmt.Sprintf("%s/act_%s/campaigns", a.conn.baseURL, params.AdAccountID)
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodPost, reqURL, body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]string{
		"id": resp.ID,
	})
}
