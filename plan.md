# Plan: Add Google Sheets Actions & Microsoft Excel Actions

## Overview

Add spreadsheet actions to both connectors:
- **Google**: 4 new Sheets actions using the Google Sheets API v4
- **Microsoft**: 4 new Excel actions using the Microsoft Graph API

No migrations, schema changes, or frontend changes needed — the manifest-driven auto-seeding and dynamic action UI handle everything automatically.

---

## Part 1: Google Sheets Actions

### New Files

All under `connectors/google/`:

#### 1. `sheets_read.go` — `google.sheets_read_range`
- **Risk:** low
- **API:** `GET https://sheets.googleapis.com/v4/spreadsheets/{spreadsheetId}/values/{range}`
- **Parameters:**
  - `spreadsheet_id` (required, string) — The spreadsheet ID from the URL
  - `range` (required, string) — A1 notation range (e.g. `"Sheet1!A1:D10"`)
- **Returns:** 2D array of cell values

#### 2. `sheets_write.go` — `google.sheets_write_range`
- **Risk:** medium
- **API:** `PUT https://sheets.googleapis.com/v4/spreadsheets/{spreadsheetId}/values/{range}?valueInputOption=USER_ENTERED`
- **Parameters:**
  - `spreadsheet_id` (required, string)
  - `range` (required, string) — A1 notation range
  - `values` (required, array of arrays) — Row data to write
- **Returns:** Updated range, updated rows/columns/cells count

#### 3. `sheets_append.go` — `google.sheets_append_rows`
- **Risk:** medium
- **API:** `POST https://sheets.googleapis.com/v4/spreadsheets/{spreadsheetId}/values/{range}:append?valueInputOption=USER_ENTERED`
- **Parameters:**
  - `spreadsheet_id` (required, string)
  - `range` (required, string) — Target range/table (e.g. `"Sheet1!A:E"`)
  - `values` (required, array of arrays) — Rows to append
- **Returns:** Updated range, updated rows count

#### 4. `sheets_list.go` — `google.sheets_list_sheets`
- **Risk:** low
- **API:** `GET https://sheets.googleapis.com/v4/spreadsheets/{spreadsheetId}?fields=sheets.properties`
- **Parameters:**
  - `spreadsheet_id` (required, string)
- **Returns:** Array of sheet names and IDs within the spreadsheet

### Changes to `google.go`

1. Add `defaultSheetsBaseURL = "https://sheets.googleapis.com/v4"` constant
2. Add `sheetsBaseURL string` field to `GoogleConnector` struct
3. Initialize in `New()` and accept in `newForTest()`
4. Register 4 new actions in `Actions()` map
5. Add 4 new `ManifestAction` entries in `Manifest()` with JSON schemas
6. Add OAuth scope: `https://www.googleapis.com/auth/spreadsheets`
7. Add templates:
   - `tpl_google_sheets_read_range` — Read any range from a specific spreadsheet
   - `tpl_google_sheets_write_range` — Write to any range in a specific spreadsheet
   - `tpl_google_sheets_append_rows` — Append rows to a specific spreadsheet
   - `tpl_google_sheets_read_any` — Read from any spreadsheet (all params wildcard)

### Test Files

- `sheets_read_test.go` — Mock Sheets API, test happy path + error cases
- `sheets_write_test.go` — Same pattern
- `sheets_append_test.go` — Same pattern
- `sheets_list_test.go` — Same pattern

---

## Part 2: Microsoft Excel Actions

### New Files

All under `connectors/microsoft/`:

#### 1. `excel_read.go` — `microsoft.excel_read_range`
- **Risk:** low
- **API:** `GET /me/drive/items/{itemId}/workbook/worksheets/{sheetName}/range(address='{range}')`
- **Parameters:**
  - `item_id` (required, string) — OneDrive file ID for the workbook
  - `sheet_name` (required, string) — Worksheet name (e.g. `"Sheet1"`)
  - `range` (required, string) — A1 range (e.g. `"A1:D10"`)
- **Returns:** 2D array of cell values

#### 2. `excel_write.go` — `microsoft.excel_write_range`
- **Risk:** medium
- **API:** `PATCH /me/drive/items/{itemId}/workbook/worksheets/{sheetName}/range(address='{range}')`
- **Parameters:**
  - `item_id` (required, string)
  - `sheet_name` (required, string)
  - `range` (required, string)
  - `values` (required, array of arrays) — Cell values to write
- **Returns:** Updated range address

#### 3. `excel_append.go` — `microsoft.excel_append_rows`
- **Risk:** medium
- **API:** `POST /me/drive/items/{itemId}/workbook/tables/{tableName}/rows`
- **Parameters:**
  - `item_id` (required, string)
  - `table_name` (required, string) — Named table in the workbook
  - `values` (required, array of arrays) — Rows to append
- **Returns:** Row index and values

#### 4. `excel_list_worksheets.go` — `microsoft.excel_list_worksheets`
- **Risk:** low
- **API:** `GET /me/drive/items/{itemId}/workbook/worksheets`
- **Parameters:**
  - `item_id` (required, string) — OneDrive file ID
- **Returns:** Array of worksheet names and IDs

### Changes to `microsoft.go`

1. Register 4 new actions in `Actions()` map
2. Add 4 new `ManifestAction` entries in `Manifest()` with JSON schemas
3. Add OAuth scope: `Files.ReadWrite` (needed for OneDrive/Excel access)
4. Update connector description: `"Microsoft 365 integration for email, calendar, and Excel via Microsoft Graph API"`
5. Add templates:
   - `tpl_microsoft_excel_read_range` — Read any range from a specific workbook
   - `tpl_microsoft_excel_write_range` — Write to any range in a specific workbook
   - `tpl_microsoft_excel_append_rows` — Append rows to a table in a specific workbook
   - `tpl_microsoft_excel_read_any` — Read from any workbook (all params wildcard)

### Test Files

- `excel_read_test.go`
- `excel_write_test.go`
- `excel_append_test.go`
- `excel_list_worksheets_test.go`

---

## Part 3: Update Documentation

- Update `connectors/google/README.md` with Sheets actions, parameters, and examples
- Update `connectors/microsoft/README.md` with Excel actions, parameters, and examples

---

## Implementation Order

1. Google Sheets actions (sheets_list → sheets_read → sheets_write → sheets_append) + tests
2. Update `google.go` (struct, manifest, actions map, scopes)
3. Microsoft Excel actions (excel_list → excel_read → excel_write → excel_append) + tests
4. Update `microsoft.go` (manifest, actions map, scopes)
5. Update both README files
6. Run `make test-backend` + `make build` to verify
7. Commit and push

## What Does NOT Need to Change

- **No database migrations** — connector_actions are auto-seeded from manifests
- **No seed file updates** — manifest-driven seeding replaces manual seeds
- **No frontend changes** — the UI dynamically renders actions/parameters from the API
- **No API spec changes** — connector endpoints already return dynamic action lists
- **No new dependencies** — both APIs use plain net/http like existing actions
