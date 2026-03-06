package microsoft

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// excelAppendRowsAction implements connectors.Action for microsoft.excel_append_rows.
// It appends rows to a named table via
// POST /me/drive/items/{itemId}/workbook/tables/{tableName}/rows.
type excelAppendRowsAction struct {
	conn *MicrosoftConnector
}

type excelAppendRowsParams struct {
	ItemID    string  `json:"item_id"`
	TableName string  `json:"table_name"`
	Values    [][]any `json:"values"`
}

func (p *excelAppendRowsParams) validate() error {
	if err := validateItemID(p.ItemID); err != nil {
		return err
	}
	if p.TableName == "" {
		return &connectors.ValidationError{Message: "table_name is required"}
	}
	if err := validatePathSegment("table_name", p.TableName); err != nil {
		return err
	}
	if len(p.Values) == 0 {
		return &connectors.ValidationError{Message: "values is required and must not be empty"}
	}
	return validateValuesGrid(p.Values)
}

// graphAddRowsRequest is the request body for the Graph API table row add.
type graphAddRowsRequest struct {
	Values [][]any `json:"values"`
}

// graphAddRowsResponse is the Microsoft Graph API response for adding rows.
type graphAddRowsResponse struct {
	Index  int     `json:"index"`
	Values [][]any `json:"values"`
}

// appendRowsResult is the simplified response returned to the caller.
type appendRowsResult struct {
	Index     int     `json:"index"`
	Values    [][]any `json:"values"`
	RowsAdded int     `json:"rows_added"`
}

// Execute appends rows to the specified table in the workbook.
func (a *excelAppendRowsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params excelAppendRowsParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	path := fmt.Sprintf("%s/tables/%s/rows", excelWorkbookPath(params.ItemID), url.PathEscape(params.TableName))

	body := graphAddRowsRequest{Values: params.Values}

	var resp graphAddRowsResponse
	if err := a.conn.doRequest(ctx, http.MethodPost, path, req.Credentials, body, &resp); err != nil {
		return nil, err
	}

	return connectors.JSONResult(appendRowsResult{
		Index:     resp.Index,
		Values:    resp.Values,
		RowsAdded: len(resp.Values),
	})
}
