package google

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
		if r.URL.Path != "/v1/documents/doc-abc-123" {
			t.Errorf("expected path /v1/documents/doc-abc-123, got %s", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer ya29.test-access-token-123" {
			t.Errorf("expected Bearer token, got %q", got)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(docsGetResponse{
			DocumentID: "doc-abc-123",
			Title:      "Test Doc",
			Body: docsBody{
				Content: []docsStructuralElement{
					{
						Paragraph: &docsParagraph{
							Elements: []docsParagraphElement{
								{TextRun: &docsTextRun{Content: "Hello, "}},
								{TextRun: &docsTextRun{Content: "world!\n"}},
							},
						},
					},
					{
						Paragraph: &docsParagraph{
							Elements: []docsParagraphElement{
								{TextRun: &docsTextRun{Content: "Second paragraph.\n"}},
							},
						},
					},
				},
			},
		})
	}))
	defer srv.Close()

	conn := newForTestDocs(srv.Client(), srv.URL, "")
	action := &getDocumentAction{conn: conn}

	params, _ := json.Marshal(getDocumentParams{DocumentID: "doc-abc-123"})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.get_document",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data["document_id"] != "doc-abc-123" {
		t.Errorf("expected document_id 'doc-abc-123', got %q", data["document_id"])
	}
	if data["title"] != "Test Doc" {
		t.Errorf("expected title 'Test Doc', got %q", data["title"])
	}
	if data["body_text"] != "Hello, world!\nSecond paragraph.\n" {
		t.Errorf("unexpected body_text: %q", data["body_text"])
	}
	if data["document_url"] != "https://docs.google.com/document/d/doc-abc-123/edit" {
		t.Errorf("unexpected document_url: %q", data["document_url"])
	}
	// word_count: "Hello," "world!" "Second" "paragraph." = 4 words
	if wc, ok := data["word_count"].(float64); !ok || int(wc) != 4 {
		t.Errorf("expected word_count 4, got %v", data["word_count"])
	}
}

func TestGetDocument_EmptyDocument(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(docsGetResponse{
			DocumentID: "doc-empty",
			Title:      "Empty Doc",
			Body:       docsBody{Content: nil},
		})
	}))
	defer srv.Close()

	conn := newForTestDocs(srv.Client(), srv.URL, "")
	action := &getDocumentAction{conn: conn}

	params, _ := json.Marshal(getDocumentParams{DocumentID: "doc-empty"})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.get_document",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data["body_text"] != "" {
		t.Errorf("expected empty body_text, got %q", data["body_text"])
	}
	if wc, ok := data["word_count"].(float64); !ok || int(wc) != 0 {
		t.Errorf("expected word_count 0, got %v", data["word_count"])
	}
}

func TestGetDocument_MissingDocumentID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &getDocumentAction{conn: conn}

	params, _ := json.Marshal(map[string]string{})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.get_document",
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

func TestGetDocument_AuthFailure(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{"code": 401, "message": "Invalid Credentials"},
		})
	}))
	defer srv.Close()

	conn := newForTestDocs(srv.Client(), srv.URL, "")
	action := &getDocumentAction{conn: conn}

	params, _ := json.Marshal(getDocumentParams{DocumentID: "doc-123"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.get_document",
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

func TestGetDocument_RateLimit(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Retry-After", "30")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	conn := newForTestDocs(srv.Client(), srv.URL, "")
	action := &getDocumentAction{conn: conn}

	params, _ := json.Marshal(getDocumentParams{DocumentID: "doc-123"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.get_document",
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

func TestGetDocument_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &getDocumentAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "google.get_document",
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
