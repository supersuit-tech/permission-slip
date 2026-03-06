package microsoft

import (
	"fmt"
	"net/url"
)

// excelWorkbookPath returns the Graph API path prefix for a workbook,
// with the item ID properly URL-encoded. All Excel actions share this
// path prefix, so centralizing it prevents encoding inconsistencies.
func excelWorkbookPath(itemID string) string {
	return fmt.Sprintf("/me/drive/items/%s/workbook", url.PathEscape(itemID))
}

// newRangeResult constructs a rangeResult from a Graph API range response,
// computing row and column counts from the values grid. Used by both
// excel_read and excel_write to avoid duplicating this logic.
func newRangeResult(resp graphRangeResponse) rangeResult {
	colCount := 0
	if len(resp.Values) > 0 {
		colCount = len(resp.Values[0])
	}
	return rangeResult{
		Address:     resp.Address,
		Values:      resp.Values,
		RowCount:    len(resp.Values),
		ColumnCount: colCount,
	}
}

