package microsoft

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestListPresentations_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer test-access-token-123" {
			t.Errorf("expected Bearer token, got %q", got)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"value": []map[string]any{
				{
					"id":                   "item-1",
					"name":                 "Q4 Review.pptx",
					"webUrl":               "https://onedrive.live.com/edit.aspx?id=item-1",
					"size":                 1048576,
					"lastModifiedDateTime": "2024-03-15T14:30:00Z",
				},
				{
					"id":                   "item-2",
					"name":                 "Strategy.pptx",
					"webUrl":               "https://onedrive.live.com/edit.aspx?id=item-2",
					"size":                 524288,
					"lastModifiedDateTime": "2024-03-10T09:00:00Z",
				},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &listPresentationsAction{conn: conn}

	params, _ := json.Marshal(listPresentationsParams{})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.list_presentations",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var summaries []presentationSummary
	if err := json.Unmarshal(result.Data, &summaries); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if len(summaries) != 2 {
		t.Fatalf("expected 2 presentations, got %d", len(summaries))
	}
	if summaries[0].ItemID != "item-1" {
		t.Errorf("expected item_id 'item-1', got %q", summaries[0].ItemID)
	}
	if summaries[0].Name != "Q4 Review.pptx" {
		t.Errorf("expected name 'Q4 Review.pptx', got %q", summaries[0].Name)
	}
	if summaries[0].Size != 1048576 {
		t.Errorf("expected size 1048576, got %d", summaries[0].Size)
	}
}

func TestListPresentations_WithFolderPath(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify the path includes the folder.
		if got := r.URL.Path; got != "/me/drive/root:/Documents/Decks:/search(q='.pptx')" {
			t.Errorf("expected folder-scoped search path, got %s", got)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"value": []map[string]any{},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &listPresentationsAction{conn: conn}

	params, _ := json.Marshal(listPresentationsParams{
		FolderPath: "Documents/Decks",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.list_presentations",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestListPresentations_FiltersNonPptx(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{
			"value": []map[string]any{
				{
					"id":                   "item-1",
					"name":                 "Slides.pptx",
					"webUrl":               "https://example.com/1",
					"lastModifiedDateTime": "2024-01-01T00:00:00Z",
				},
				{
					"id":                   "item-2",
					"name":                 "notes-about-pptx.docx",
					"webUrl":               "https://example.com/2",
					"lastModifiedDateTime": "2024-01-01T00:00:00Z",
				},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &listPresentationsAction{conn: conn}

	params, _ := json.Marshal(listPresentationsParams{})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.list_presentations",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var summaries []presentationSummary
	if err := json.Unmarshal(result.Data, &summaries); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if len(summaries) != 1 {
		t.Fatalf("expected 1 presentation (non-pptx filtered out), got %d", len(summaries))
	}
	if summaries[0].Name != "Slides.pptx" {
		t.Errorf("expected 'Slides.pptx', got %q", summaries[0].Name)
	}
}

func TestListPresentations_DefaultParams(t *testing.T) {
	t.Parallel()

	var params listPresentationsParams
	params.defaults()
	if params.Top != 10 {
		t.Errorf("expected default top 10, got %d", params.Top)
	}
}

func TestListPresentations_TopClamped(t *testing.T) {
	t.Parallel()

	params := listPresentationsParams{Top: 100}
	params.defaults()
	if params.Top != 50 {
		t.Errorf("expected top clamped to 50, got %d", params.Top)
	}
}

func TestListPresentations_FolderPathTraversal(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &listPresentationsAction{conn: conn}

	params, _ := json.Marshal(listPresentationsParams{
		FolderPath: "../../secrets",
	})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.list_presentations",
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

func TestListPresentations_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &listPresentationsAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "microsoft.list_presentations",
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
