package meta

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestCreateInstagramPost_Success(t *testing.T) {
	t.Parallel()

	var pollCount atomic.Int32

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		// Step 1: Create container
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/media"):
			if r.URL.Path != "/ig_123/media" {
				t.Errorf("expected path /ig_123/media, got %s", r.URL.Path)
			}
			var body createContainerRequest
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if body.ImageURL != "https://example.com/photo.jpg" {
				t.Errorf("expected image_url, got %q", body.ImageURL)
			}
			if body.Caption != "Check this out! #travel #photo" {
				t.Errorf("expected caption with hashtags, got %q", body.Caption)
			}
			json.NewEncoder(w).Encode(createContainerResponse{ID: "container_456"})

		// Step 2: Poll status
		case r.Method == http.MethodGet && strings.Contains(r.URL.Path, "container_456"):
			pollCount.Add(1)
			json.NewEncoder(w).Encode(containerStatusResponse{StatusCode: "FINISHED"})

		// Step 3: Publish
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/media_publish"):
			if r.URL.Path != "/ig_123/media_publish" {
				t.Errorf("expected path /ig_123/media_publish, got %s", r.URL.Path)
			}
			var body publishRequest
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			if body.CreationID != "container_456" {
				t.Errorf("expected creation_id 'container_456', got %q", body.CreationID)
			}
			json.NewEncoder(w).Encode(publishResponse{ID: "media_789"})

		default:
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &createInstagramPostAction{conn: conn}

	params, _ := json.Marshal(createInstagramPostParams{
		InstagramAccountID: "ig_123",
		ImageURL:           "https://example.com/photo.jpg",
		Caption:            "Check this out!",
		Hashtags:           "#travel #photo",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "meta.create_instagram_post",
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
	if data["id"] != "media_789" {
		t.Errorf("expected id 'media_789', got %q", data["id"])
	}
	if data["container_id"] != "container_456" {
		t.Errorf("expected container_id 'container_456', got %q", data["container_id"])
	}
	if pollCount.Load() < 1 {
		t.Error("expected at least one status poll")
	}
}

func TestCreateInstagramPost_ContainerError(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		switch {
		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/media"):
			json.NewEncoder(w).Encode(createContainerResponse{ID: "container_err"})
		case r.Method == http.MethodGet:
			json.NewEncoder(w).Encode(containerStatusResponse{StatusCode: "ERROR"})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &createInstagramPostAction{conn: conn}

	params, _ := json.Marshal(createInstagramPostParams{
		InstagramAccountID: "ig_123",
		ImageURL:           "https://example.com/photo.jpg",
		Caption:            "Test",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "meta.create_instagram_post",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for container processing failure")
	}
	if !connectors.IsExternalError(err) {
		t.Errorf("expected ExternalError, got: %T (%v)", err, err)
	}
}

func TestCreateInstagramPost_MissingImageURL(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createInstagramPostAction{conn: conn}

	params, _ := json.Marshal(map[string]string{
		"instagram_account_id": "ig_123",
		"caption":              "Hello",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "meta.create_instagram_post",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing image_url")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCreateInstagramPost_NonHTTPSImageURL(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createInstagramPostAction{conn: conn}

	params, _ := json.Marshal(createInstagramPostParams{
		InstagramAccountID: "ig_123",
		ImageURL:           "http://example.com/photo.jpg",
		Caption:            "Hello",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "meta.create_instagram_post",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for non-HTTPS image_url")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCreateInstagramPost_CaptionTooLong(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createInstagramPostAction{conn: conn}

	params, _ := json.Marshal(createInstagramPostParams{
		InstagramAccountID: "ig_123",
		ImageURL:           "https://example.com/photo.jpg",
		Caption:            strings.Repeat("a", maxCaptionLength+1),
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "meta.create_instagram_post",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for caption too long")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}
