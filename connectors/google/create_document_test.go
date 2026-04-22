package google

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestCreateDocument_Success(t *testing.T) {
	t.Parallel()

	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer ya29.test-access-token-123" {
			t.Errorf("expected Bearer token, got %q", got)
		}

		calls++
		if r.URL.Path == "/v1/documents" {
			var body docsCreateRequest
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("failed to decode request body: %v", err)
			}
			if body.Title != "My New Doc" {
				t.Errorf("expected title 'My New Doc', got %q", body.Title)
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(docsCreateResponse{
				DocumentID: "doc-abc-123",
				Title:      "My New Doc",
			})
		} else if r.URL.Path == "/v1/documents/doc-abc-123:batchUpdate" {
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{}`))
		} else {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	conn := newForTestDocs(srv.Client(), srv.URL, "")
	action := &createDocumentAction{conn: conn}

	params, _ := json.Marshal(createDocumentParams{
		Title:   "My New Doc",
		Content: "Hello, world!",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.create_document",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]string
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data["document_id"] != "doc-abc-123" {
		t.Errorf("expected document_id 'doc-abc-123', got %q", data["document_id"])
	}
	if data["title"] != "My New Doc" {
		t.Errorf("expected title 'My New Doc', got %q", data["title"])
	}
	if data["document_url"] != "https://docs.google.com/document/d/doc-abc-123/edit" {
		t.Errorf("unexpected document_url: %q", data["document_url"])
	}
	if calls != 2 {
		t.Errorf("expected 2 API calls (create + batchUpdate), got %d", calls)
	}
}

func TestCreateDocument_DeprecatedBodyAlias(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/documents" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(docsCreateResponse{
				DocumentID: "doc-alias",
				Title:      "Alias Doc",
			})
		} else if r.URL.Path == "/v1/documents/doc-alias:batchUpdate" {
			var batch docsBatchUpdateRequest
			if err := json.NewDecoder(r.Body).Decode(&batch); err != nil {
				t.Fatalf("decode batch: %v", err)
			}
			if batch.Requests[0].InsertText == nil || batch.Requests[0].InsertText.Text != "from body" {
				t.Fatalf("expected InsertText from deprecated body, got %+v", batch.Requests[0].InsertText)
			}
			w.Header().Set("Content-Type", "application/json")
			w.Write([]byte(`{}`))
		} else {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
	}))
	defer srv.Close()

	conn := newForTestDocs(srv.Client(), srv.URL, "")
	action := &createDocumentAction{conn: conn}

	params, _ := json.Marshal(map[string]string{"title": "Alias Doc", "body": "from body"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.create_document",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreateDocument_SuccessNoBody(t *testing.T) {
	t.Parallel()

	calls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.URL.Path != "/v1/documents" {
			t.Errorf("unexpected path for no-body create: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(docsCreateResponse{
			DocumentID: "doc-xyz",
			Title:      "Empty Doc",
		})
	}))
	defer srv.Close()

	conn := newForTestDocs(srv.Client(), srv.URL, "")
	action := &createDocumentAction{conn: conn}

	params, _ := json.Marshal(createDocumentParams{Title: "Empty Doc"})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.create_document",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]string
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data["document_id"] != "doc-xyz" {
		t.Errorf("expected document_id 'doc-xyz', got %q", data["document_id"])
	}
	if calls != 1 {
		t.Errorf("expected 1 API call (create only), got %d", calls)
	}
}

func TestCreateDocument_MissingTitle(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createDocumentAction{conn: conn}

	params, _ := json.Marshal(map[string]string{"body": "some text"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.create_document",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing title")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCreateDocument_AuthFailure(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"code": 401, "message": "Invalid Credentials"},
		})
	}))
	defer srv.Close()

	conn := newForTestDocs(srv.Client(), srv.URL, "")
	action := &createDocumentAction{conn: conn}

	params, _ := json.Marshal(createDocumentParams{Title: "Test"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.create_document",
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

func TestCreateDocument_RateLimit(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "60")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	conn := newForTestDocs(srv.Client(), srv.URL, "")
	action := &createDocumentAction{conn: conn}

	params, _ := json.Marshal(createDocumentParams{Title: "Test"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.create_document",
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

func TestCreateDocument_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createDocumentAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.create_document",
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

func TestCreateDocument_BodyInsertionFailure(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/v1/documents" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(docsCreateResponse{
				DocumentID: "doc-partial",
				Title:      "Partial Doc",
			})
		} else if r.URL.Path == "/v1/documents/doc-partial:batchUpdate" {
			// batchUpdate fails
			w.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(w).Encode(map[string]any{
				"error": map[string]any{"code": 500, "message": "Internal error"},
			})
		}
	}))
	defer srv.Close()

	conn := newForTestDocs(srv.Client(), srv.URL, "")
	action := &createDocumentAction{conn: conn}

	params, _ := json.Marshal(createDocumentParams{
		Title:   "Partial Doc",
		Content: "some text",
	})

	// Should succeed with a warning, not error — the document was created.
	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.create_document",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("expected no error (partial success), got: %v", err)
	}

	var data map[string]string
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data["document_id"] != "doc-partial" {
		t.Errorf("expected document_id 'doc-partial', got %q", data["document_id"])
	}
	if data["warning"] == "" {
		t.Error("expected warning about body insertion failure")
	}
}
