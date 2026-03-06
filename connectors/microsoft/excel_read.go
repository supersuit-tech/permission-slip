package microsoft

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// excelReadRangeAction implements connectors.Action for microsoft.excel_read_range.
// It reads cell values from a worksheet range via
// GET /me/drive/items/{itemId}/workbook/worksheets/{sheetName}/range(address='{range}').
type excelReadRangeAction struct {
	conn *MicrosoftConnector
}

// excelReadRangeParams holds the validated parameters for reading a cell range.
type excelReadRangeParams struct {
	ItemID    string `json:"item_id"`
	SheetName string `json:"sheet_name"`
	Range     string `json:"range"`
}

func (p *excelReadRangeParams) validate() error {
	if err := validateItemID(p.ItemID); err != nil {
		return err
	}
	if p.SheetName == "" {
		return &connectors.ValidationError{Message: "sheet_name is required"}
	}
	if err := validatePathSegment("sheet_name", p.SheetName); err != nil {
		return err
	}
	if p.Range == "" {
		return &connectors.ValidationError{Message: "range is required"}
	}
	return nil
}

// graphRangeResponse is the Microsoft Graph API response for reading a range.
type graphRangeResponse struct {
	Address string `json:"address"`
	Values  [][]any `json:"values"`
}

// rangeResult is the simplified response returned to the caller.
// RowCount and ColumnCount are computed from the values grid so callers
// can work with data dimensions without counting manually.
type rangeResult struct {
	Address     string  `json:"address"`
	Values      [][]any `json:"values"`
	RowCount    int     `json:"row_count"`
	ColumnCount int     `json:"column_count"`
}

// Execute reads cell values from the specified worksheet range.
func (a *excelReadRangeAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params excelReadRangeParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	path := fmt.Sprintf("%s/worksheets/%s/range(address='%s')",
		excelWorkbookPath(params.ItemID),
		url.PathEscape(params.SheetName),
		url.QueryEscape(params.Range),
	)

	var resp graphRangeResponse
	if err := a.conn.doRequest(ctx, http.MethodGet, path, req.Credentials, nil, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(newRangeResult(resp))
}
