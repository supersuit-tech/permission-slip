package meta

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestCreateInstagramStory_Success(t *testing.T) {
	t.Parallel()

	callCount := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/17841400123/media":
			var body createStoryContainerRequest
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if body.MediaType != "STORIES" {
				t.Errorf("expected media_type STORIES, got %q", body.MediaType)
			}
			json.NewEncoder(w).Encode(createContainerResponse{ID: "container-123"})
		case r.Method == http.MethodGet && r.URL.Path == "/container-123":
			json.NewEncoder(w).Encode(containerStatusResponse{StatusCode: "FINISHED"})
		case r.Method == http.MethodPost && r.URL.Path == "/17841400123/media_publish":
			json.NewEncoder(w).Encode(publishResponse{ID: "story-456"})
		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			http.Error(w, "unexpected", http.StatusBadRequest)
		}
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &createInstagramStoryAction{conn: conn}

	params, _ := json.Marshal(createInstagramStoryParams{
		InstagramAccountID: "17841400123",
		ImageURL:           "https://example.com/story.jpg",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "meta.create_instagram_story",
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
	if data["id"] != "story-456" {
		t.Errorf("expected id 'story-456', got %q", data["id"])
	}
	if data["container_id"] != "container-123" {
		t.Errorf("expected container_id 'container-123', got %q", data["container_id"])
	}
}

func TestCreateInstagramStory_MissingAccountID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createInstagramStoryAction{conn: conn}

	params, _ := json.Marshal(map[string]string{"image_url": "https://example.com/story.jpg"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "meta.create_instagram_story",
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

func TestCreateInstagramStory_NonHTTPSImageURL(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createInstagramStoryAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"instagram_account_id": "17841400123",
		"image_url":            "http://example.com/story.jpg",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "meta.create_instagram_story",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected validation error for non-HTTPS URL, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Fatalf("expected ValidationError, got %T", err)
	}
}
