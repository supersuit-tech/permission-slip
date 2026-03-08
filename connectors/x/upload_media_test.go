package x

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestUploadMedia_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("method = %s, want POST", r.Method)
		}
		if r.URL.Path != "/media/upload.json" {
			t.Errorf("path = %s, want /media/upload.json", r.URL.Path)
		}
		if r.Header.Get("Content-Type") != "application/x-www-form-urlencoded" {
			t.Errorf("Content-Type = %s, want application/x-www-form-urlencoded", r.Header.Get("Content-Type"))
		}
		if err := r.ParseForm(); err != nil {
			t.Fatalf("ParseForm: %v", err)
		}
		if r.FormValue("media_data") == "" {
			t.Error("media_data is empty")
		}

		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]any{
			"media_id":        12345,
			"media_id_string": "12345",
			"size":            1024,
			"expires_after_secs": 86400,
			"image": map[string]any{
				"image_type": "image/jpeg",
				"w":          100,
				"h":          100,
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["x.upload_media"]

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "x.upload_media",
		Parameters:  json.RawMessage(`{"media_data":"/9j/base64encodedimage==","media_category":"tweet_image"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}

	var data map[string]any
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("unmarshaling result: %v", err)
	}
	if data["media_id_string"] != "12345" {
		t.Errorf("media_id_string = %v, want 12345", data["media_id_string"])
	}
}

func TestUploadMedia_MissingMediaData(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["x.upload_media"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "x.upload_media",
		Parameters:  json.RawMessage(`{}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestUploadMedia_InvalidCategory(t *testing.T) {
	t.Parallel()

	conn := New()
	action := conn.Actions()["x.upload_media"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "x.upload_media",
		Parameters:  json.RawMessage(`{"media_data":"abc","media_category":"invalid_type"}`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("Execute() expected error, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestUploadMedia_WithAltText(t *testing.T) {
	t.Parallel()

	altTextSet := false
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/media/upload.json" {
			w.WriteHeader(http.StatusOK)
			json.NewEncoder(w).Encode(map[string]any{
				"media_id":        99,
				"media_id_string": "99",
				"size":            512,
			})
			return
		}
		if r.URL.Path == "/media/metadata/create.json" {
			altTextSet = true
			w.WriteHeader(http.StatusNoContent)
			return
		}
		t.Errorf("unexpected path: %s", r.URL.Path)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := conn.Actions()["x.upload_media"]

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "x.upload_media",
		Parameters:  json.RawMessage(`{"media_data":"abc","alt_text":"A test image"}`),
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("Execute() unexpected error: %v", err)
	}
	if !altTextSet {
		t.Error("expected alt text metadata endpoint to be called")
	}
}
