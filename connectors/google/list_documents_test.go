package google

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestListDocuments_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if !strings.HasPrefix(r.URL.Path, "/drive/v3/files") {
			t.Errorf("expected path /drive/v3/files, got %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer ya29.test-access-token-123" {
			t.Errorf("expected Bearer token, got %q", got)
		}

		q := r.URL.Query().Get("q")
		if !strings.Contains(q, "mimeType='application/vnd.google-apps.document'") {
			t.Errorf("expected mimeType filter in query, got %q", q)
		}
		if !strings.Contains(q, "trashed=false") {
			t.Errorf("expected trashed=false filter in query, got %q", q)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(driveListResponse{
			Files: []driveFile{
				{
					ID:           "doc-1",
					Name:         "Meeting Notes",
					CreatedTime:  "2024-01-15T09:00:00Z",
					ModifiedTime: "2024-01-16T10:00:00Z",
					WebViewLink:  "https://docs.google.com/document/d/doc-1/edit",
				},
				{
					ID:           "doc-2",
					Name:         "Project Plan",
					CreatedTime:  "2024-01-10T08:00:00Z",
					ModifiedTime: "2024-01-14T12:00:00Z",
					WebViewLink:  "https://docs.google.com/document/d/doc-2/edit",
				},
			},
		})
	}))
	defer srv.Close()

	conn := newForTestDocs(srv.Client(), "", srv.URL)
	action := &listDocumentsAction{conn: conn}

	params, _ := json.Marshal(listDocumentsParams{MaxResults: 10})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.list_documents",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data struct {
		Documents []documentSummary `json:"documents"`
		Count     int               `json:"count"`
	}
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if len(data.Documents) != 2 {
		t.Fatalf("expected 2 documents, got %d", len(data.Documents))
	}
	if data.Count != 2 {
		t.Errorf("expected count 2, got %d", data.Count)
	}
	if data.Documents[0].Name != "Meeting Notes" {
		t.Errorf("expected first doc name 'Meeting Notes', got %q", data.Documents[0].Name)
	}
	if data.Documents[1].ID != "doc-2" {
		t.Errorf("expected second doc ID 'doc-2', got %q", data.Documents[1].ID)
	}
}

func TestListDocuments_WithQuery(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query().Get("q")
		if !strings.Contains(q, "name contains 'meeting'") {
			t.Errorf("expected name filter in query, got %q", q)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(driveListResponse{
			Files: []driveFile{
				{ID: "doc-1", Name: "Meeting Notes"},
			},
		})
	}))
	defer srv.Close()

	conn := newForTestDocs(srv.Client(), "", srv.URL)
	action := &listDocumentsAction{conn: conn}

	params, _ := json.Marshal(listDocumentsParams{Query: "meeting", MaxResults: 5})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.list_documents",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data struct {
		Documents []documentSummary `json:"documents"`
	}
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if len(data.Documents) != 1 {
		t.Fatalf("expected 1 document, got %d", len(data.Documents))
	}
}

func TestListDocuments_EmptyResult(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(driveListResponse{Files: nil})
	}))
	defer srv.Close()

	conn := newForTestDocs(srv.Client(), "", srv.URL)
	action := &listDocumentsAction{conn: conn}

	params, _ := json.Marshal(listDocumentsParams{})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.list_documents",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data struct {
		Documents []documentSummary `json:"documents"`
	}
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if len(data.Documents) != 0 {
		t.Errorf("expected 0 documents, got %d", len(data.Documents))
	}
}

func TestListDocuments_QueryEscaping(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query().Get("q")
		// The single quote in "it's" should be escaped.
		if !strings.Contains(q, `name contains 'it\'s'`) {
			t.Errorf("expected escaped query, got %q", q)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(driveListResponse{Files: nil})
	}))
	defer srv.Close()

	conn := newForTestDocs(srv.Client(), "", srv.URL)
	action := &listDocumentsAction{conn: conn}

	params, _ := json.Marshal(listDocumentsParams{Query: "it's"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.list_documents",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListDocuments_AuthFailure(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"code": 401, "message": "Invalid Credentials"},
		})
	}))
	defer srv.Close()

	conn := newForTestDocs(srv.Client(), "", srv.URL)
	action := &listDocumentsAction{conn: conn}

	params, _ := json.Marshal(listDocumentsParams{})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.list_documents",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for auth failure")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError, got: %T (%v)", err, err)
	}
}

func TestListDocuments_RateLimit(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "60")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	conn := newForTestDocs(srv.Client(), "", srv.URL)
	action := &listDocumentsAction{conn: conn}

	params, _ := json.Marshal(listDocumentsParams{})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.list_documents",
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

func TestListDocuments_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &listDocumentsAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.list_documents",
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

func TestEscapeDriveQuery(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input string
		want  string
	}{
		{"simple", "simple"},
		{"it's", `it\'s`},
		{`back\slash`, `back\\slash`},
		{`it's a "test"`, `it\'s a "test"`},
	}

	for _, tt := range tests {
		got := escapeDriveQuery(tt.input)
		if got != tt.want {
			t.Errorf("escapeDriveQuery(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
