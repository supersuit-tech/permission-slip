package figma

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/supersuit-tech/permission-slip/connectors"
)

func TestListProjects_Success(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Errorf("expected GET, got %s", r.Method)
		}
		if r.URL.Path != "/teams/1234567890/projects" {
			t.Errorf("expected path /teams/1234567890/projects, got %s", r.URL.Path)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(listProjectsResponse{
			Name: "Acme Design Team",
			Projects: []figmaProject{
				{ID: "proj-1", Name: "Mobile App"},
				{ID: "proj-2", Name: "Web Dashboard"},
			},
		})
	}))
	defer srv.Close()

	conn := newForTest(srv.Client(), srv.URL)
	action := &listProjectsAction{conn: conn}

	params, _ := json.Marshal(listProjectsParams{TeamID: "1234567890"})

	result, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "figma.list_projects",
		Parameters:  params,
		Credentials: validCreds(),
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var data listProjectsResponse
	if err := json.Unmarshal(result.Data, &data); err != nil {
		t.Fatalf("failed to unmarshal result: %v", err)
	}
	if len(data.Projects) != 2 {
		t.Fatalf("expected 2 projects, got %d", len(data.Projects))
	}
	if data.Projects[0].Name != "Mobile App" {
		t.Errorf("expected first project 'Mobile App', got %q", data.Projects[0].Name)
	}
}

func TestListProjects_MissingTeamID(t *testing.T) {
	t.Parallel()

	conn := New()
	action := &listProjectsAction{conn: conn}

	params, _ := json.Marshal(map[string]string{})

	_, err := action.Execute(t.Context(), connectors.ActionRequest{
		ActionType:  "figma.list_projects",
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
