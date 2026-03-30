package meta

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestListPagePosts_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/page_123/posts" {
			t.Errorf("expected path /page_123/posts, got %s", r.URL.Path)
		}
		if r.URL.Query().Get("limit") != "5" {
			t.Errorf("expected limit=5, got %q", r.URL.Query().Get("limit"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listPagePostsResponse{
			Data: []pagePost{
				{
					ID:          "page_123_post_1",
					Message:     "Hello world",
					CreatedTime: "2026-03-07T12:00:00+0000",
				},
				{
					ID:          "page_123_post_2",
					Message:     "Another post",
					CreatedTime: "2026-03-06T12:00:00+0000",
				},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &listPagePostsAction{conn: conn}

	params, _ := json.Marshal(listPagePostsParams{
		PageID: "page_123",
		Limit:  5,
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "meta.list_page_posts",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var resp listPagePostsResponse
	if err := json.Unmarshal(result.Data, &resp); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if len(resp.Data) != 2 {
		t.Fatalf("expected 2 posts, got %d", len(resp.Data))
	}
	if resp.Data[0].ID != "page_123_post_1" {
		t.Errorf("expected first post ID 'page_123_post_1', got %q", resp.Data[0].ID)
	}
}

func TestListPagePosts_WithTimeRange(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("since") != "1709251200" {
			t.Errorf("expected since=1709251200, got %q", r.URL.Query().Get("since"))
		}
		if r.URL.Query().Get("until") != "1709337600" {
			t.Errorf("expected until=1709337600, got %q", r.URL.Query().Get("until"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listPagePostsResponse{Data: []pagePost{}})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &listPagePostsAction{conn: conn}

	params, _ := json.Marshal(listPagePostsParams{
		PageID: "page_123",
		Since:  "1709251200",
		Until:  "1709337600",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "meta.list_page_posts",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListPagePosts_DefaultLimit(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("limit") != "10" {
			t.Errorf("expected default limit=10, got %q", r.URL.Query().Get("limit"))
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listPagePostsResponse{Data: []pagePost{}})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &listPagePostsAction{conn: conn}

	params, _ := json.Marshal(listPagePostsParams{PageID: "page_123"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "meta.list_page_posts",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListPagePosts_MissingPageID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &listPagePostsAction{conn: conn}

	params, _ := json.Marshal(map[string]string{})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "meta.list_page_posts",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing page_id")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestListPagePosts_AuthFailure(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"message": "Requires pages_read_engagement permission",
				"type":    "OAuthException",
				"code":    200,
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &listPagePostsAction{conn: conn}

	params, _ := json.Marshal(listPagePostsParams{PageID: "page_123"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "meta.list_page_posts",
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
