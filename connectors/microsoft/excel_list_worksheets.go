package microsoft

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// excelListWorksheetsAction implements connectors.Action for microsoft.excel_list_worksheets.
// It lists all worksheets in a workbook via GET /me/drive/items/{itemId}/workbook/worksheets.
type excelListWorksheetsAction struct {
	conn *MicrosoftConnector
}

type excelListWorksheetsParams struct {
	ItemID string `json:"item_id"`
}

func (p *excelListWorksheetsParams) validate() error {
	if p.ItemID == "" {
		return &connectors.ValidationError{Message: "item_id is required"}
	}
	if err := validatePathSegment("item_id", p.ItemID); err != nil {
		return err
	}
	return nil
}

// graphWorksheetsResponse is the Microsoft Graph API response for listing worksheets.
type graphWorksheetsResponse struct {
	Value []graphWorksheet `json:"value"`
}

type graphWorksheet struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Position   int    `json:"position"`
	Visibility string `json:"visibility"`
}

// worksheetSummary is the simplified response returned to the caller.
type worksheetSummary struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Position   int    `json:"position"`
	Visibility string `json:"visibility"`
}

// Execute lists all worksheets in the specified workbook.
func (a *excelListWorksheetsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params excelListWorksheetsParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	path := fmt.Sprintf("/me/drive/items/%s/workbook/worksheets", url.PathEscape(params.ItemID))

	var resp graphWorksheetsResponse
	if err := a.conn.doRequest(ctx, http.MethodGet, path, req.Credentials, nil, &resp); err != nil {
		return nil, err
	}

	summaries := make([]worksheetSummary, len(resp.Value))
	for i, ws := range resp.Value {
		summaries[i] = worksheetSummary{
			ID:         ws.ID,
			Name:       ws.Name,
			Position:   ws.Position,
			Visibility: ws.Visibility,
		}
	}

	return connectors.JSONResult(summaries)
}
