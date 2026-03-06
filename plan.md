# Plan: Add Google Docs & Microsoft Word Connector Actions

## Overview

Add document management actions to the existing **Google** connector (Google Docs via Google Docs API) and **Microsoft** connector (Word via Microsoft Graph API). These follow the exact same patterns as the existing Gmail/Calendar actions.

## Scope of Actions

### Google Docs Actions (4 actions)

| Action Type | Name | Risk | Description |
|---|---|---|---|
| `google.create_document` | Create Document | medium | Create a new Google Doc with a title and optional initial body content |
| `google.get_document` | Get Document | low | Retrieve the content/metadata of an existing Google Doc by document ID |
| `google.update_document` | Update Document | medium | Append or insert text into an existing Google Doc using batchUpdate |
| `google.list_documents` | List Documents | low | Search/list Google Docs from Google Drive (uses Drive API files.list with mimeType filter) |

### Microsoft Word Actions (4 actions)

| Action Type | Name | Risk | Description |
|---|---|---|---|
| `microsoft.create_document` | Create Document | medium | Create a new Word document in OneDrive |
| `microsoft.get_document` | Get Document | low | Get the content/metadata of a Word document from OneDrive by item ID or path |
| `microsoft.update_document` | Update Document | medium | Update the content of a Word document in OneDrive |
| `microsoft.list_documents` | List Documents | low | List Word documents from OneDrive (filter by .docx file extension) |

## API Details

### Google Docs API Endpoints
- **Create**: `POST https://docs.googleapis.com/v1/documents` — creates a new doc with a title
- **Get**: `GET https://docs.googleapis.com/v1/documents/{documentId}` — retrieves doc content
- **Update**: `POST https://docs.googleapis.com/v1/documents/{documentId}:batchUpdate` — applies structural edits (insertText, etc.)
- **List**: `GET https://www.googleapis.com/drive/v3/files?q=mimeType='application/vnd.google-apps.document'` — uses Drive API to list docs

### Microsoft Graph API Endpoints
- **Create**: `PUT /me/drive/root:/{filename}.docx:/content` — upload a new Word file to OneDrive root (or a specified folder)
- **Get**: `GET /me/drive/items/{itemId}` for metadata, `GET /me/drive/items/{itemId}/content` for downloading content
- **Update**: `PUT /me/drive/items/{itemId}/content` — replace file content
- **List**: `GET /me/drive/root/search(q='.docx')` or `GET /me/drive/root/children?$filter=...` — list Word documents

## Implementation Steps

### Step 1: Update Google Connector Manifest

**File**: `connectors/google/google.go`

1. Add a `docsBaseURL` field to `GoogleConnector` struct (default: `https://docs.googleapis.com`)
2. Add a `driveBaseURL` field to `GoogleConnector` struct (default: `https://www.googleapis.com/drive/v3`)
3. Update `New()` and `newForTest()` to accept the new base URLs
4. Add 4 new `ManifestAction` entries to `Manifest()` with parameter schemas
5. Update connector description: `"Google integration for Gmail, Calendar, and Docs"`
6. Add OAuth scope: `https://www.googleapis.com/auth/documents` and `https://www.googleapis.com/auth/drive.readonly`
7. Register action handlers in `Actions()` map
8. Add templates for each new action

### Step 2: Implement Google Docs Action Handlers

Each action gets its own file following the existing pattern (struct, params, validate, Execute):

**`connectors/google/create_document.go`**
- Params: `title` (required), `body` (optional initial text content)
- API: `POST /v1/documents` with `{"title": "..."}`, then optionally `batchUpdate` to insert body text
- Returns: `{document_id, title, document_url}`

**`connectors/google/get_document.go`**
- Params: `document_id` (required)
- API: `GET /v1/documents/{documentId}`
- Returns: `{document_id, title, body_text}` (extracts plain text from the structural content)

**`connectors/google/update_document.go`**
- Params: `document_id` (required), `text` (required), `index` (optional, defaults to end-of-doc)
- API: `POST /v1/documents/{documentId}:batchUpdate` with `insertText` request
- Returns: `{document_id, status}`

**`connectors/google/list_documents.go`**
- Params: `query` (optional search term), `max_results` (optional, default 10, max 100)
- API: `GET /drive/v3/files?q=mimeType='application/vnd.google-apps.document'` (appends user query with `and name contains '...'`)
- Returns: `{documents: [{id, name, created_time, modified_time, web_view_link}]}`

### Step 3: Write Google Docs Action Tests

Each action gets a `_test.go` file following the same pattern as `send_email_test.go`:

- **`connectors/google/create_document_test.go`**: success, missing title, auth failure, rate limit, invalid JSON
- **`connectors/google/get_document_test.go`**: success, missing document_id, not found (404), auth failure
- **`connectors/google/update_document_test.go`**: success, missing document_id, missing text, not found, auth failure
- **`connectors/google/list_documents_test.go`**: success, empty results, auth failure, rate limit

### Step 4: Update Microsoft Connector Manifest

**File**: `connectors/microsoft/microsoft.go`

1. Add 4 new `ManifestAction` entries to `Manifest()` with parameter schemas
2. Update connector description: `"Microsoft 365 integration for email, calendar, and Word documents via Microsoft Graph API"`
3. Add OAuth scope: `Files.ReadWrite` (covers OneDrive file operations including Word docs)
4. Register action handlers in `Actions()` map
5. Add templates for each new action

### Step 5: Implement Microsoft Word Action Handlers

**`connectors/microsoft/create_document.go`**
- Params: `filename` (required, `.docx` appended if missing), `folder_path` (optional, defaults to root), `content` (optional initial text)
- API: `PUT /me/drive/root:/{folder_path}/{filename}:/content` with empty/initial docx content
- Returns: `{id, name, web_url, created_date_time}`

**`connectors/microsoft/get_document.go`**
- Params: `item_id` (required)
- API: `GET /me/drive/items/{itemId}` for metadata
- Returns: `{id, name, web_url, size, created_date_time, last_modified_date_time}`

**`connectors/microsoft/update_document.go`**
- Params: `item_id` (required), `content` (required — raw file content to upload)
- API: `PUT /me/drive/items/{itemId}/content`
- Returns: `{id, name, web_url, last_modified_date_time}`

**`connectors/microsoft/list_documents.go`**
- Params: `folder_path` (optional, defaults to root), `top` (optional, default 10, max 50)
- API: `GET /me/drive/root:/{folder_path}:/children?$filter=file/mimeType eq 'application/vnd.openxmlformats-officedocument.wordprocessingml.document'&$top={top}` (or root children if no folder)
- Returns: `{documents: [{id, name, web_url, size, last_modified_date_time}]}`

### Step 6: Write Microsoft Word Action Tests

Same test pattern as the Google tests:

- **`connectors/microsoft/create_document_test.go`**: success, missing filename, auth failure, rate limit
- **`connectors/microsoft/get_document_test.go`**: success, missing item_id, not found, auth failure
- **`connectors/microsoft/update_document_test.go`**: success, missing item_id, missing content, not found, auth failure
- **`connectors/microsoft/list_documents_test.go`**: success, empty results, auth failure, rate limit

### Step 7: Update Manifest Test

**File**: `connectors/google/google_test.go` and `connectors/microsoft/microsoft_test.go`

Update the manifest validation tests to account for the new action count (Google: 4→8, Microsoft: 4→8).

### Step 8: Run Tests & Build

1. `make test-backend` — ensure all Go tests pass
2. `make build` — ensure TypeScript compilation succeeds (in case frontend references connector types)

## Files Created (New)

| File | Purpose |
|---|---|
| `connectors/google/create_document.go` | Google Docs create action |
| `connectors/google/create_document_test.go` | Tests |
| `connectors/google/get_document.go` | Google Docs get action |
| `connectors/google/get_document_test.go` | Tests |
| `connectors/google/update_document.go` | Google Docs update action |
| `connectors/google/update_document_test.go` | Tests |
| `connectors/google/list_documents.go` | Google Docs list action |
| `connectors/google/list_documents_test.go` | Tests |
| `connectors/microsoft/create_document.go` | Microsoft Word create action |
| `connectors/microsoft/create_document_test.go` | Tests |
| `connectors/microsoft/get_document.go` | Microsoft Word get action |
| `connectors/microsoft/get_document_test.go` | Tests |
| `connectors/microsoft/update_document.go` | Microsoft Word update action |
| `connectors/microsoft/update_document_test.go` | Tests |
| `connectors/microsoft/list_documents.go` | Microsoft Word list action |
| `connectors/microsoft/list_documents_test.go` | Tests |

## Files Modified (Existing)

| File | Changes |
|---|---|
| `connectors/google/google.go` | Add docsBaseURL, driveBaseURL fields; 4 new actions in Manifest(); 4 new handlers in Actions(); new templates; new OAuth scopes |
| `connectors/microsoft/microsoft.go` | 4 new actions in Manifest(); 4 new handlers in Actions(); new templates; new OAuth scope (Files.ReadWrite) |
| `connectors/google/google_test.go` | Update expected action count |
| `connectors/microsoft/microsoft_test.go` | Update expected action count |

## No Database Migrations Required

The connector manifest system handles action registration automatically via `UpsertConnectorFromManifest()` on server startup. New actions are added to the `connector_actions` table automatically — no manual migration needed.

## OAuth Scope Additions

### Google
- Existing: `gmail.send`, `gmail.readonly`, `calendar.events`
- Adding: `https://www.googleapis.com/auth/documents`, `https://www.googleapis.com/auth/drive.readonly`

### Microsoft
- Existing: `Mail.Send`, `Mail.Read`, `Calendars.ReadWrite`
- Adding: `Files.ReadWrite`

## Template Definitions

### Google Templates
- `tpl_google_create_document` — "Create documents freely" (title/body wildcarded)
- `tpl_google_get_document` — "Read any document" (document_id wildcarded)
- `tpl_google_update_document` — "Edit any document" (all wildcarded)
- `tpl_google_list_documents` — "Search documents" (query/max_results wildcarded)

### Microsoft Templates
- `tpl_microsoft_create_document` — "Create Word documents" (filename/content wildcarded)
- `tpl_microsoft_get_document` — "Read any document" (item_id wildcarded)
- `tpl_microsoft_update_document` — "Edit any document" (all wildcarded)
- `tpl_microsoft_list_documents` — "Browse documents" (folder_path/top wildcarded)

## Risks & Tradeoffs

1. **Google Docs content model is complex** — The Docs API returns a structural representation (paragraphs, text runs, etc.), not plain text. The `get_document` action will extract plain text from this structure, which loses formatting. This is a reasonable MVP; richer content handling can be added later.

2. **Microsoft Word binary format** — Word documents are binary (.docx = zipped XML). The `create_document` action will need to either upload a minimal valid .docx or use an empty content body and let Graph handle it. The `update_document` action replaces the entire file content, which is appropriate for simple use cases.

3. **OAuth scope expansion** — Adding new scopes means existing OAuth tokens won't have the new permissions. Users will need to re-authorize. This is expected behavior and the OAuth flow will automatically prompt for the new scopes.

4. **No migration needed** — The manifest upsert system handles everything, which is a major advantage of the existing architecture.
