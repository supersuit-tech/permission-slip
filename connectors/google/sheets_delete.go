package google

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/supersuit-tech/permission-slip/connectors"
)

const (
	sheetsDimensionRows    = "ROWS"
	sheetsDimensionColumns = "COLUMNS"

	// maxSheetsDeletePreviewRows caps the number of rows fetched for the
	// approval preview so a large delete does not pull an entire sheet.
	maxSheetsDeletePreviewRows = 50
)

// sheetsDeleteRowsAction implements connectors.Action for google.sheets_delete_rows.
// It deletes rows or columns via spreadsheets.batchUpdate deleteDimension.
type sheetsDeleteRowsAction struct {
	conn *GoogleConnector
}

// sheetsDeleteRange is one half-open dimension span (start inclusive, end exclusive).
type sheetsDeleteRange struct {
	StartIndex int `json:"start_index"`
	EndIndex   int `json:"end_index"`
}

// sheetsDeleteRowsParams is the user-facing parameter schema.
//
// sheet_id is a pointer because 0 is a valid Google Sheets tab id (the first
// worksheet). start_index is a pointer for the same reason (row/column 0).
type sheetsDeleteRowsParams struct {
	SpreadsheetID string              `json:"spreadsheet_id"`
	SheetID       *int                `json:"sheet_id,omitempty"`
	SheetName     string              `json:"sheet_name"`
	StartIndex    *int                `json:"start_index,omitempty"`
	EndIndex      *int                `json:"end_index,omitempty"`
	Dimension     string              `json:"dimension"`
	Ranges        []sheetsDeleteRange `json:"ranges,omitempty"`
}

func (p *sheetsDeleteRowsParams) dimension() string {
	d := strings.ToUpper(strings.TrimSpace(p.Dimension))
	if d == "" {
		return sheetsDimensionRows
	}
	return d
}

func (p *sheetsDeleteRowsParams) validate() error {
	if p.SpreadsheetID == "" {
		return &connectors.ValidationError{Message: "missing required parameter: spreadsheet_id"}
	}
	if p.SheetID == nil && strings.TrimSpace(p.SheetName) == "" {
		return &connectors.ValidationError{Message: "missing required parameter: sheet_id or sheet_name"}
	}
	d := p.dimension()
	if d != sheetsDimensionRows && d != sheetsDimensionColumns {
		return &connectors.ValidationError{Message: "dimension must be ROWS or COLUMNS"}
	}
	if _, err := p.normalizedRanges(); err != nil {
		return err
	}
	return nil
}

func (p *sheetsDeleteRowsParams) normalizedRanges() ([]sheetsDeleteRange, error) {
	raw := make([]sheetsDeleteRange, 0, len(p.Ranges)+1)
	if p.StartIndex != nil || p.EndIndex != nil {
		if p.StartIndex == nil || p.EndIndex == nil {
			return nil, &connectors.ValidationError{Message: "start_index and end_index must both be provided"}
		}
		raw = append(raw, sheetsDeleteRange{StartIndex: *p.StartIndex, EndIndex: *p.EndIndex})
	}
	raw = append(raw, p.Ranges...)
	if len(raw) == 0 {
		return nil, &connectors.ValidationError{Message: "missing required parameter: start_index and end_index, or ranges"}
	}

	for i, r := range raw {
		if r.StartIndex < 0 {
			return nil, &connectors.ValidationError{
				Message: fmt.Sprintf("range %d: start_index must be >= 0", i),
			}
		}
		if r.EndIndex <= r.StartIndex {
			return nil, &connectors.ValidationError{
				Message: fmt.Sprintf("range %d: end_index must be greater than start_index", i),
			}
		}
	}

	merged := mergeDeleteRanges(raw)
	total := 0
	for _, r := range merged {
		total += r.EndIndex - r.StartIndex
	}
	if total > maxSheetsRows {
		return nil, &connectors.ValidationError{
			Message: fmt.Sprintf("cannot delete more than %d %s in a single request (got %d)",
				maxSheetsRows, strings.ToLower(p.dimension()), total),
		}
	}
	return merged, nil
}

// mergeDeleteRanges sorts ranges by start index and merges overlapping or
// adjacent spans so a single batchUpdate does not send conflicting deletes.
func mergeDeleteRanges(ranges []sheetsDeleteRange) []sheetsDeleteRange {
	if len(ranges) == 0 {
		return nil
	}
	sorted := append([]sheetsDeleteRange(nil), ranges...)
	sort.Slice(sorted, func(i, j int) bool {
		if sorted[i].StartIndex == sorted[j].StartIndex {
			return sorted[i].EndIndex < sorted[j].EndIndex
		}
		return sorted[i].StartIndex < sorted[j].StartIndex
	})
	out := []sheetsDeleteRange{sorted[0]}
	for _, r := range sorted[1:] {
		last := &out[len(out)-1]
		if r.StartIndex <= last.EndIndex {
			if r.EndIndex > last.EndIndex {
				last.EndIndex = r.EndIndex
			}
			continue
		}
		out = append(out, r)
	}
	return out
}

// deleteRangesBottomUp returns a copy of ranges sorted so higher indices are
// deleted first. Google applies batchUpdate requests in order; deleting from
// the bottom keeps earlier indices stable.
func deleteRangesBottomUp(ranges []sheetsDeleteRange) []sheetsDeleteRange {
	out := append([]sheetsDeleteRange(nil), ranges...)
	sort.Slice(out, func(i, j int) bool {
		return out[i].StartIndex > out[j].StartIndex
	})
	return out
}

type sheetsDeleteDimensionRange struct {
	SheetID    int    `json:"sheetId"`
	Dimension  string `json:"dimension"`
	StartIndex int    `json:"startIndex"`
	EndIndex   int    `json:"endIndex"`
}

type sheetsDeleteDimensionRequest struct {
	Range sheetsDeleteDimensionRange `json:"range"`
}

type sheetsBatchUpdateRequest struct {
	Requests []sheetsBatchUpdateRequestItem `json:"requests"`
}

type sheetsBatchUpdateRequestItem struct {
	DeleteDimension *sheetsDeleteDimensionRequest `json:"deleteDimension,omitempty"`
}

type sheetsBatchUpdateResponse struct {
	SpreadsheetID string `json:"spreadsheetId"`
}

type sheetIdentity struct {
	SheetID int
	Title   string
}

type spreadsheetSheetsLookup struct {
	Title  string
	Sheets []sheetIdentity
}

// Execute deletes the requested rows or columns from a Google Sheets tab.
func (a *sheetsDeleteRowsAction) Execute(ctx context.Context, req connectors.ActionRequest) (*connectors.ActionResult, error) {
	var params sheetsDeleteRowsParams
	if err := json.Unmarshal(req.Parameters, &params); err != nil {
		return nil, &connectors.ValidationError{Message: fmt.Sprintf("invalid parameters: %v", err)}
	}
	if err := params.validate(); err != nil {
		return nil, err
	}

	ranges, err := params.normalizedRanges()
	if err != nil {
		return nil, err
	}

	sheet, err := a.conn.resolveSheetIdentity(ctx, req.Credentials, params.SpreadsheetID, params.SheetID, params.SheetName)
	if err != nil {
		return nil, err
	}

	dimension := params.dimension()
	items := make([]sheetsBatchUpdateRequestItem, 0, len(ranges))
	for _, r := range deleteRangesBottomUp(ranges) {
		items = append(items, sheetsBatchUpdateRequestItem{
			DeleteDimension: &sheetsDeleteDimensionRequest{
				Range: sheetsDeleteDimensionRange{
					SheetID:    sheet.SheetID,
					Dimension:  dimension,
					StartIndex: r.StartIndex,
					EndIndex:   r.EndIndex,
				},
			},
		})
	}

	batchURL := a.conn.sheetsBaseURL + "/spreadsheets/" +
		url.PathEscape(params.SpreadsheetID) + ":batchUpdate"
	var resp sheetsBatchUpdateResponse
	if err := a.conn.doJSON(ctx, req.Credentials, http.MethodPost, batchURL, sheetsBatchUpdateRequest{Requests: items}, &resp); err != nil {
		return nil, err
	}

	deleted := 0
	for _, r := range ranges {
		deleted += r.EndIndex - r.StartIndex
	}
	spreadsheetID := resp.SpreadsheetID
	if spreadsheetID == "" {
		spreadsheetID = params.SpreadsheetID
	}
	return connectors.JSONResult(map[string]any{
		"spreadsheet_id": spreadsheetID,
		"sheet_id":       sheet.SheetID,
		"sheet_name":     sheet.Title,
		"dimension":      dimension,
		"deleted_count":  deleted,
		"ranges":         ranges,
	})
}

func (c *GoogleConnector) lookupSpreadsheetSheets(ctx context.Context, creds connectors.Credentials, spreadsheetID string) (spreadsheetSheetsLookup, error) {
	listURL := c.sheetsBaseURL + "/spreadsheets/" +
		url.PathEscape(spreadsheetID) +
		"?fields=" + url.QueryEscape("properties.title,sheets.properties(sheetId,title)")

	var resp struct {
		Properties struct {
			Title string `json:"title"`
		} `json:"properties"`
		Sheets []struct {
			Properties sheetsProperties `json:"properties"`
		} `json:"sheets"`
	}
	if err := c.doJSON(ctx, creds, http.MethodGet, listURL, nil, &resp); err != nil {
		return spreadsheetSheetsLookup{}, err
	}
	sheets := make([]sheetIdentity, 0, len(resp.Sheets))
	for _, s := range resp.Sheets {
		sheets = append(sheets, sheetIdentity{
			SheetID: s.Properties.SheetID,
			Title:   s.Properties.Title,
		})
	}
	return spreadsheetSheetsLookup{Title: resp.Properties.Title, Sheets: sheets}, nil
}

func (c *GoogleConnector) resolveSheetIdentity(ctx context.Context, creds connectors.Credentials, spreadsheetID string, sheetID *int, sheetName string) (sheetIdentity, error) {
	lookup, err := c.lookupSpreadsheetSheets(ctx, creds, spreadsheetID)
	if err != nil {
		return sheetIdentity{}, err
	}
	return matchSheetIdentity(lookup.Sheets, sheetID, sheetName)
}

func matchSheetIdentity(sheets []sheetIdentity, sheetID *int, sheetName string) (sheetIdentity, error) {
	sheetName = strings.TrimSpace(sheetName)
	if sheetID != nil {
		for _, s := range sheets {
			if s.SheetID == *sheetID {
				if sheetName != "" && s.Title != sheetName {
					return sheetIdentity{}, &connectors.ValidationError{
						Message: fmt.Sprintf("sheet_id %d is tab %q, not %q", *sheetID, s.Title, sheetName),
					}
				}
				return s, nil
			}
		}
		return sheetIdentity{}, unknownSheetError(sheets, fmt.Sprintf("sheet_id %d", *sheetID))
	}
	for _, s := range sheets {
		if s.Title == sheetName {
			return s, nil
		}
	}
	return sheetIdentity{}, unknownSheetError(sheets, fmt.Sprintf("sheet_name %q", sheetName))
}

func unknownSheetError(sheets []sheetIdentity, requested string) error {
	msg := fmt.Sprintf("%s does not match a tab in this spreadsheet. ", requested)
	if len(sheets) == 0 {
		msg += "The spreadsheet has no worksheets."
	} else {
		quoted := make([]string, len(sheets))
		for i, s := range sheets {
			quoted[i] = fmt.Sprintf("%q", s.Title)
		}
		msg += "Available tabs: " + strings.Join(quoted, ", ") + ". Use google.sheets_list_sheets to discover tab names."
	}
	return &connectors.ValidationError{Message: msg}
}

func quoteA1SheetName(name string) string {
	if name == "" {
		return name
	}
	needsQuote := false
	for _, r := range name {
		if (r >= 'A' && r <= 'Z') || (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '_' {
			continue
		}
		needsQuote = true
		break
	}
	if !needsQuote {
		return name
	}
	return "'" + strings.ReplaceAll(name, "'", "''") + "'"
}

func columnIndexToA1(index int) string {
	if index < 0 {
		return ""
	}
	// 0 -> A, 25 -> Z, 26 -> AA
	n := index + 1
	var b strings.Builder
	for n > 0 {
		n--
		b.WriteByte(byte('A' + (n % 26)))
		n /= 26
	}
	runes := []rune(b.String())
	for i, j := 0, len(runes)-1; i < j; i, j = i+1, j-1 {
		runes[i], runes[j] = runes[j], runes[i]
	}
	return string(runes)
}

func formatOneDeleteSpan(dimension string, start, end int) string {
	if dimension == sheetsDimensionColumns {
		a := columnIndexToA1(start)
		b := columnIndexToA1(end - 1)
		if a == b {
			return a
		}
		return a + "–" + b
	}
	start1 := start + 1
	end1 := end
	if start1 == end1 {
		return strconv.Itoa(start1)
	}
	return fmt.Sprintf("%d–%d", start1, end1)
}

func formatDimensionLabel(dimension string, ranges []sheetsDeleteRange) string {
	noun := "rows"
	if dimension == sheetsDimensionColumns {
		noun = "columns"
	}
	if len(ranges) == 0 {
		return noun
	}
	parts := make([]string, 0, len(ranges))
	for _, r := range ranges {
		parts = append(parts, formatOneDeleteSpan(dimension, r.StartIndex, r.EndIndex))
	}
	if len(parts) == 1 {
		return noun + " " + parts[0]
	}
	if len(parts) == 2 {
		return noun + " " + parts[0] + " and " + parts[1]
	}
	return noun + " " + strings.Join(parts[:len(parts)-1], ", ") + ", and " + parts[len(parts)-1]
}

func a1RangeForDeletePreview(sheetName, dimension string, r sheetsDeleteRange, previewCap int) string {
	quoted := quoteA1SheetName(sheetName)
	if dimension == sheetsDimensionColumns {
		startCol := columnIndexToA1(r.StartIndex)
		endCol := columnIndexToA1(r.EndIndex - 1)
		endRow := previewCap
		if endRow < 1 {
			endRow = maxSheetsDeletePreviewRows
		}
		return fmt.Sprintf("%s!%s1:%s%d", quoted, startCol, endCol, endRow)
	}
	start1 := r.StartIndex + 1
	end1 := r.EndIndex
	if previewCap > 0 && end1-r.StartIndex > previewCap {
		end1 = r.StartIndex + previewCap
	}
	return fmt.Sprintf("%s!%d:%d", quoted, start1, end1)
}

func previewRowNumbers(dimension string, ranges []sheetsDeleteRange, previewCap int) []int {
	if dimension == sheetsDimensionColumns {
		return nil
	}
	out := make([]int, 0, previewCap)
	for _, r := range ranges {
		for i := r.StartIndex; i < r.EndIndex && len(out) < previewCap; i++ {
			out = append(out, i+1)
		}
		if len(out) >= previewCap {
			break
		}
	}
	return out
}

func (c *GoogleConnector) fetchDeletePreviewValues(ctx context.Context, creds connectors.Credentials, spreadsheetID, sheetName, dimension string, ranges []sheetsDeleteRange) ([][]any, bool, error) {
	total := 0
	for _, r := range ranges {
		total += r.EndIndex - r.StartIndex
	}
	truncated := dimension == sheetsDimensionColumns || total > maxSheetsDeletePreviewRows

	a1Ranges := make([]string, 0, len(ranges))
	remaining := maxSheetsDeletePreviewRows
	for _, r := range ranges {
		if remaining <= 0 {
			truncated = true
			break
		}
		span := r.EndIndex - r.StartIndex
		capForRange := remaining
		if dimension == sheetsDimensionColumns {
			capForRange = maxSheetsDeletePreviewRows
		}
		a1Ranges = append(a1Ranges, a1RangeForDeletePreview(sheetName, dimension, r, capForRange))
		if dimension != sheetsDimensionColumns {
			if span > remaining {
				truncated = true
				remaining = 0
			} else {
				remaining -= span
			}
		}
	}

	q := url.Values{}
	for _, r := range a1Ranges {
		q.Add("ranges", r)
	}
	readURL := c.sheetsBaseURL + "/spreadsheets/" +
		url.PathEscape(spreadsheetID) + "/values:batchGet?" + q.Encode()

	var batchResp sheetsBatchGetValuesResponse
	if err := c.doJSON(ctx, creds, http.MethodGet, readURL, nil, &batchResp); err != nil {
		return nil, truncated, err
	}
	values := make([][]any, 0)
	for _, vr := range batchResp.ValueRanges {
		values = append(values, vr.Values...)
		if len(values) > maxSheetsDeletePreviewRows {
			values = values[:maxSheetsDeletePreviewRows]
			truncated = true
			break
		}
	}
	return values, truncated, nil
}

func (c *GoogleConnector) resolveSpreadsheetDelete(ctx context.Context, creds connectors.Credentials, params json.RawMessage) (map[string]any, error) {
	var p sheetsDeleteRowsParams
	if err := json.Unmarshal(params, &p); err != nil || p.SpreadsheetID == "" {
		return nil, fmt.Errorf("missing spreadsheet_id")
	}

	lookup, err := c.lookupSpreadsheetSheets(ctx, creds, p.SpreadsheetID)
	if err != nil {
		return nil, err
	}

	details := map[string]any{"title": lookup.Title}
	if file, fileErr := c.lookupDriveFile(ctx, creds, p.SpreadsheetID); fileErr == nil {
		c.attachSharedDriveDetails(ctx, creds, details, file.DriveID)
	}
	connectors.AttachResources(details, connectors.ResourceRef{
		Param: "spreadsheet_id",
		ID:    p.SpreadsheetID,
		Name:  lookup.Title,
		URL:   spreadsheetURL(p.SpreadsheetID),
	})

	sheet, matchErr := matchSheetIdentity(lookup.Sheets, p.SheetID, p.SheetName)
	if matchErr != nil {
		if p.SheetName != "" {
			details["sheet_name"] = p.SheetName
		}
		return details, nil
	}
	details["sheet_name"] = sheet.Title
	details["sheet_id"] = sheet.SheetID

	ranges, rangeErr := p.normalizedRanges()
	if rangeErr != nil {
		return details, nil
	}
	dimension := p.dimension()
	details["dimension"] = dimension
	details["row_label"] = formatDimensionLabel(dimension, ranges)
	if nums := previewRowNumbers(dimension, ranges, maxSheetsDeletePreviewRows); len(nums) > 0 {
		details["row_numbers"] = nums
	}

	values, truncated, valuesErr := c.fetchDeletePreviewValues(ctx, creds, p.SpreadsheetID, sheet.Title, dimension, ranges)
	if valuesErr == nil {
		details["values"] = values
		if truncated {
			details["values_truncated"] = true
		}
	}
	return details, nil
}
