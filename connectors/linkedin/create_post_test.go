package linkedin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestCreatePost_Success(t *testing.T) {
	t.Parallel()

	var gotBody linkedInPostRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/userinfo":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(userinfoResponse{Sub: "person123"})
		case "/posts":
			if r.Method != http.MethodPost {
				t.Errorf("expected POST, got %s", r.Method)
			}
			if got := r.Header.Get("LinkedIn-Version"); got != linkedInVersion {
				t.Errorf("expected LinkedIn-Version %q, got %q", linkedInVersion, got)
			}
			if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
				t.Fatalf("failed to decode request body: %v", err)
			}
			w.WriteHeader(http.StatusCreated)
		default:
			t.Errorf("unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL, srv.URL)
	action := &createPostAction{conn: conn}

	params, _ := json.Marshal(createPostParams{
		Text:       "Hello LinkedIn!",
		Visibility: "CONNECTIONS",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linkedin.create_post",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify request body
	if gotBody.Author != "urn:li:person:person123" {
		t.Errorf("expected author 'urn:li:person:person123', got %q", gotBody.Author)
	}
	if gotBody.Commentary != "Hello LinkedIn!" {
		t.Errorf("expected commentary 'Hello LinkedIn!', got %q", gotBody.Commentary)
	}
	if gotBody.Visibility != "CONNECTIONS" {
		t.Errorf("expected visibility 'CONNECTIONS', got %q", gotBody.Visibility)
	}
	if gotBody.LifecycleState != "PUBLISHED" {
		t.Errorf("expected lifecycleState 'PUBLISHED', got %q", gotBody.LifecycleState)
	}
	if gotBody.Content != nil {
		t.Error("expected nil content for text-only post")
	}

	var data map[string]string
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data["status"] != "created" {
		t.Errorf("expected status 'created', got %q", data["status"])
	}
}

func TestCreatePost_WithArticle(t *testing.T) {
	t.Parallel()

	var gotBody linkedInPostRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/userinfo":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(userinfoResponse{Sub: "person123"})
		case "/posts":
			json.NewDecoder(r.Body).Decode(&gotBody)
			w.WriteHeader(http.StatusCreated)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL, srv.URL)
	action := &createPostAction{conn: conn}

	params, _ := json.Marshal(createPostParams{
		Text:               "Check out this article",
		ArticleURL:         "https://example.com/article",
		ArticleTitle:       "Great Article",
		ArticleDescription: "A must-read piece",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linkedin.create_post",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotBody.Content == nil || gotBody.Content.Article == nil {
		t.Fatal("expected article content in request body")
	}
	if gotBody.Content.Article.Source != "https://example.com/article" {
		t.Errorf("expected article source 'https://example.com/article', got %q", gotBody.Content.Article.Source)
	}
	if gotBody.Content.Article.Title != "Great Article" {
		t.Errorf("expected article title 'Great Article', got %q", gotBody.Content.Article.Title)
	}
}

func TestCreatePost_MissingText(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createPostAction{conn: conn}

	params, _ := json.Marshal(map[string]string{"visibility": "PUBLIC"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linkedin.create_post",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing text")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCreatePost_TextTooLong(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createPostAction{conn: conn}

	params, _ := json.Marshal(createPostParams{
		Text: strings.Repeat("a", maxPostTextLen+1),
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linkedin.create_post",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for text too long")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCreatePost_InvalidVisibility(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createPostAction{conn: conn}

	params, _ := json.Marshal(createPostParams{
		Text:       "Hello",
		Visibility: "PRIVATE",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linkedin.create_post",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid visibility")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCreatePost_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createPostAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linkedin.create_post",
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

func TestCreatePost_DefaultVisibility(t *testing.T) {
	t.Parallel()

	var gotBody linkedInPostRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/userinfo":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(userinfoResponse{Sub: "person123"})
		case "/posts":
			json.NewDecoder(r.Body).Decode(&gotBody)
			w.WriteHeader(http.StatusCreated)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL, srv.URL)
	action := &createPostAction{conn: conn}

	params, _ := json.Marshal(createPostParams{Text: "Hello"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linkedin.create_post",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotBody.Visibility != "PUBLIC" {
		t.Errorf("expected default visibility 'PUBLIC', got %q", gotBody.Visibility)
	}
}
