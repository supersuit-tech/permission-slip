package google

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// sheetsListSheetsAction implements connectors.Action for google.sheets_list_sheets.
// It lists all worksheets in a spreadsheet via the Google Sheets API
// GET /v4/spreadsheets/{id}?fields=sheets.properties.
type sheetsListSheetsAction struct {
	conn *GoogleConnector
}

// sheetsListSheetsParams is the user-facing parameter schema.
type sheetsListSheetsParams struct {
	SpreadsheetID string `json:"spreadsheet_id"`
}

func (p *sheetsListSheetsParams) validate() error {
	if p.SpreadsheetID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: spreadsheet_id"}
	}
	return nil
}

// sheetsSpreadsheetResponse is the Google Sheets API response for spreadsheets.get.
type sheetsSpreadsheetResponse struct {
	Sheets []struct {
		Properties sheetsProperties `json:"properties"`
	} `json:"sheets"`
}

type sheetsProperties struct {
	SheetID    int    `json:"sheetId"`
	Title      string `json:"title"`
	Index      int    `json:"index"`
	SheetType  string `json:"sheetType"`
	RowCount   int    `json:"rowCount,omitempty"`
	ColumnCount int   `json:"columnCount,omitempty"`
}

// sheetSummary is the shape returned to the agent.
type sheetSummary struct {
	SheetID   int    `json:"sheet_id"`
	Title     string `json:"title"`
	Index     int    `json:"index"`
	SheetType string `json:"sheet_type"`
}

// Execute lists all worksheets in a Google Sheets spreadsheet.
func (a *sheetsListSheetsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params sheetsListSheetsParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	listURL := a.conn.sheetsBaseURL + "/spreadsheets/" +
		url.PathEscape(params.SpreadsheetID) + "?fields=sheets.properties"

	var resp sheetsSpreadsheetResponse
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodGet, listURL, nil, &resp); err != nil {
		return nil, err
	}

	sheets := make([]sheetSummary, 0, len(resp.Sheets))
	for _, s := range resp.Sheets {
		sheets = append(sheets, sheetSummary{
			SheetID:   s.Properties.SheetID,
			Title:     s.Properties.Title,
			Index:     s.Properties.Index,
			SheetType: s.Properties.SheetType,
		})
	}

	return connectors.JSONResult(map[string]any{
		"sheets": sheets,
	})
}
