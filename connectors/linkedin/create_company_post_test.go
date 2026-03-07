package linkedin

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestCreateCompanyPost_Success(t *testing.T) {
	t.Parallel()

	var gotBody linkedInPostRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/posts" {
			t.Errorf("expected path /posts, got %s", r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if got := r.Header.Get("LinkedIn-Version"); got != linkedInVersion {
			t.Errorf("expected LinkedIn-Version %q, got %q", linkedInVersion, got)
		}
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.Header().Set("x-restli-id", "urn:li:share:9999999")
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL, srv.URL)
	action := &createCompanyPostAction{conn: conn}

	params, _ := json.Marshal(createCompanyPostParams{
		OrganizationID: "12345",
		Text:           "Company update!",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linkedin.create_company_post",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotBody.Author != "urn:li:organization:12345" {
		t.Errorf("expected author 'urn:li:organization:12345', got %q", gotBody.Author)
	}
	if gotBody.Commentary != "Company update!" {
		t.Errorf("expected commentary 'Company update!', got %q", gotBody.Commentary)
	}
	if gotBody.Visibility != "PUBLIC" {
		t.Errorf("expected visibility 'PUBLIC', got %q", gotBody.Visibility)
	}

	var data map[string]string
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data["status"] != "created" {
		t.Errorf("expected status 'created', got %q", data["status"])
	}
	if data["organization_id"] != "12345" {
		t.Errorf("expected organization_id '12345', got %q", data["organization_id"])
	}
	if data["post_urn"] != "urn:li:share:9999999" {
		t.Errorf("expected post_urn 'urn:li:share:9999999', got %q", data["post_urn"])
	}
}

func TestCreateCompanyPost_WithArticle(t *testing.T) {
	t.Parallel()

	var gotBody linkedInPostRequest
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewDecoder(r.Body).Decode(&gotBody)
		w.WriteHeader(http.StatusCreated)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL, srv.URL)
	action := &createCompanyPostAction{conn: conn}

	params, _ := json.Marshal(createCompanyPostParams{
		OrganizationID: "12345",
		Text:           "Check this out",
		ArticleURL:     "https://example.com/blog",
		ArticleTitle:   "Blog Post",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linkedin.create_company_post",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if gotBody.Content == nil || gotBody.Content.Article == nil {
		t.Fatal("expected article content in request body")
	}
	if gotBody.Content.Article.Source != "https://example.com/blog" {
		t.Errorf("expected article source, got %q", gotBody.Content.Article.Source)
	}
}

func TestCreateCompanyPost_MissingOrganizationID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createCompanyPostAction{conn: conn}

	params, _ := json.Marshal(map[string]string{"text": "Hello"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linkedin.create_company_post",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing organization_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCreateCompanyPost_MissingText(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createCompanyPostAction{conn: conn}

	params, _ := json.Marshal(map[string]string{"organization_id": "12345"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linkedin.create_company_post",
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

func TestCreateCompanyPost_InvalidVisibility(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createCompanyPostAction{conn: conn}

	params, _ := json.Marshal(createCompanyPostParams{
		OrganizationID: "12345",
		Text:           "Hello",
		Visibility:     "CONNECTIONS",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linkedin.create_company_post",
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

func TestCreateCompanyPost_InvalidArticleURL(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createCompanyPostAction{conn: conn}

	params, _ := json.Marshal(createCompanyPostParams{
		OrganizationID: "12345",
		Text:           "Check this out",
		ArticleURL:     "not a url",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linkedin.create_company_post",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid article_url")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCreateCompanyPost_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createCompanyPostAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "linkedin.create_company_post",
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
