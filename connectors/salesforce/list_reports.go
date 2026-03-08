package salesforce

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// listReportsAction implements connectors.Action for salesforce.list_reports.
type listReportsAction struct {
	conn *SalesforceConnector
}

// sfReportListItem represents a single report in the list response.
type sfReportListItem struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	FolderName  string `json:"folderName"`
	URL         string `json:"url"`
	Description string `json:"description"`
}

func (a *listReportsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	// No parameters required
	if len(req.Parameters) > 0 && string(req.Parameters) != "null" && string(req.Parameters) != "{}" {
		var check map[string]any
		if err := json.Unmarshal(req.Parameters, &check); err != nil {
			return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
		}
	}

	baseURL, err := a.conn.apiBaseURL(req.Credentials)
	if err != nil {
		return nil, err
	}

	// The analytics reports endpoint is at the same API version base path.
	apiURL := baseURL + "/analytics/reports"

	var reports []sfReportListItem
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodGet, apiURL, nil, &reports); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]any{
		"reports": reports,
		"count":   len(reports),
	})
}
