package google

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"

	"github.com/supersuit-tech/permission-slip/connectors"
)

const (
	// maxSheetsRows is the maximum number of rows allowed in a single write or
	// append request. This prevents excessive memory usage during JSON marshaling
	// before the request is sent. The Google Sheets API has its own limits, but
	// this catches obviously oversized payloads early.
	maxSheetsRows = 10_000

	// maxSheetsCells is the maximum total cell count (rows × columns) allowed in
	// a single request. This provides a tighter bound than maxSheetsRows alone,
	// catching cases like 1,000 rows × 1,000 columns.
	maxSheetsCells = 500_000
)

// sheetsUpdateValuesRequest is the request body sent to both values.update
// (write) and values.append. It lives here because both actions share it.
type sheetsUpdateValuesRequest struct {
	Range          string  `json:"range"`
	MajorDimension string  `json:"majorDimension"`
	Values         [][]any `json:"values"`
}

// validateValues checks that the values array is within safe limits and that
// all rows have a consistent column count.
func validateValues(values [][]any) error {
	if len(values) == 0 {
		return nil
	}
	if len(values) > maxSheetsRows {
		return &connectors.ValidationError{
			Message: fmt.Sprintf("values exceeds maximum of %d rows (got %d)", maxSheetsRows, len(values)),
		}
	}
	firstLen := len(values[0])
	for i, row := range values[1:] {
		if len(row) != firstLen {
			return &connectors.ValidationError{
				Message: fmt.Sprintf(
					"all rows must have the same number of columns: row 0 has %d but row %d has %d",
					firstLen, i+1, len(row)),
			}
		}
	}
	totalCells := len(values) * firstLen
	if totalCells > maxSheetsCells {
		return &connectors.ValidationError{
			Message: fmt.Sprintf("values exceeds maximum of %d cells (got %d rows × %d columns = %d cells)",
				maxSheetsCells, len(values), firstLen, totalCells),
		}
	}
	return nil
}

// remapInvalidRangeError converts Google's "Unable to parse range" HTTP 400
// into a ValidationError that lists the spreadsheet's actual tab titles, so
// the agent can self-correct on the next call instead of this surfacing as
// an opaque ExternalError (which gets captured in Sentry). For any other
// error, or if listing the spreadsheet's tabs fails, the original error is
// returned unchanged so we never mask a genuine external failure.
func remapInvalidRangeError(ctx context.Context, conn *GoogleConnector, creds connectors.Credentials, spreadsheetID, rangeStr string, err error) error {
	if err == nil {
		return nil
	}
	var extErr *connectors.ExternalError
	if !errors.As(err, &extErr) {
		return err
	}
	if !strings.Contains(extErr.Message, "Unable to parse range") {
		return err
	}

	titles, listErr := fetchSheetTitles(ctx, conn, creds, spreadsheetID)
	if listErr != nil {
		return err
	}

	msg := fmt.Sprintf("range %q refers to a tab that does not exist in this spreadsheet. ", rangeStr)
	if len(titles) == 0 {
		msg += "The spreadsheet has no worksheets."
	} else {
		quoted := make([]string, len(titles))
		for i, t := range titles {
			quoted[i] = fmt.Sprintf("%q", t)
		}
		msg += "Available tabs: " + strings.Join(quoted, ", ") + ". Use google.sheets_list_sheets to discover tab names."
	}
	return &connectors.ValidationError{Message: msg}
}

// fetchSheetTitles calls the Google Sheets API to list the tab titles in a
// spreadsheet. It uses the same endpoint as sheetsListSheetsAction.Execute.
func fetchSheetTitles(ctx context.Context, conn *GoogleConnector, creds connectors.Credentials, spreadsheetID string) ([]string, error) {
	listURL := conn.sheetsBaseURL + "/spreadsheets/" +
		url.PathEscape(spreadsheetID) +
		"?fields=sheets.properties(title)"

	var resp sheetsSpreadsheetResponse
	if err := conn.doJSON(ctx, creds, http.MethodGet, listURL, nil, &resp); err != nil {
		return nil, err
	}
	titles := make([]string, 0, len(resp.Sheets))
	for _, s := range resp.Sheets {
		titles = append(titles, s.Properties.Title)
	}
	return titles, nil
}
