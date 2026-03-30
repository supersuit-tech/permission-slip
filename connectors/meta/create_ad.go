package meta

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// createAdAction implements connectors.Action for meta.create_ad.
// It creates a Facebook/Instagram ad within an ad set via POST /act_{ad_account_id}/ads.
type createAdAction struct {
	conn *MetaConnector
}

type createAdParams struct {
	AdAccountID  string `json:"ad_account_id"`
	Name         string `json:"name"`
	AdSetID      string `json:"adset_id"`
	CreativeID   string `json:"creative_id"`
	Status       string `json:"status,omitempty"`
}

var validAdStatuses = map[string]bool{
	"ACTIVE":   true,
	"PAUSED":   true,
	"ARCHIVED": true,
}

func (p *createAdParams) validate() error {
	if p.AdAccountID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: ad_account_id"}
	}
	if !isValidGraphID(p.AdAccountID) {
		return &connectors.ValidationError{Message: "ad_account_id contains invalid characters"}
	}
	if p.Name == "" {
		return &connectors.ValidationError{Message: "missing required parameter: name"}
	}
	if p.AdSetID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: adset_id"}
	}
	if p.CreativeID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: creative_id"}
	}
	if p.Status != "" && !validAdStatuses[p.Status] {
		return &connectors.ValidationError{Message: "invalid status — must be one of: ACTIVE, PAUSED, ARCHIVED"}
	}
	return nil
}

type createAdRequest struct {
	Name     string                 `json:"name"`
	AdSetID  string                 `json:"adset_id"`
	Creative map[string]interface{} `json:"creative"`
	Status   string                 `json:"status"`
}

type createAdResponse struct {
	ID string `json:"id"`
}

func (a *createAdAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createAdParams
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

	body := createAdRequest{
		Name:    params.Name,
		AdSetID: params.AdSetID,
		Creative: map[string]interface{}{
			"creative_id": params.CreativeID,
		},
		Status: status,
	}

	var resp createAdResponse
	reqURL := fmt.Sprintf("%s/act_%s/ads", a.conn.baseURL, params.AdAccountID)
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodPost, reqURL, body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]string{
		"id": resp.ID,
	})
}
