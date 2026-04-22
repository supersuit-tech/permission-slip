package google

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// sheetsReadRangeAction implements connectors.Action for google.sheets_read_range.
// It reads cell values from a specified range via the Google Sheets API
// GET /v4/spreadsheets/{id}/values:batchGet?ranges=... .
//
// We use values:batchGet with the range in the query string instead of
// values/{range} in the path. Path-segment encoding turns "!" in A1 ranges into
// "%21", which breaks some proxies and the read path in production while write
// (which uses the same encoded segment) can still succeed depending on routing.
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

// sheetsValueRange is the Google Sheets API ValueRange resource (values.get and batchGet).
type sheetsValueRange struct {
	Range          string  `json:"range"`
	MajorDimension string  `json:"majorDimension"`
	Values         [][]any `json:"values"`
}

// sheetsBatchGetValuesResponse is the Google Sheets API response for values.batchGet.
type sheetsBatchGetValuesResponse struct {
	SpreadsheetID string             `json:"spreadsheetId,omitempty"`
	ValueRanges   []sheetsValueRange `json:"valueRanges"`
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

	q := url.Values{}
	q.Set("ranges", params.Range)
	readURL := a.conn.sheetsBaseURL + "/spreadsheets/" +
		url.PathEscape(params.SpreadsheetID) + "/values:batchGet?" + q.Encode()

	var batchResp sheetsBatchGetValuesResponse
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodGet, readURL, nil, &batchResp); err != nil {
		return nil, remapInvalidRangeError(ctx, a.conn, req.Credentials, params.SpreadsheetID, params.Range, err)
	}
	if len(batchResp.ValueRanges) == 0 {
		return nil, &connectors.ExternalError{Message: "Google Sheets API returned no value ranges for the requested range"}
	}
	resp := batchResp.ValueRanges[0]

	return connectors.JSONResult(map[string]any{
		"range":  resp.Range,
		"values": resp.Values,
	})
}
