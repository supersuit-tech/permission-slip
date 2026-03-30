package intercom

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestCreateArticle_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/articles" {
			t.Errorf("expected path /articles, got %s", r.URL.Path)
		}

		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatalf("failed to decode request: %v", err)
		}
		if body["state"] != "draft" {
			t.Errorf("expected state draft, got %v", body["state"])
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(intercomArticle{
			Type:  "article",
			ID:    "art_001",
			Title: "Getting Started",
			State: "draft",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &createArticleAction{conn: conn}

	params, _ := json.Marshal(createArticleParams{
		Title:    "Getting Started",
		Body:     "<p>Welcome!</p>",
		AuthorID: 12345,
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "intercom.create_article",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data intercomArticle
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if data.ID != "art_001" {
		t.Errorf("expected id art_001, got %q", data.ID)
	}
}

func TestCreateArticle_MissingTitle(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createArticleAction{conn: conn}

	params, _ := json.Marshal(createArticleParams{AuthorID: 12345})
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "intercom.create_article",
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

func TestCreateArticle_InvalidState(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createArticleAction{conn: conn}

	params, _ := json.Marshal(createArticleParams{
		Title:    "Test",
		AuthorID: 12345,
		State:    "archived",
	})
	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "intercom.create_article",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for invalid state")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}
