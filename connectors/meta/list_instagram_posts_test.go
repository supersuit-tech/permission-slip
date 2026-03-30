package meta

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestListInstagramPosts_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/17841400123/media" {
			t.Errorf("expected path /17841400123/media, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listInstagramPostsResponse{
			Data: []instagramPost{
				{
					ID:            "17855590000000001",
					Caption:       "Hello Instagram!",
					MediaType:     "IMAGE",
					MediaURL:      "https://cdn.instagram.com/image1.jpg",
					Permalink:     "https://www.instagram.com/p/abc123/",
					Timestamp:     "2024-01-15T12:00:00+0000",
					LikeCount:     42,
					CommentsCount: 5,
				},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &listInstagramPostsAction{conn: conn}

	params, _ := json.Marshal(listInstagramPostsParams{
		InstagramAccountID: "17841400123",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "meta.list_instagram_posts",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data listInstagramPostsResponse
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if len(data.Data) != 1 {
		t.Fatalf("expected 1 post, got %d", len(data.Data))
	}
	if data.Data[0].Caption != "Hello Instagram!" {
		t.Errorf("expected caption 'Hello Instagram!', got %q", data.Data[0].Caption)
	}
}

func TestListInstagramPosts_MissingAccountID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &listInstagramPostsAction{conn: conn}

	params, _ := json.Marshal(map[string]int{"limit": 5})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "meta.list_instagram_posts",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Fatalf("expected ValidationError, got %T", err)
	}
}

func TestListInstagramPosts_InvalidLimit(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &listInstagramPostsAction{conn: conn}

	params, _ := json.Marshal(map[string]interface{}{
		"instagram_account_id": "17841400123",
		"limit":                200,
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "meta.list_instagram_posts",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected validation error for limit > 100, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Fatalf("expected ValidationError, got %T", err)
	}
}
