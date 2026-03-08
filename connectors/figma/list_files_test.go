package figma

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestListFiles_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/projects/proj-42/files" {
			t.Errorf("expected path /projects/proj-42/files, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listFilesResponse{
			Name: "Mobile App",
			Files: []figmaFile{
				{
					Key:          "abc123DEF",
					Name:         "iOS Screens",
					LastModified: "2024-01-15T12:00:00Z",
				},
				{
					Key:          "xyz789GHI",
					Name:         "Android Screens",
					LastModified: "2024-01-14T10:00:00Z",
				},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &listFilesAction{conn: conn}

	params, _ := json.Marshal(listFilesParams{ProjectID: "proj-42"})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "figma.list_files",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data listFilesResponse
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if len(data.Files) != 2 {
		t.Fatalf("expected 2 files, got %d", len(data.Files))
	}
	if data.Files[0].Name != "iOS Screens" {
		t.Errorf("expected first file 'iOS Screens', got %q", data.Files[0].Name)
	}
}

func TestListFiles_MissingProjectID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &listFilesAction{conn: conn}

	params, _ := json.Marshal(map[string]string{})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "figma.list_files",
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
