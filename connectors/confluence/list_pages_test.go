package confluence

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestListPages_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/pages" {
			t.Errorf("expected path /pages, got %s", r.URL.Path)
		}
		if got := r.URL.Query().Get("space-id"); got != "65538" {
			t.Errorf("expected space-id=65538, got %q", got)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listPagesResponse{
			Results: []pageListItem{
				{
					ID:      "123456",
					Title:   "Getting Started",
					Status:  "current",
					SpaceID: "65538",
				},
				{
					ID:      "123457",
					Title:   "API Reference",
					Status:  "current",
					SpaceID: "65538",
				},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &listPagesAction{conn: conn}

	params, _ := json.Marshal(listPagesParams{
		SpaceID: "65538",
		Limit:   10,
	})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "confluence.list_pages",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data listPagesResponse
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if len(data.Results) != 2 {
		t.Fatalf("expected 2 pages, got %d", len(data.Results))
	}
	if data.Results[0].Title != "Getting Started" {
		t.Errorf("expected first page title 'Getting Started', got %q", data.Results[0].Title)
	}
}

func TestListPages_MissingSpaceID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &listPagesAction{conn: conn}

	params, _ := json.Marshal(map[string]int{"limit": 10})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "confluence.list_pages",
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
