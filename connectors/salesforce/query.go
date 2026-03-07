package salesforce

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	neturl "net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// queryAction implements connectors.Action for salesforce.query.
type queryAction struct {
	conn *SalesforceConnector
}

type queryParams struct {
	SOQL       string `json:"soql"`
	MaxRecords int    `json:"max_records"`
}

const (
	defaultMaxRecords = 200
	maxMaxRecords     = 2000
)

func (p *queryParams) validate() error {
	if p.SOQL == "" {
		return &connectors.ValidationError{Message: "missing required parameter: soql"}
	}
	if p.MaxRecords > maxMaxRecords {
		return &connectors.ValidationError{
			Message: fmt.Sprintf("max_records must be at most %d, got %d", maxMaxRecords, p.MaxRecords),
		}
	}
	return nil
}

// sfQueryResponse is the Salesforce SOQL query response.
type sfQueryResponse struct {
	TotalSize int               `json:"totalSize"`
	Done      bool              `json:"done"`
	Records   []json.RawMessage `json:"records"`
}

func (a *queryAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params queryParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	if params.MaxRecords <= 0 {
		params.MaxRecords = defaultMaxRecords
	}

	baseURL, err := a.conn.apiBaseURL(req.Credentials)
	if err != nil {
		return nil, err
	}

	apiURL := baseURL + "/query/?q=" + neturl.QueryEscape(params.SOQL)

	var resp sfQueryResponse
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodGet, apiURL, nil, &resp); err != nil {
		return nil, err
	}

	// Truncate records to max_records and indicate when results were cut.
	records := resp.Records
	truncated := len(records) > params.MaxRecords
	if truncated {
		records = records[:params.MaxRecords]
	}

	return connectors.JSONResult(map[string]any{
		"total_size": resp.TotalSize,
		"done":       resp.Done,
		"records":    records,
		"truncated":  truncated,
	})
}
