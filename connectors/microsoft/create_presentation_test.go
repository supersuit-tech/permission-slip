package microsoft

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestCreatePresentation_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if got := r.URL.Path; got != "/me/drive/root:/Quarterly Report.pptx:/content" {
			t.Errorf("expected path /me/drive/root:/Quarterly Report.pptx:/content, got %s", got)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-access-token-123" {
			t.Errorf("expected Bearer token, got %q", got)
		}
		if got := r.Header.Get("Content-Type"); got != "application/vnd.openxmlformats-officedocument.presentationml.presentation" {
			t.Errorf("expected PPTX content type, got %q", got)
		}

		body, _ := io.ReadAll(r.Body)
		if len(body) == 0 {
			t.Error("expected non-empty request body with PPTX bytes")
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":     "item-abc-123",
			"name":   "Quarterly Report.pptx",
			"webUrl": "https://onedrive.live.com/edit.aspx?id=item-abc-123",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &createPresentationAction{conn: conn}

	params, _ := json.Marshal(createPresentationParams{
		Filename: "Quarterly Report",
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.create_presentation",
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
	if data["item_id"] != "item-abc-123" {
		t.Errorf("expected item_id 'item-abc-123', got %q", data["item_id"])
	}
	if data["name"] != "Quarterly Report.pptx" {
		t.Errorf("expected name 'Quarterly Report.pptx', got %q", data["name"])
	}
	if data["web_url"] != "https://onedrive.live.com/edit.aspx?id=item-abc-123" {
		t.Errorf("expected web_url, got %q", data["web_url"])
	}
	if data["folder_path"] != "/" {
		t.Errorf("expected folder_path '/', got %q", data["folder_path"])
	}
}

func TestCreatePresentation_WithFolderPath(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Path; got != "/me/drive/root:/Documents/Presentations/deck.pptx:/content" {
			t.Errorf("expected path with folder, got %s", got)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":     "item-456",
			"name":   "deck.pptx",
			"webUrl": "https://onedrive.live.com/edit.aspx?id=item-456",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &createPresentationAction{conn: conn}

	params, _ := json.Marshal(createPresentationParams{
		Filename:   "deck.pptx",
		FolderPath: "Documents/Presentations",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.create_presentation",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreatePresentation_AppendsPptxExtension(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got := r.URL.Path; got != "/me/drive/root:/My Slides.pptx:/content" {
			t.Errorf("expected .pptx appended, got path %s", got)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"id":     "item-789",
			"name":   "My Slides.pptx",
			"webUrl": "https://onedrive.live.com/edit.aspx?id=item-789",
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &createPresentationAction{conn: conn}

	params, _ := json.Marshal(createPresentationParams{
		Filename: "My Slides",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.create_presentation",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCreatePresentation_MissingFilename(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createPresentationAction{conn: conn}

	params, _ := json.Marshal(createPresentationParams{})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.create_presentation",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for missing filename")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCreatePresentation_FilenamePathTraversal(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createPresentationAction{conn: conn}

	params, _ := json.Marshal(createPresentationParams{
		Filename: "../../evil.pptx",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.create_presentation",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for path traversal in filename")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCreatePresentation_FolderPathTraversal(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createPresentationAction{conn: conn}

	params, _ := json.Marshal(createPresentationParams{
		Filename:   "deck.pptx",
		FolderPath: "../../../etc",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.create_presentation",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for path traversal in folder_path")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCreatePresentation_FolderPathSpecialChars(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createPresentationAction{conn: conn}

	params, _ := json.Marshal(createPresentationParams{
		Filename:   "deck.pptx",
		FolderPath: "folder?query=inject",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.create_presentation",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for special chars in folder_path")
	}
	if !connectors.IsValidationError(err) {
		t.Errorf("expected ValidationError, got: %T", err)
	}
}

func TestCreatePresentation_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &createPresentationAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.create_presentation",
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

func TestCreatePresentation_AuthFailure(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]string{
				"code":    "InvalidAuthenticationToken",
				"message": "Access token has expired",
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &createPresentationAction{conn: conn}

	params, _ := json.Marshal(createPresentationParams{
		Filename: "test.pptx",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.create_presentation",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected error for auth failure")
	}
	if !connectors.IsAuthError(err) {
		t.Errorf("expected AuthError, got: %T", err)
	}
}

func TestCreatePresentation_RateLimit(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Retry-After", "60")
		w.WriteHeader(http.StatusTooManyRequests)
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &createPresentationAction{conn: conn}

	params, _ := json.Marshal(createPresentationParams{
		Filename: "test.pptx",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.create_presentation",
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
