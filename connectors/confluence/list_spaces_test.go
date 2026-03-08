package confluence

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip-web/connectors"
)

func TestListSpaces_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/spaces" {
			t.Errorf("expected path /spaces, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listSpacesResponse{
			Results: []spaceItem{
				{
					ID:     "65538",
					Key:    "DEV",
					Name:   "Developer Docs",
					Type:   "global",
					Status: "current",
				},
				{
					ID:     "65539",
					Key:    "OPS",
					Name:   "Operations",
					Type:   "global",
					Status: "current",
				},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &listSpacesAction{conn: conn}

	params, _ := json.Marshal(listSpacesParams{Limit: 10})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "confluence.list_spaces",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data listSpacesResponse
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if len(data.Results) != 2 {
		t.Fatalf("expected 2 spaces, got %d", len(data.Results))
	}
	if data.Results[0].Key != "DEV" {
		t.Errorf("expected first space key 'DEV', got %q", data.Results[0].Key)
	}
}

func TestListSpaces_InvalidStatus(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &listSpacesAction{conn: conn}

	params, _ := json.Marshal(map[string]string{"status": "deleted"})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "confluence.list_spaces",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected validation error for invalid status, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Fatalf("expected ValidationError, got %T", err)
	}
}

func TestListSpaces_InvalidJSON(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &listSpacesAction{conn: conn}

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "confluence.list_spaces",
		Parameters:  []byte(`{invalid`),
		Credentials: validCreds(),
	})
	if err == nil {
		t.Fatal("expected validation error for invalid JSON, got nil")
	}
	if !connectors.IsValidationError(err) {
		t.Fatalf("expected ValidationError, got %T", err)
	}
}
