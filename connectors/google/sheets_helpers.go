package google

import (
	"fmt"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

// validateRowLengths checks that all rows in a 2D values array have the same
// number of columns. Google Sheets API rejects ragged arrays, so catching this
// early provides a clearer error message.
func validateRowLengths(values [][]any) error {
	if len(values) == 0 {
		return nil
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
	return nil
}
