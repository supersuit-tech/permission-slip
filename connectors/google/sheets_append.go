package google

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// columnOnlyRange matches ranges like "Sheet1!A:C" or "Sheet1!A:A" where the
// column range has no row anchors. The Sheets API values.append endpoint does
// not accept these; callers should pass just the sheet name instead.
var columnOnlyRange = regexp.MustCompile(`^([^!]+)![A-Za-z]+:[A-Za-z]+$`)

// sheetsAppendRowsAction implements connectors.Action for google.sheets_append_rows.
// It appends rows to a sheet via the Google Sheets API
// POST /v4/spreadsheets/{id}/values/{range}:append?valueInputOption=USER_ENTERED.
type sheetsAppendRowsAction struct {
	conn *GoogleConnector
}

// sheetsAppendRowsParams is the user-facing parameter schema.
type sheetsAppendRowsParams struct {
	SpreadsheetID string  `json:"spreadsheet_id"`
	Range         string  `json:"range"`
	Values        [][]any `json:"values"`
}

// validate checks that required fields are present and values are well-formed.
// normalizeRange converts a column-only range like "Sheet1!A:C" to just the
// sheet name "Sheet1". values.append requires a cell anchor, not a bare column
// span, so callers who pass "Sheet1!A:C" get the same result as "Sheet1".
func normalizeAppendRange(r string) string {
	if m := columnOnlyRange.FindStringSubmatch(r); m != nil {
		return strings.TrimSpace(m[1])
	}
	return r
}

func (p *sheetsAppendRowsParams) validate() error {
	if p.SpreadsheetID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: spreadsheet_id"}
	}
	if p.Range == "" {
		return &connectors.ValidationError{Message: "missing required parameter: range (e.g. 'Sheet1' or 'Sheet1!A1')"}
	}
	if len(p.Values) == 0 {
		return &connectors.ValidationError{Message: "values must contain at least one row of data"}
	}
	if err := validateValues(p.Values); err != nil {
		return err
	}
	return nil
}

// sheetsAppendValuesResponse is the Google Sheets API response for values.append.
type sheetsAppendValuesResponse struct {
	SpreadsheetID string `json:"spreadsheetId"`
	Updates       struct {
		UpdatedRange   string `json:"updatedRange"`
		UpdatedRows    int    `json:"updatedRows"`
		UpdatedColumns int    `json:"updatedColumns"`
		UpdatedCells   int    `json:"updatedCells"`
	} `json:"updates"`
}

// Execute appends rows to the specified range in a Google Sheets spreadsheet.
func (a *sheetsAppendRowsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params sheetsAppendRowsParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	appendRange := normalizeAppendRange(params.Range)
	appendURL := a.conn.sheetsBaseURL + "/spreadsheets/" +
		url.PathEscape(params.SpreadsheetID) + "/values/" +
		url.PathEscape(appendRange) + ":append?valueInputOption=USER_ENTERED"

	body := sheetsUpdateValuesRequest{
		Range:          appendRange,
		MajorDimension: "ROWS",
		Values:         params.Values,
	}

	var resp sheetsAppendValuesResponse
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodPost, appendURL, body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(map[string]any{
		"updated_range":   resp.Updates.UpdatedRange,
		"updated_rows":    resp.Updates.UpdatedRows,
		"updated_columns": resp.Updates.UpdatedColumns,
		"updated_cells":   resp.Updates.UpdatedCells,
	})
}
