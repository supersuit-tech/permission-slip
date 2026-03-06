package microsoft

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestListDocuments_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-access-token-123" {
			t.Errorf("expected Bearer token, got %q", got)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"value": []map[string]any{
				{
					"id":                   "item-1",
					"name":                 "report.docx",
					"webUrl":               "https://onedrive.live.com/report.docx",
					"size":                 12345,
					"lastModifiedDateTime": "2024-01-16T10:00:00Z",
				},
				{
					"id":                   "item-2",
					"name":                 "notes.docx",
					"webUrl":               "https://onedrive.live.com/notes.docx",
					"size":                 6789,
					"lastModifiedDateTime": "2024-01-15T08:00:00Z",
				},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &listDocumentsAction{conn: conn}

	params, _ := json.Marshal(listDocumentsParams{})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.list_documents",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var resp struct {
		Documents []documentListItem `json:"documents"`
	}
	if err := json.Unmarshal(result.Data, &resp); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if len(resp.Documents) != 2 {
		t.Fatalf("expected 2 documents, got %d", len(resp.Documents))
	}
	if resp.Documents[0].ID != "item-1" {
		t.Errorf("expected first doc id 'item-1', got %q", resp.Documents[0].ID)
	}
	if resp.Documents[0].Name != "report.docx" {
		t.Errorf("expected first doc name 'report.docx', got %q", resp.Documents[0].Name)
	}
	if resp.Documents[1].ID != "item-2" {
		t.Errorf("expected second doc id 'item-2', got %q", resp.Documents[1].ID)
	}
}

func TestListDocuments_WithFolderPath(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/me/drive/root:/Documents:/children" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"value": []any{}})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &listDocumentsAction{conn: conn}

	params, _ := json.Marshal(listDocumentsParams{FolderPath: "Documents"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.list_documents",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListDocuments_DefaultParams(t *testing.T) {
	t.Parallel()

	var params listDocumentsParams
	params.defaults()
	if params.Top != 10 {
		t.Errorf("expected default top 10, got %d", params.Top)
	}
}

func TestListDocuments_TopClamped(t *testing.T) {
	t.Parallel()

	params := listDocumentsParams{Top: 100}
	params.defaults()
	if params.Top != 50 {
		t.Errorf("expected top clamped to 50, got %d", params.Top)
	}
}

func TestListDocuments_FolderPathTraversal(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &listDocumentsAction{conn: conn}

	params, _ := json.Marshal(listDocumentsParams{FolderPath: "../../admin"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.list_documents",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for path traversal folder")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestListDocuments_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &listDocumentsAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.list_documents",
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

func TestListDocuments_AuthError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]string{
				"code":    "InvalidAuthenticationToken",
				"message": "Access token is expired",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &listDocumentsAction{conn: conn}

	params, _ := json.Marshal(listDocumentsParams{})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.list_documents",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for 401")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError, got: %T", err)
	}
}
