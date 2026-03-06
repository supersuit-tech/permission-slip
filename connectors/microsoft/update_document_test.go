package microsoft

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestUpdateDocument_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-access-token-123" {
			t.Errorf("expected Bearer token, got %q", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/vnd.openxmlformats-officedocument.wordprocessingml.document" {
			t.Errorf("expected docx content type, got %q", got)
		}
		if r.URL.Path != "/me/drive/items/item-123/content" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		body, _ := io.ReadAll(r.Body)
		if string(body) != "Updated content" {
			t.Errorf("expected body 'Updated content', got %q", string(body))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":                   "item-123",
			"name":                 "report.docx",
			"webUrl":               "https://onedrive.live.com/report.docx",
			"lastModifiedDateTime": "2024-01-16T10:00:00Z",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &updateDocumentAction{conn: conn}

	params, _ := json.Marshal(updateDocumentParams{
		ItemID:  "item-123",
		Content: "Updated content",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.update_document",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var doc documentResult
	if err := json.Unmarshal(result.Data, &doc); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if doc.ID != "item-123" {
		t.Errorf("expected id 'item-123', got %q", doc.ID)
	}
	if doc.Name != "report.docx" {
		t.Errorf("expected name 'report.docx', got %q", doc.Name)
	}
	if doc.LastModifiedDateTime != "2024-01-16T10:00:00Z" {
		t.Errorf("unexpected last_modified_date_time: %q", doc.LastModifiedDateTime)
	}
}

func TestUpdateDocument_MissingItemID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &updateDocumentAction{conn: conn}

	params, _ := json.Marshal(updateDocumentParams{Content: "some content"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.update_document",
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

func TestUpdateDocument_MissingContent(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &updateDocumentAction{conn: conn}

	params, _ := json.Marshal(updateDocumentParams{ItemID: "item-123"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.update_document",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing content")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestUpdateDocument_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &updateDocumentAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.update_document",
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

func TestUpdateDocument_AuthError(t *testing.T) {
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
	action := &updateDocumentAction{conn: conn}

	params, _ := json.Marshal(updateDocumentParams{
		ItemID:  "item-123",
		Content: "content",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.update_document",
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

func TestUpdateDocument_RateLimit(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &updateDocumentAction{conn: conn}

	params, _ := json.Marshal(updateDocumentParams{
		ItemID:  "item-123",
		Content: "content",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.update_document",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for 429")
	}
	if !connectors.IsRateLimitError(err) {
		t.Errorf("expected RateLimitError, got: %T", err)
	}
}
