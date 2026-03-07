package google

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// sheetsReadRangeAction implements connectors.Action for google.sheets_read_range.
// It reads cell values from a specified range via the Google Sheets API
// GET /v4/spreadsheets/{id}/values/{range}.
type sheetsReadRangeAction struct {
	conn *GoogleConnector
}

// sheetsReadRangeParams is the user-facing parameter schema.
type sheetsReadRangeParams struct {
	SpreadsheetID string `json:"spreadsheet_id"`
	Range         string `json:"range"`
}

// validate checks that required fields are present.
func (p *sheetsReadRangeParams) validate() error {
	if p.SpreadsheetID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: spreadsheet_id"}
	}
	if p.Range == "" {
		return &connectors.ValidationError{Message: "missing required parameter: range (e.g. 'Sheet1!A1:D10')"}
	}
	return nil
}

// sheetsValueRange is the Google Sheets API response for values.get.
type sheetsValueRange struct {
	Range          string     `json:"range"`
	MajorDimension string    `json:"majorDimension"`
	Values         [][]any   `json:"values"`
}

// Execute reads cell values from the specified range in a Google Sheets spreadsheet.
func (a *sheetsReadRangeAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params sheetsReadRangeParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	readURL := a.conn.sheetsBaseURL + "/spreadsheets/" +
		url.PathEscape(params.SpreadsheetID) + "/values/" +
		url.PathEscape(params.Range)

	var resp sheetsValueRange
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodGet, readURL, nil, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]any{
		"range":  resp.Range,
		"values": resp.Values,
	})
}
