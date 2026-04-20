package google

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/supersuit-tech/permission-slip/connectors"
)

// createSpreadsheetAction implements connectors.Action for google.create_spreadsheet.
// It creates a new Google Sheets spreadsheet via the Sheets API
// POST /v4/spreadsheets.
type createSpreadsheetAction struct {
	conn *GoogleConnector
}

// createSpreadsheetParams is the user-facing parameter schema.
type createSpreadsheetParams struct {
	Title       string   `json:"title"`
	SheetTitles []string `json:"sheet_titles,omitempty"`
}

func (p *createSpreadsheetParams) validate() error {
	if p.Title == "" {
		return &connectors.ValidationError{Message: "missing required parameter: title"}
	}
	return nil
}

type sheetsSpreadsheetProperties struct {
	Title string `json:"title"`
}

type sheetsSheetProperties struct {
	Title string `json:"title"`
}

type sheetsSheetConfig struct {
	Properties sheetsSheetProperties `json:"properties"`
}

type sheetsCreateSpreadsheetRequest struct {
	Properties sheetsSpreadsheetProperties `json:"properties"`
	Sheets     []sheetsSheetConfig         `json:"sheets,omitempty"`
}

type sheetsCreateSpreadsheetResponse struct {
	SpreadsheetID  string `json:"spreadsheetId"`
	SpreadsheetURL string `json:"spreadsheetUrl"`
}

// Execute creates a new Google Sheets spreadsheet and returns its ID and URL.
func (a *createSpreadsheetAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params createSpreadsheetParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	body := sheetsCreateSpreadsheetRequest{
		Properties: sheetsSpreadsheetProperties{Title: params.Title},
	}
	for _, sheetTitle := range params.SheetTitles {
		body.Sheets = append(body.Sheets, sheetsSheetConfig{
			Properties: sheetsSheetProperties{Title: sheetTitle},
		})
	}

	var resp sheetsCreateSpreadsheetResponse
	createURL := a.conn.sheetsBaseURL + "/spreadsheets"
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodPost, createURL, body, &resp); err != nil {
		return nil, err
	}

	spreadsheetURL := resp.SpreadsheetURL
	if spreadsheetURL == "" {
		spreadsheetURL = "https://docs.google.com/spreadsheets/d/" + resp.SpreadsheetID + "/edit"
	}

	return connectors.JSONResult(map[string]string{
		"spreadsheet_id":  resp.SpreadsheetID,
		"spreadsheet_url": spreadsheetURL,
		"title":           params.Title,
	})
}
