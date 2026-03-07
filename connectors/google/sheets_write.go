package google

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// sheetsWriteRangeAction implements connectors.Action for google.sheets_write_range.
// It writes cell values to a specified range via the Google Sheets API
// PUT /v4/spreadsheets/{id}/values/{range}?valueInputOption=USER_ENTERED.
type sheetsWriteRangeAction struct {
	conn *GoogleConnector
}

// sheetsWriteRangeParams is the user-facing parameter schema.
type sheetsWriteRangeParams struct {
	SpreadsheetID string  `json:"spreadsheet_id"`
	Range         string  `json:"range"`
	Values        [][]any `json:"values"`
}

func (p *sheetsWriteRangeParams) validate() error {
	if p.SpreadsheetID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: spreadsheet_id"}
	}
	if p.Range == "" {
		return &connectors.ValidationError{Message: "missing required parameter: range (e.g. 'Sheet1!A1:C3')"}
	}
	if len(p.Values) == 0 {
		return &connectors.ValidationError{Message: "values must contain at least one row of data"}
	}
	if err := validateRowLengths(p.Values); err != nil {
		return err
	}
	return nil
}

// sheetsUpdateValuesRequest is the request body for values.update.
type sheetsUpdateValuesRequest struct {
	Range          string  `json:"range"`
	MajorDimension string `json:"majorDimension"`
	Values         [][]any `json:"values"`
}

// sheetsUpdateValuesResponse is the Google Sheets API response for values.update.
type sheetsUpdateValuesResponse struct {
	SpreadsheetID  string `json:"spreadsheetId"`
	UpdatedRange   string `json:"updatedRange"`
	UpdatedRows    int    `json:"updatedRows"`
	UpdatedColumns int    `json:"updatedColumns"`
	UpdatedCells   int    `json:"updatedCells"`
}

// Execute writes cell values to the specified range in a Google Sheets spreadsheet.
func (a *sheetsWriteRangeAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params sheetsWriteRangeParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	writeURL := a.conn.sheetsBaseURL + "/spreadsheets/" +
		url.PathEscape(params.SpreadsheetID) + "/values/" +
		url.PathEscape(params.Range) + "?valueInputOption=USER_ENTERED"

	body := sheetsUpdateValuesRequest{
		Range:          params.Range,
		MajorDimension: "ROWS",
		Values:         params.Values,
	}

	var resp sheetsUpdateValuesResponse
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodPut, writeURL, body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]any{
		"updated_range":   resp.UpdatedRange,
		"updated_rows":    resp.UpdatedRows,
		"updated_columns": resp.UpdatedColumns,
		"updated_cells":   resp.UpdatedCells,
	})
}
