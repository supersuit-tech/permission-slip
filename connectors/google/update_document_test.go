package google

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestUpdateDocument_DeprecatedTextAlias(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body docsBatchUpdateRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if body.Requests[0].InsertText == nil || body.Requests[0].InsertText.Text != "via text" {
			t.Fatalf("expected text via deprecated alias, got %+v", body.Requests[0].InsertText)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	conn := newForTestDocs(srv.Client(), srv.URL, "")
	action := &updateDocumentAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"document_id": "doc-abc-123",
		"text":        "via text",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.update_document",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUpdateDocument_SuccessAppend(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/v1/documents/doc-abc-123:batchUpdate" {
			t.Errorf("expected path /v1/documents/doc-abc-123:batchUpdate, got %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer ya29.test-access-token-123" {
			t.Errorf("expected Bearer token, got %q", got)
		}

		var body docsBatchUpdateRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		if len(body.Requests) != 1 {
			t.Fatalf("expected 1 request, got %d", len(body.Requests))
		}
		req := body.Requests[0]
		if req.InsertText == nil {
			t.Fatal("expected InsertText request")
		}
		if req.InsertText.Text != "appended text" {
			t.Errorf("expected content 'appended text', got %q", req.InsertText.Text)
		}
		if req.InsertText.EndOfSegmentLocation == nil {
			t.Error("expected EndOfSegmentLocation for append")
		}
		if req.InsertText.Location != nil {
			t.Error("expected nil Location for append")
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	conn := newForTestDocs(srv.Client(), srv.URL, "")
	action := &updateDocumentAction{conn: conn}

	params, _ := json.Marshal(updateDocumentParams{
		DocumentID: "doc-abc-123",
		Content:    "appended text",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.update_document",
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
	if data["status"] != "updated" {
		t.Errorf("expected status 'updated', got %q", data["status"])
	}
}

func TestUpdateDocument_SuccessInsertAtIndex(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var body docsBatchUpdateRequest
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request body: %v", err)
		}
		req := body.Requests[0]
		if req.InsertText.Location == nil {
			t.Fatal("expected Location for index insert")
		}
		if req.InsertText.Location.Index != 5 {
			t.Errorf("expected index 5, got %d", req.InsertText.Location.Index)
		}
		if req.InsertText.EndOfSegmentLocation != nil {
			t.Error("expected nil EndOfSegmentLocation for index insert")
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{}`))
	}))
	defer srv.Close()

	conn := newForTestDocs(srv.Client(), srv.URL, "")
	action := &updateDocumentAction{conn: conn}

	params, _ := json.Marshal(updateDocumentParams{
		DocumentID: "doc-abc-123",
		Content:    "inserted text",
		Index:      5,
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.update_document",
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
	if data["status"] != "updated" {
		t.Errorf("expected status 'updated', got %q", data["status"])
	}
}

func TestUpdateDocument_MissingDocumentID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &updateDocumentAction{conn: conn}

	params, _ := json.Marshal(map[string]string{"text": "hello"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.update_document",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing document_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestUpdateDocument_MissingContent(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &updateDocumentAction{conn: conn}

	params, _ := json.Marshal(map[string]string{"document_id": "doc-123"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.update_document",
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

func TestUpdateDocument_NegativeIndex(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &updateDocumentAction{conn: conn}

	params, _ := json.Marshal(map[string]any{
		"document_id": "doc-123",
		"content":     "hello",
		"index":       -1,
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.update_document",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for negative index")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestUpdateDocument_AuthFailure(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"code": 401, "message": "Invalid Credentials"},
		})
	}))
	defer srv.Close()

	conn := newForTestDocs(srv.Client(), srv.URL, "")
	action := &updateDocumentAction{conn: conn}

	params, _ := json.Marshal(updateDocumentParams{DocumentID: "doc-123", Content: "hello"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.update_document",
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

func TestUpdateDocument_RateLimit(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "60")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	conn := newForTestDocs(srv.Client(), srv.URL, "")
	action := &updateDocumentAction{conn: conn}

	params, _ := json.Marshal(updateDocumentParams{DocumentID: "doc-123", Content: "hello"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.update_document",
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

func TestUpdateDocument_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &updateDocumentAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.update_document",
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
