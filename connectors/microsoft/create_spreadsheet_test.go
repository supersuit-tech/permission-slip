package microsoft

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestCreateSpreadsheet_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if got := r.URL.Path; got != "/me/drive/root:/Budget 2026.xlsx:/content" {
			t.Errorf("expected decoded path /me/drive/root:/Budget 2026.xlsx:/content, got %s", got)
		}
		// Verify percent-encoding via RequestURI (RawPath is empty when
		// the encoding round-trips cleanly, which is standard Go behavior).
		if got := r.RequestURI; got != "/me/drive/root:/Budget%202026.xlsx:/content" {
			t.Errorf("expected encoded request URI /me/drive/root:/Budget%%202026.xlsx:/content, got %s", got)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-access-token-123" {
			t.Errorf("expected Bearer token, got %q", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" {
			t.Errorf("expected XLSX content type, got %q", got)
		}

		body, _ := io.ReadAll(r.Body)
		if len(body) == 0 {
			t.Error("expected non-empty request body with XLSX bytes")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":     "item-abc-123",
			"name":   "Budget 2026.xlsx",
			"webUrl": "https://onedrive.live.com/edit.aspx?id=item-abc-123",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &createSpreadsheetAction{conn: conn}

	params, _ := json.Marshal(createSpreadsheetParams{
		Filename: "Budget 2026",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.create_spreadsheet",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data spreadsheetResult
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data.ItemID != "item-abc-123" {
		t.Errorf("expected item_id 'item-abc-123', got %q", data.ItemID)
	}
	if data.Name != "Budget 2026.xlsx" {
		t.Errorf("expected name 'Budget 2026.xlsx', got %q", data.Name)
	}
	if data.WebURL != "https://onedrive.live.com/edit.aspx?id=item-abc-123" {
		t.Errorf("expected web_url, got %q", data.WebURL)
	}
	if data.FolderPath != "/" {
		t.Errorf("expected folder_path '/', got %q", data.FolderPath)
	}
}

func TestCreateSpreadsheet_WithFolderPath(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Path; got != "/me/drive/root:/Documents/Finance/budget.xlsx:/content" {
			t.Errorf("expected path with folder, got %s", got)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":     "item-456",
			"name":   "budget.xlsx",
			"webUrl": "https://onedrive.live.com/edit.aspx?id=item-456",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &createSpreadsheetAction{conn: conn}

	params, _ := json.Marshal(createSpreadsheetParams{
		Filename:   "budget.xlsx",
		FolderPath: "Documents/Finance",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.create_spreadsheet",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data spreadsheetResult
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data.FolderPath != "/Documents/Finance" {
		t.Errorf("expected folder_path '/Documents/Finance', got %q", data.FolderPath)
	}
}

func TestCreateSpreadsheet_AppendsXlsxExtension(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Path; got != "/me/drive/root:/My Data.xlsx:/content" {
			t.Errorf("expected .xlsx appended, got path %s", got)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":     "item-789",
			"name":   "My Data.xlsx",
			"webUrl": "https://onedrive.live.com/edit.aspx?id=item-789",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &createSpreadsheetAction{conn: conn}

	params, _ := json.Marshal(createSpreadsheetParams{
		Filename: "My Data",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.create_spreadsheet",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateSpreadsheet_MissingFilename(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createSpreadsheetAction{conn: conn}

	params, _ := json.Marshal(createSpreadsheetParams{})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.create_spreadsheet",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing filename")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCreateSpreadsheet_FilenamePathTraversal(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createSpreadsheetAction{conn: conn}

	params, _ := json.Marshal(createSpreadsheetParams{
		Filename: "../../evil.xlsx",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.create_spreadsheet",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for path traversal in filename")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCreateSpreadsheet_FolderPathTraversal(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createSpreadsheetAction{conn: conn}

	params, _ := json.Marshal(createSpreadsheetParams{
		Filename:   "data.xlsx",
		FolderPath: "../../../etc",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.create_spreadsheet",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for path traversal in folder_path")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCreateSpreadsheet_FolderPathSpecialChars(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createSpreadsheetAction{conn: conn}

	params, _ := json.Marshal(createSpreadsheetParams{
		Filename:   "data.xlsx",
		FolderPath: "folder?query=inject",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.create_spreadsheet",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for special chars in folder_path")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCreateSpreadsheet_WhitespaceOnlyFilename(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createSpreadsheetAction{conn: conn}

	params, _ := json.Marshal(createSpreadsheetParams{Filename: "   "})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.create_spreadsheet",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for whitespace-only filename")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCreateSpreadsheet_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createSpreadsheetAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.create_spreadsheet",
		Parameters:  []byte(`{invalid`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCreateSpreadsheet_AuthFailure(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]string{
				"code":    "InvalidAuthenticationToken",
				"message": "Access token has expired",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &createSpreadsheetAction{conn: conn}

	params, _ := json.Marshal(createSpreadsheetParams{
		Filename: "test.xlsx",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.create_spreadsheet",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for auth failure")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError, got: %T", err)
	}
}

func TestCreateSpreadsheet_RateLimit(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "60")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &createSpreadsheetAction{conn: conn}

	params, _ := json.Marshal(createSpreadsheetParams{
		Filename: "test.xlsx",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.create_spreadsheet",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for rate limit")
	}
	if !connectors.IsRateLimitError(err) {
		t.Errorf("expected RateLimitError, got: %T", err)
	}
}
