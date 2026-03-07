package microsoft

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestCreateDocument_Success(t *testing.T) {
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
		if r.URL.Path != "/me/drive/root:/report.docx:/content" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		body, _ := io.ReadAll(r.Body)
		if string(body) != "Hello World" {
			t.Errorf("expected body 'Hello World', got %q", string(body))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":              "item-123",
			"name":            "report.docx",
			"webUrl":          "https://onedrive.live.com/report.docx",
			"createdDateTime": "2024-01-15T09:00:00Z",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &createDocumentAction{conn: conn}

	params, _ := json.Marshal(createDocumentParams{
		Filename: "report",
		Content:  "Hello World",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.create_document",
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
	if doc.WebURL != "https://onedrive.live.com/report.docx" {
		t.Errorf("unexpected web_url: %q", doc.WebURL)
	}
}

func TestCreateDocument_WithFolderPath(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/me/drive/root:/Documents/Work/report.docx:/content" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":              "item-456",
			"name":            "report.docx",
			"webUrl":          "https://onedrive.live.com/report.docx",
			"createdDateTime": "2024-01-15T09:00:00Z",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &createDocumentAction{conn: conn}

	params, _ := json.Marshal(createDocumentParams{
		Filename:   "report.docx",
		FolderPath: "Documents/Work",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.create_document",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateDocument_FilenameWithSpecialChars(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// The filename is percent-encoded at the wire level; Go's URL parser
		// decodes it in r.URL.Path. Seeing the decoded form here confirms
		// the correct endpoint was reached.
		if r.URL.Path != "/me/drive/root:/Q&A Notes.docx:/content" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":              "item-789",
			"name":            "Q&A Notes.docx",
			"webUrl":          "https://onedrive.live.com/qanda.docx",
			"createdDateTime": "2024-01-15T09:00:00Z",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &createDocumentAction{conn: conn}

	params, _ := json.Marshal(createDocumentParams{Filename: "Q&A Notes.docx"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.create_document",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateDocument_MissingFilename(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createDocumentAction{conn: conn}

	params, _ := json.Marshal(createDocumentParams{})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.create_document",
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

func TestCreateDocument_FilenamePathTraversal(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createDocumentAction{conn: conn}

	params, _ := json.Marshal(createDocumentParams{Filename: "../evil.docx"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.create_document",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for path traversal filename")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCreateDocument_FolderPathTraversal(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createDocumentAction{conn: conn}

	params, _ := json.Marshal(createDocumentParams{
		Filename:   "report.docx",
		FolderPath: "../../admin",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.create_document",
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

func TestCreateDocument_DocxAppended(t *testing.T) {
	t.Parallel()

	params := createDocumentParams{Filename: "report"}
	params.defaults()
	if params.Filename != "report.docx" {
		t.Errorf("expected 'report.docx', got %q", params.Filename)
	}
}

func TestCreateDocument_DocxNotDuplicated(t *testing.T) {
	t.Parallel()

	params := createDocumentParams{Filename: "report.docx"}
	params.defaults()
	if params.Filename != "report.docx" {
		t.Errorf("expected 'report.docx', got %q", params.Filename)
	}
}

func TestCreateDocument_WhitespaceOnlyFilename(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createDocumentAction{conn: conn}

	params, _ := json.Marshal(createDocumentParams{Filename: "   "})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.create_document",
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

func TestCreateDocument_ContentTooLarge(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createDocumentAction{conn: conn}

	// Create content just over the 4MB limit.
	bigContent := string(make([]byte, maxSimpleUploadSize+1))
	params, _ := json.Marshal(createDocumentParams{
		Filename: "big.docx",
		Content:  bigContent,
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.create_document",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for oversized content")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCreateDocument_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createDocumentAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.create_document",
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

func TestCreateDocument_AuthError(t *testing.T) {
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
	action := &createDocumentAction{conn: conn}

	params, _ := json.Marshal(createDocumentParams{Filename: "report.docx"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.create_document",
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

func TestCreateDocument_RateLimit(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "60")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &createDocumentAction{conn: conn}

	params, _ := json.Marshal(createDocumentParams{Filename: "report.docx"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.create_document",
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
