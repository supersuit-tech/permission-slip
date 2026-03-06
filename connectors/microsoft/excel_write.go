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

// excelWriteRangeParams holds the validated parameters for writing to a cell range.
type excelWriteRangeParams struct {
	ItemID    string  `json:"item_id"`
	SheetName string  `json:"sheet_name"`
	Range     string  `json:"range"`
	Values    [][]any `json:"values"`
}

func (p *excelWriteRangeParams) validate() error {
	if err := validateItemID(p.ItemID); err != nil {
		return err
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
	return validateValuesGrid(p.Values)
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

	path := fmt.Sprintf("%s/worksheets/%s/range(address='%s')",
		excelWorkbookPath(params.ItemID),
		url.PathEscape(params.SheetName),
		url.QueryEscape(params.Range),
	)

	body := graphWriteRangeRequest{Values: params.Values}

	var resp graphRangeResponse
	if err := a.conn.doRequest(ctx, http.MethodPatch, path, req.Credentials, body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(newRangeResult(resp))
}
