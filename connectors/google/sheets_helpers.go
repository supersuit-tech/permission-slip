package google

import (
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
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
