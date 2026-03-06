package microsoft

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestGetDocument_Success(t *testing.T) {
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
			"id":                   "item-123",
			"name":                 "report.docx",
			"webUrl":               "https://onedrive.live.com/report.docx",
			"size":                 12345,
			"createdDateTime":      "2024-01-15T09:00:00Z",
			"lastModifiedDateTime": "2024-01-16T10:00:00Z",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &getDocumentAction{conn: conn}

	params, _ := json.Marshal(getDocumentParams{ItemID: "item-123"})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.get_document",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var doc documentMetadata
	if err := json.Unmarshal(result.Data, &doc); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if doc.ID != "item-123" {
		t.Errorf("expected id 'item-123', got %q", doc.ID)
	}
	if doc.Name != "report.docx" {
		t.Errorf("expected name 'report.docx', got %q", doc.Name)
	}
	if doc.Size != 12345 {
		t.Errorf("expected size 12345, got %d", doc.Size)
	}
	if doc.CreatedDateTime != "2024-01-15T09:00:00Z" {
		t.Errorf("unexpected created_date_time: %q", doc.CreatedDateTime)
	}
	if doc.LastModifiedDateTime != "2024-01-16T10:00:00Z" {
		t.Errorf("unexpected last_modified_date_time: %q", doc.LastModifiedDateTime)
	}
}

func TestGetDocument_MissingItemID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &getDocumentAction{conn: conn}

	params, _ := json.Marshal(getDocumentParams{})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.get_document",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing item_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestGetDocument_NotFound(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]string{
				"code":    "itemNotFound",
				"message": "The resource could not be found",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &getDocumentAction{conn: conn}

	params, _ := json.Marshal(getDocumentParams{ItemID: "nonexistent"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.get_document",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for 404")
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError, got: %T", err)
	}
}

func TestGetDocument_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &getDocumentAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.get_document",
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

func TestGetDocument_AuthError(t *testing.T) {
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
	action := &getDocumentAction{conn: conn}

	params, _ := json.Marshal(getDocumentParams{ItemID: "item-123"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.get_document",
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
