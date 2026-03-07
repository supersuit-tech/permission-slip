package shopify

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// getAnalyticsAction implements connectors.Action for shopify.get_analytics.
// It retrieves shop reports via GET /admin/api/2024-10/reports.json.
// Note: Analytics API access varies by Shopify plan — not all plans support it.
type getAnalyticsAction struct {
	conn *ShopifyConnector
}

// getAnalyticsParams maps the JSON parameters for the get_analytics action.
type getAnalyticsParams struct {
	Since  string `json:"since,omitempty"`
	Until  string `json:"until,omitempty"`
	Fields string `json:"fields,omitempty"`
}

func (p *getAnalyticsParams) validate() error {
	// All parameters are optional for this read-only action.
	return nil
}

// Execute retrieves shop analytics reports from the Shopify API.
func (a *getAnalyticsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params getAnalyticsParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	q := url.Values{}
	if params.Since != "" {
		q.Set("since", params.Since)
	}
	if params.Until != "" {
		q.Set("until", params.Until)
	}
	if params.Fields != "" {
		q.Set("fields", params.Fields)
	}

	path := "/reports.json"
	if len(q) > 0 {
		path += "?" + q.Encode()
	}

	var resp struct {
		Reports json.RawMessage `json:"reports"`
	}
	if err := a.conn.do(ctx, req.Credentials, http.MethodGet, path, nil, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(resp)
}
