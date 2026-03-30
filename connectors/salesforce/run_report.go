package salesforce

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// runReportAction implements connectors.Action for salesforce.run_report.
type runReportAction struct {
	conn *SalesforceConnector
}

type runReportParams struct {
	ReportID       string `json:"report_id"`
	IncludeDetails bool   `json:"include_details,omitempty"`
}

func (p *runReportParams) validate() error {
	if p.ReportID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: report_id"}
	}
	if err := validateRecordID(p.ReportID, "report_id"); err != nil {
		return err
	}
	return nil
}

func (a *runReportAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params runReportParams
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

	apiURL := baseURL + "/analytics/reports/" + url.PathEscape(params.ReportID)
	if params.IncludeDetails {
		apiURL += "?includeDetails=true"
	}

	var result json.RawMessage
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodGet, apiURL, nil, &result); err != nil {
		return nil, err
	}

	return &connectors.ActionResult{Data: result}, nil
}
