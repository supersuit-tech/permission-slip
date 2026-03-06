package microsoft

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// excelWriteRangeAction implements connectors.Action for microsoft.excel_write_range.
// It writes cell values to a worksheet range via
// PATCH /me/drive/items/{itemId}/workbook/worksheets/{sheetName}/range(address='{range}').
type excelWriteRangeAction struct {
	conn *MicrosoftConnector
}

type excelWriteRangeParams struct {
	ItemID    string  `json:"item_id"`
	SheetName string  `json:"sheet_name"`
	Range     string  `json:"range"`
	Values    [][]any `json:"values"`
}

func (p *excelWriteRangeParams) validate() error {
	if p.ItemID == "" {
		return &connectors.ValidationError{Message: "item_id is required"}
	}
	if p.SheetName == "" {
		return &connectors.ValidationError{Message: "sheet_name is required"}
	}
	if p.Range == "" {
		return &connectors.ValidationError{Message: "range is required"}
	}
	if len(p.Values) == 0 {
		return &connectors.ValidationError{Message: "values is required and must not be empty"}
	}
	if err := validatePathSegment("item_id", p.ItemID); err != nil {
		return err
	}
	if err := validateValuesGrid(p.Values); err != nil {
		return err
	}
	return nil
}

// graphWriteRangeRequest is the request body for the Graph API range update.
type graphWriteRangeRequest struct {
	Values [][]any `json:"values"`
}

// Execute writes cell values to the specified worksheet range.
func (a *excelWriteRangeAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params excelWriteRangeParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/me/drive/items/%s/workbook/worksheets/%s/range(address='%s')",
		params.ItemID,
		url.PathEscape(params.SheetName),
		url.QueryEscape(params.Range),
	)

	body := graphWriteRangeRequest{Values: params.Values}

	var resp graphRangeResponse
	if err := a.conn.doRequest(ctx, http.MethodPatch, path, req.Credentials, body, &resp); err != nil {
		return nil, err
	}

	colCount := 0
	if len(resp.Values) > 0 {
		colCount = len(resp.Values[0])
	}

	return connectors.JSONResult(rangeResult{
		Address:     resp.Address,
		Values:      resp.Values,
		RowCount:    len(resp.Values),
		ColumnCount: colCount,
	})
}
